package Twister

import "math/bits"

type State512 [16]uint32

func NewState512_32bitSeed(seed uint32) State512 {
	result := State512{}
	result[0] = seed

	result.Permute()

	return result
}

func NewState512_128bitSeed(seed [4]uint32) State512 {
	s := State512{}
	indices := [4]int{0x0, 0xd, 0xe, 0x3}

	for i := 0; i < 4; i++ {
		s[indices[i]] = seed[i]
	}

	s.Permute()
	return s
}

// First number gets combined with a, and the last number getting combined is also a.
func (s *State512) quarter_ab2ab(i [4]int) {
	var x, y uint32
	a, b, c, d := s[i[0]], s[i[1]], s[i[2]], s[i[3]]

	// Injecting "Twister! Sarmadi"
	a ^= c0_hi
	b ^= c0_lo
	c ^= c1_hi
	d ^= c1_lo

	// Pre-round mixture
	mixture := a ^ b ^ c ^ d
	a, b, c, d = mixture^d, mixture^c, mixture^b, mixture^a

	// Sixteenth round 1
	x, y = bits.Mul32(a, b)

	a = bits.RotateLeft32(a, 16)
	b = bits.RotateLeft32(b, 16)
	c ^= x
	d ^= y

	// Sixteenth round 2
	x, y = bits.Mul32(b, c)

	a ^= x
	c = bits.RotateLeft32(c, 16)
	d += y

	// Sixteenth round 3
	x, y = bits.Mul32(a, d)

	b ^= x
	c += y
	d = bits.RotateLeft32(d, 16)

	// Sixteenth round 4
	x, y = bits.Mul32(c, d)

	a += x
	b += y

	// Post-round mixture
	mixture = a + b + c + d
	s[i[0]], s[i[1]], s[i[2]], s[i[3]] = mixture-d, mixture-c, mixture-b, mixture-a
}

// Independent starters: 04 9d e2 f3
// Total Similarities to Rdiag2Row: 04d2f3 (75%)
func (s *State512) permute_col2rdiag() {
	// col 0, a2a
	s.quarter_ab2ab([4]int{0x0, 0x4, 0x8, 0xc})

	// col 1, c2c
	s.quarter_ab2ab([4]int{0x9, 0xd, 0x1, 0x5})

	// col 2, d2d
	s.quarter_ab2ab([4]int{0xe, 0x2, 0x6, 0xa})

	// col 3, d2d
	s.quarter_ab2ab([4]int{0xf, 0x3, 0x7, 0xb})
}

// Independent starters: 0d b4 f2 63
// Total Similarities to Row2Ldiag: 0db4f63 (87.5%)
func (s *State512) permute_rdiag2row() {
	// rdiag 0, a2a
	s.quarter_ab2ab([4]int{0x0, 0xd, 0x7, 0xa})

	// rdiag 1, c2c
	s.quarter_ab2ab([4]int{0xb, 0x4, 0xe, 0x1})

	// rdiag 2, d2d
	s.quarter_ab2ab([4]int{0xf, 0x2, 0x5, 0x8})

	// rdiag 3, b2b
	s.quarter_ab2ab([4]int{0x6, 0x3, 0x9, 0xc})
}

// Independent starters: 03 46 b9 df
// Total Similarities to Ldiag2Col: 046b9df (87.5%)
func (s *State512) permute_row2ldiag() {
	// row 0, a2a
	s.quarter_ab2ab([4]int{0x0, 0x3, 0x1, 0x2})

	// row 1, a2a
	s.quarter_ab2ab([4]int{0x4, 0x6, 0x5, 0x7})

	// row 2, d2d
	s.quarter_ab2ab([4]int{0xb, 0x9, 0x8, 0xa})

	// row 3, b2b
	s.quarter_ab2ab([4]int{0xd, 0xf, 0xe, 0xc})
}

// Independent starters: 0f b6 2d 94
// Total Similarities to Col2Rdiag: 0fb2d94 (87.5%)
func (s *State512) permute_ldiag2col() {
	// ldiag 0, a2a
	s.quarter_ab2ab([4]int{0x0, 0xf, 0x5, 0xa})

	// ldiag 1, c2c
	s.quarter_ab2ab([4]int{0xb, 0x6, 0xc, 0x1})

	// ldiag 2, a2a
	s.quarter_ab2ab([4]int{0x2, 0xd, 0x7, 0x8})

	// ldiag 3, c2c
	s.quarter_ab2ab([4]int{0x9, 0x4, 0xe, 0x3})
}

func (s *State512) Permute() {
	const rounds = 4 // 4 * 4 * 4 == 64 overall

	for range rounds {
		s.permute_col2rdiag()

		s.permute_rdiag2row()

		s.permute_row2ldiag()

		s.permute_ldiag2col()
	}
}

func (s *State512) Wipe() {
	for i := range s {
		s[i] = 0
	}
}

// "Twister! Sarmadi"
const c0_lo, c0_hi uint32 = 0x54776973, 0x74657221
const c1_lo, c1_hi uint32 = 0x20536172, 0x6d616469
