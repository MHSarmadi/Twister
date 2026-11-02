// Package feistel verifies the Feistel expansion pattern on a small state.
//
// The point: if 2048 = [16]State128 with this Feistel construction passes the
// same battery State16384 = [16]State1024 will be subjected to, we trust the
// pattern scales. State128 matches mul.State128 exactly — if that type changes,
// this ref must be updated.
package feistel

import "math/bits"

type State128 [4]uint32
type State2048 [16]State128

// Constants matching mul.State128 and 512.go. Same naming quirk as State512:
// "lo"/"hi" labels are swapped relative to natural 32-bit split, values are
// what's actually XORed in.
const (
	c0_lo, c0_hi uint32 = 0x54776973, 0x74657221
	c1_lo, c1_hi uint32 = 0x20536172, 0x6d616469
	c0f_hi, c0f_lo uint32 = 0x46656973, 0x74656c45
	c1f_hi, c1f_lo uint32 = 0x7870616e, 0x73696f6e
)

// MultPermuter mirrors mul.State128.MultPermuter.
func (s *State128) MultPermuter() {
	var x, y uint32
	s[0] ^= c0_hi
	s[1] ^= c0_lo
	s[2] ^= c1_hi
	s[3] ^= c1_lo

	mixture := s[0] ^ s[1] ^ s[2] ^ s[3]
	s[0], s[1], s[2], s[3] = mixture^s[3], mixture^s[2], mixture^s[1], mixture^s[0]

	x, y = bits.Mul32(s[0], s[1])
	s[0] = bits.RotateLeft32(s[0], 16)
	s[1] = bits.RotateLeft32(s[1], 16)
	s[2] ^= x
	s[3] ^= y

	x, y = bits.Mul32(s[1], s[2])
	s[0] ^= x
	s[2] = bits.RotateLeft32(s[2], 16)
	s[3] += y

	x, y = bits.Mul32(s[0], s[3])
	s[1] ^= x
	s[2] += y
	s[3] = bits.RotateLeft32(s[3], 16)

	x, y = bits.Mul32(s[2], s[3])
	s[0] += x
	s[1] += y

	mixture = s[0] + s[1] + s[2] + s[3]
	s[0], s[1], s[2], s[3] = mixture-s[3], mixture-s[2], mixture-s[1], mixture-s[0]
}

func (s *State128) xor(o State128) {
	for i := range s {
		s[i] ^= o[i]
	}
}

var qrIndices = [16][4]int{
	{0x0, 0x4, 0x8, 0xc}, {0x9, 0xd, 0x1, 0x5}, {0xe, 0x2, 0x6, 0xa}, {0xf, 0x3, 0x7, 0xb},
	{0x0, 0xd, 0x7, 0xa}, {0xb, 0x4, 0xe, 0x1}, {0xf, 0x2, 0x5, 0x8}, {0x6, 0x3, 0x9, 0xc},
	{0x0, 0x3, 0x1, 0x2}, {0x4, 0x6, 0x5, 0x7}, {0xb, 0x9, 0x8, 0xa}, {0xd, 0xf, 0xe, 0xc},
	{0x0, 0xf, 0x5, 0xa}, {0xb, 0x6, 0xc, 0x1}, {0x2, 0xd, 0x7, 0x8}, {0x9, 0x4, 0xe, 0x3},
}

// quarterFeistel mirrors State16384/State8192.quarter_feistel_ab2ab (without the
// naming asymmetry that would break at 32-bit constants — matches State8192's
// 4-constant pattern).
func (s *State2048) quarterFeistel(i [4]int) {
	a, b, c, d := s[i[0]], s[i[1]], s[i[2]], s[i[3]]

	a[0] += c0f_hi
	b[1] += c0f_lo
	c[2] += c1f_hi
	d[3] += c1f_lo

	c.xor(a)
	d.xor(b)
	c.MultPermuter()
	d.MultPermuter()

	a.xor(d)
	b.xor(c)
	a.MultPermuter()
	b.MultPermuter()

	c.xor(b)
	d.xor(a)
	c.MultPermuter()
	d.MultPermuter()

	a.xor(c)
	b.xor(d)
	a.MultPermuter()
	b.MultPermuter()

	s[i[0]], s[i[1]], s[i[2]], s[i[3]] = a, b, c, d
}

// Permute applies 4 outer rounds × 4 layers × 4 QRs.
func (s *State2048) Permute() {
	for r := 0; r < 4; r++ {
		for i := 0; i < 16; i++ {
			s.quarterFeistel(qrIndices[i])
		}
	}
}

// applyCalls applies the first n QR-calls (n/4 = layers).
func (s *State2048) applyCalls(n int) State2048 {
	out := *s
	for i := 0; i < n; i++ {
		out.quarterFeistel(qrIndices[i%16])
	}
	return out
}