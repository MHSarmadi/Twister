package Twister

type State8192 [16]State512

// WARNING: ONLY WHEN YOU ACTUALLY NEED SUCH HUGE STATE. MUCH SLOWER PERMUTATION.
func NewState8192_32bitSeed(seed uint32) State8192 {
	result := State8192{}
	result[0][0] = seed

	result[0].permute(1)
	result.Permute()

	return result
}

// WARNING: ONLY WHEN YOU ACTUALLY NEED SUCH HUGE STATE. MUCH SLOWER PERMUTATION.
func NewState8192_512bitSeed(seed State512) State8192 {
	s := State8192{}
	indices := [4]int{0x0, 0xd, 0xe, 0x3}

	for i := 0; i < 4; i++ {
		s[indices[i]] = NewState512_128bitSeed([4]uint32(seed[i*4 : (i+1)*4]))
	}

	s.Permute()

	return s
}

// First number gets combined with a, and the last number getting combined is also a.
func (s *State8192) quarter_feistel_ab2ab(i [4]int) {
	a, b, c, d := s[i[0]], s[i[1]], s[i[2]], s[i[3]]

	// Injecting "FeistelExpansion"
	a[0] += c0f_hi
	b[1] += c0f_lo
	c[2] += c1f_hi
	d[3] += c1f_lo

	// Sixteenth round 1
	c.xor(a)
	d.xor(b)
	c.permute(1)
	d.permute(1)

	// Sixteenth round 2
	a.xor(d)
	b.xor(c)
	a.permute(1)
	b.permute(1)

	// Sixteenth round 3
	c.add(b)
	d.add(a)
	c.permute(1)
	d.permute(1)

	// Sixteenth round 4
	a.add(c)
	b.add(d)
	a.permute(1)
	b.permute(1)

	s[i[0]], s[i[1]], s[i[2]], s[i[3]] = a, b, c, d
}

// Independent starters: 04 9d e2 f3
// Total Similarities to Rdiag2Row: 04d2f3 (75%)
func (s *State8192) permute_col2rdiag() {
	// col 0, a2a
	s.quarter_feistel_ab2ab([4]int{0x0, 0x4, 0x8, 0xc})

	// col 1, c2c
	s.quarter_feistel_ab2ab([4]int{0x9, 0xd, 0x1, 0x5})

	// col 2, d2d
	s.quarter_feistel_ab2ab([4]int{0xe, 0x2, 0x6, 0xa})

	// col 3, d2d
	s.quarter_feistel_ab2ab([4]int{0xf, 0x3, 0x7, 0xb})
}

// Independent starters: 0d b4 f2 63
// Total Similarities to Row2Ldiag: 0db4f63 (87.5%)
func (s *State8192) permute_rdiag2row() {
	// rdiag 0, a2a
	s.quarter_feistel_ab2ab([4]int{0x0, 0xd, 0x7, 0xa})

	// rdiag 1, c2c
	s.quarter_feistel_ab2ab([4]int{0xb, 0x4, 0xe, 0x1})

	// rdiag 2, d2d
	s.quarter_feistel_ab2ab([4]int{0xf, 0x2, 0x5, 0x8})

	// rdiag 3, b2b
	s.quarter_feistel_ab2ab([4]int{0x6, 0x3, 0x9, 0xc})
}

// Independent starters: 03 46 b9 df
// Total Similarities to Ldiag2Col: 046b9df (87.5%)
func (s *State8192) permute_row2ldiag() {
	// row 0, a2a
	s.quarter_feistel_ab2ab([4]int{0x0, 0x3, 0x1, 0x2})

	// row 1, a2a
	s.quarter_feistel_ab2ab([4]int{0x4, 0x6, 0x5, 0x7})

	// row 2, d2d
	s.quarter_feistel_ab2ab([4]int{0xb, 0x9, 0x8, 0xa})

	// row 3, b2b
	s.quarter_feistel_ab2ab([4]int{0xd, 0xf, 0xe, 0xc})
}

// Independent starters: 0f b6 2d 94
// Total Similarities to Col2Rdiag: 0fb2d94 (87.5%)
func (s *State8192) permute_ldiag2col() {
	// ldiag 0, a2a
	s.quarter_feistel_ab2ab([4]int{0x0, 0xf, 0x5, 0xa})

	// ldiag 1, c2c
	s.quarter_feistel_ab2ab([4]int{0xb, 0x6, 0xc, 0x1})

	// ldiag 2, a2a
	s.quarter_feistel_ab2ab([4]int{0x2, 0xd, 0x7, 0x8})

	// ldiag 3, c2c
	s.quarter_feistel_ab2ab([4]int{0x9, 0x4, 0xe, 0x3})
}

func (s *State8192) Permute() {
	const rounds = 2 // 2 * 4 * 4 * 8 * (1 * 4 * 4) == 4096 overall

	s.permute(rounds)
}

func (s *State8192) permute(rounds int) {
	for range rounds {
		s.permute_col2rdiag()

		s.permute_rdiag2row()

		s.permute_row2ldiag()

		s.permute_ldiag2col()
	}
}

func (s *State8192) xor(x State8192) {
	for i := range 16 {
		s[i].xor(x[i])
	}
}

func (s *State8192) add(x State8192) {
	for i := range 16 {
		s[i].add(x[i])
	}
}

func (s *State8192) Wipe() {
	for i := range s {
		s[i].Wipe()
	}
}

// "FeistelExpansion"
const c0f_hi, c0f_lo uint32 = 0x46656973, 0x74656c45
const c1f_hi, c1f_lo uint32 = 0x7870616e, 0x73696f6e
