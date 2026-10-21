package perm

import (
	"math/bits"

	"github.com/MHSarmadi/Twister"
)

const c0, c1 uint64 = 0x5477697374657221, 0x205361726d616469

// qrIndices is the sequence of 16 QR index tuples applied per super-round.
// Concatenation of the four permute layers, in order:
//
//	[0..3]   permute_col2rdiag
//	[4..7]   permute_rdiag2row
//	[8..11]  permute_row2ldiag
//	[12..15] permute_ldiag2col
var qrIndices = [16][4]int{
	{0x0, 0x4, 0x8, 0xc}, // col 0
	{0x9, 0xd, 0x1, 0x5}, // col 1
	{0xe, 0x2, 0x6, 0xa}, // col 2
	{0xf, 0x3, 0x7, 0xb}, // col 3
	{0x0, 0xd, 0x7, 0xa}, // rdiag 0
	{0xb, 0x4, 0xe, 0x1}, // rdiag 1
	{0xf, 0x2, 0x5, 0x8}, // rdiag 2
	{0x6, 0x3, 0x9, 0xc}, // rdiag 3
	{0x0, 0x3, 0x1, 0x2}, // row 0
	{0x4, 0x6, 0x5, 0x7}, // row 1
	{0xb, 0x9, 0x8, 0xa}, // row 2
	{0xd, 0xf, 0xe, 0xc}, // row 3
	{0x0, 0xf, 0x5, 0xa}, // ldiag 0
	{0xb, 0x6, 0xc, 0x1}, // ldiag 1
	{0x2, 0xd, 0x7, 0x8}, // ldiag 2
	{0x9, 0x4, 0xe, 0x3}, // ldiag 3
}

// quarterAB2AB mirrors State1024.quarter_ab2ab.
func quarterAB2AB(s Twister.State1024, i [4]int) Twister.State1024 {
	a, b, c, d := s[i[0]], s[i[1]], s[i[2]], s[i[3]]

	a ^= c0
	b ^= c1

	mixture := a ^ b ^ c ^ d
	a, b, c, d = mixture^d, mixture^c, mixture^b, mixture^a

	x, y := bits.Mul64(a, b)
	a = bits.RotateLeft64(a, 32)
	b = bits.RotateLeft64(b, 32)
	c ^= x
	d ^= y

	x, y = bits.Mul64(b, c)
	a ^= x
	c = bits.RotateLeft64(c, 32)
	d += y

	x, y = bits.Mul64(a, d)
	b ^= x
	c += y
	d = bits.RotateLeft64(d, 32)

	x, y = bits.Mul64(c, d)
	a += x
	b += y

	mixture = a + b + c + d
	s[i[0]], s[i[1]], s[i[2]], s[i[3]] = mixture-d, mixture-c, mixture-b, mixture-a
	return s
}

func permuteCol2RDiag(s Twister.State1024) Twister.State1024 {
	for i := 0; i < 4; i++ {
		s = quarterAB2AB(s, qrIndices[i])
	}
	return s
}

func permuteRDiag2Row(s Twister.State1024) Twister.State1024 {
	for i := 4; i < 8; i++ {
		s = quarterAB2AB(s, qrIndices[i])
	}
	return s
}

func permuteRow2LDiag(s Twister.State1024) Twister.State1024 {
	for i := 8; i < 12; i++ {
		s = quarterAB2AB(s, qrIndices[i])
	}
	return s
}

func permuteLDiag2Col(s Twister.State1024) Twister.State1024 {
	for i := 12; i < 16; i++ {
		s = quarterAB2AB(s, qrIndices[i])
	}
	return s
}

// applyQRs applies the first n QRs of the permutation, cycling per 16.
// n=4 → col layer only; n=8 → col+rdiag; n=16 → full super-round.
func applyQRs(s Twister.State1024, n int) Twister.State1024 {
	for i := 0; i < n; i++ {
		s = quarterAB2AB(s, qrIndices[i%16])
	}
	return s
}

// applyPermuteCalls applies the first n permute calls (n in [1,16]).
// n=4 → one super-round; n=16 → full Permute().
func applyPermuteCalls(s Twister.State1024, n int) Twister.State1024 {
	for i := 0; i < n; i++ {
		switch i % 4 {
		case 0:
			s = permuteCol2RDiag(s)
		case 1:
			s = permuteRDiag2Row(s)
		case 2:
			s = permuteRow2LDiag(s)
		case 3:
			s = permuteLDiag2Col(s)
		}
	}
	return s
}

func permuteFull(s Twister.State1024) Twister.State1024 {
	return applyPermuteCalls(s, 16)
}
