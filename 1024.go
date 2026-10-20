package Twister

import "math/bits"

type State1024 [16]uint64

// First number gets combined with a, and the last number getting combined is also a.
func (s *State1024) quarter_a2a(rotl_c, i [4]int) {
	a, b, c, d := s[i[0]], s[i[1]], s[i[2]], s[i[3]]

	b += a
	c ^= b
	c = bits.RotateLeft64(c, rotl_c[0])
	d += c
	a ^= d
	a = bits.RotateLeft64(a, rotl_c[1])

	c += a
	d ^= c
	d = bits.RotateLeft64(d, rotl_c[2])
	b += d
	b = bits.RotateLeft64(b, rotl_c[3])
	a ^= b

	s[i[0]], s[i[1]], s[i[2]], s[i[3]] = a, b, c, d
}

// Independent starters: 0, 9, e, f
func (s *State1024) permute_col2rdiag(rotl_c [4]int) {
	// col 0, a2a
	s.quarter_a2a(rotl_c, [4]int{0x0, 0x4, 0x8, 0xc}) // 0 is the a-th in the col

	// col 1, c2c
	s.quarter_a2a(rotl_c, [4]int{0x9, 0xd, 0x1, 0x5}) // 9 is the c-th in the col

	// col 2, d2d
	s.quarter_a2a(rotl_c, [4]int{0xe, 0x2, 0x6, 0xa}) // e is the d-th in the col

	// col 3, d2d
	s.quarter_a2a(rotl_c, [4]int{0xf, 0x3, 0x7, 0xb}) // f is the d-th in the col
}

// Independent starters: 0 b f 6
func (s *State1024) permute_rdiag2row(rotl_c [4]int) {
	// rdiag 0, a2a
	s.quarter_a2a(rotl_c, [4]int{0x0, 0x7, 0xa, 0xd}) // 0 is the a-th in the rdiag

	// rdiag 1, c2c
	s.quarter_a2a(rotl_c, [4]int{0xb, 0xe, 0x1, 0x4}) // b is the c-th in the rdiag

	// rdiag 2, d2d
	s.quarter_a2a(rotl_c, [4]int{0xf, 0x2, 0x5, 0x8}) // f is the d-th in the rdiag

	// rdiag 3, b2b
	s.quarter_a2a(rotl_c, [4]int{0x6, 0x9, 0xc, 0x3}) // 6 is the b-th in the rdiag
}

// Independent starters: 0 4 b d
func (s *State1024) permute_row2ldiag(rotl_c [4]int) {
	// row 0, a2a
	s.quarter_a2a(rotl_c, [4]int{0x0, 0x1, 0x2, 0x3}) // 0 is the a-th in the row

	// row 1, a2a
	s.quarter_a2a(rotl_c, [4]int{0x4, 0x5, 0x6, 0x7}) // 4 is the a-th in the row

	// row 2, d2d
	s.quarter_a2a(rotl_c, [4]int{0xb, 0x8, 0x9, 0xa}) // b is the d-th in the row

	// row 3, b2b
	s.quarter_a2a(rotl_c, [4]int{0xd, 0xe, 0xf, 0xc}) // d is the b-th in the row
}

// Independent starters: 0 b 2 9
func (s *State1024) permute_ldiag2col(rotl_c [4]int) {
	// ldiag 0, a2a
	s.quarter_a2a(rotl_c, [4]int{0x0, 0x5, 0xa, 0xf}) // 0 is the a-th in the ldiag

	// ldiag 1, c2c
	s.quarter_a2a(rotl_c, [4]int{0xb, 0xc, 0x1, 0x6}) // b is the c-th in the ldiag

	// ldiag 2, a2a
	s.quarter_a2a(rotl_c, [4]int{0x2, 0x7, 0x8, 0xd}) // 2 is the a-th in the ldiag

	// ldiag 3, c2c
	s.quarter_a2a(rotl_c, [4]int{0x9, 0xe, 0x3, 0x4}) // 9 is the c-th in the ldiag
}

func (s *State1024) inject(injectibles [4]uint64, i [4]int) {
	for n := range 4 {
		s[i[n]] ^= injectibles[n]
	}
}

func (s *State1024) Permute() {
	const rounds = 5 // 5 * 4 == 20 overall

	for i := range rounds {
		s.inject(injectables[i], [4]int{0x0, 0x9, 0xe, 0xf}) // Col independent starters
		s.permute_col2rdiag(rotl_c[i])

		s.inject(injectables[i], [4]int{0x0, 0xb, 0xf, 0x6}) // Rdiag independent starters
		s.permute_rdiag2row(rotl_c[i])

		s.inject(injectables[i], [4]int{0x0, 0x4, 0xb, 0xd}) // Row independent starters
		s.permute_row2ldiag(rotl_c[i])

		s.inject(injectables[i], [4]int{0x0, 0xb, 0x2, 0x9}) // Ldiag independent starters
		s.permute_ldiag2col(rotl_c[i])
	}
}

func (s *State1024) Wipe() {
	for i := range s {
		s[i] = 0
	}
}

var rotl_c = [5][4]int{
	{5, -13, 16, -29},
	{-13, 16, -29, 5},
	{16, -29, 5, -13},
	{-29, 5, -13, 16},
	{5, -13, 16, -29},
}

// Truly random generated
var injectables = [5][4]uint64{
	{
		0x24690a45a36babab,
		0x5917cb2b135a3977,
		0xd0d2f5d029fef71d,
		0x2611c7c3f2035149,
	},
	{
		0x439e5ce0062265c8,
		0x860a47a159a16ab8,
		0x5e64403011b6419b,
		0x8d8497e54e53c8de,
	},
	{
		0x0011e1fe1f467a11,
		0x51cf1b0c57c37f7d,
		0x950242da495f13fd,
		0x5dd09b2f0ca7d642,
	},
	{
		0x4085225bc9504bd0,
		0xa6d41eec06b3bd18,
		0x4ccf41b4abb8eeec,
		0x8978c36993c7087b,
	},
	{
		0x548660187bf9a40e,
		0x12ac87fdfb2f77a4,
		0xbffe72d36d656785,
		0x18fad47860e3ee86,
	},
}
