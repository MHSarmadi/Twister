package Twister

type State16384 [16]State1024

// WARNING: ONLY WHEN YOU ACTUALLY NEED SUCH HUGE STATE. MUCH SLOWER PERMUTATION.
func NewState16384_64bitSeed(seed uint64) State16384 {
	result := State16384{}
	result[0][0] = seed

	result[0].permute(1)
	result.Permute()

	return result
}

// WARNING: ONLY WHEN YOU ACTUALLY NEED SUCH HUGE STATE. MUCH SLOWER PERMUTATION.
func NewState16384_1024bitSeed(seed State1024) State16384 {
	s := State16384{}
	indices := [4]int{0x0, 0xd, 0xe, 0x3}

	for i := 0; i < 4; i++ {
		s[indices[i]] = NewState1024_256bitSeed([4]uint64(seed[i*4 : (i+1)*4]))
	}

	s.Permute()

	return s
}

// First number gets combined with a, and the last number getting combined is also a.
func (s *State16384) quarter_feistel_ab2ab(i [4]int) {
	a, b, c, d := s[i[0]], s[i[1]], s[i[2]], s[i[3]]

	// Injecting "FeistelExpansion"
	a[0] += c0f
	b[1] += c1f
	c[2] += c1f
	d[3] += c0f

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
func (s *State16384) permute_col2rdiag() {
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
func (s *State16384) permute_rdiag2row() {
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
func (s *State16384) permute_row2ldiag() {
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
func (s *State16384) permute_ldiag2col() {
	// ldiag 0, a2a
	s.quarter_feistel_ab2ab([4]int{0x0, 0xf, 0x5, 0xa})

	// ldiag 1, c2c
	s.quarter_feistel_ab2ab([4]int{0xb, 0x6, 0xc, 0x1})

	// ldiag 2, a2a
	s.quarter_feistel_ab2ab([4]int{0x2, 0xd, 0x7, 0x8})

	// ldiag 3, c2c
	s.quarter_feistel_ab2ab([4]int{0x9, 0x4, 0xe, 0x3})
}

func (s *State16384) Permute() {
	const rounds = 2 // 2 * 4 * 4 * 8 * (1 * 4 * 4) == 4096 overall

	s.permute(rounds)
}

func (s *State16384) permute(rounds int) {
	for range rounds {
		s.permute_col2rdiag()

		s.permute_rdiag2row()

		s.permute_row2ldiag()

		s.permute_ldiag2col()
	}
}

func (s *State16384) xor(x State16384) {
	for i := range 16 {
		s[i].xor(x[i])
	}
}

func (s *State16384) add(x State16384) {
	for i := range 16 {
		s[i].add(x[i])
	}
}

func (s *State16384) Wipe() {
	for i := range s {
		s[i].Wipe()
	}
}

// "FeistelExpansion"
const c0f, c1f uint64 = 0x4665697374656c45, 0x7870616e73696f6e
