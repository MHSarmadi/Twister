package perm

import (
	"math/bits"

	"github.com/MHSarmadi/Twister"
)

// Note: Twister's 512.go declares `c0_lo, c0_hi = 0x54776973, 0x74657221`
// — the "lo"/"hi" labels are swapped relative to the natural 64→32 split.
// The QR uses these exact values. Our ref must match.
const (
	c512_a uint32 = 0x74657221 // XORed into a
	c512_b uint32 = 0x54776973 // XORed into b
	c512_c uint32 = 0x6d616469 // XORed into c
	c512_d uint32 = 0x20536172 // XORed into d
)

// quarter512 mirrors State512.quarter_marx_ab2ab.
func quarter512(s Twister.State512, i [4]int) Twister.State512 {
	a, b, c, d := s[i[0]], s[i[1]], s[i[2]], s[i[3]]

	a ^= c512_a
	b ^= c512_b
	c ^= c512_c
	d ^= c512_d

	mixture := a ^ b ^ c ^ d
	a, b, c, d = mixture^d, mixture^c, mixture^b, mixture^a

	x, y := bits.Mul32(a, b)
	a = bits.RotateLeft32(a, 16)
	b = bits.RotateLeft32(b, 16)
	c ^= x
	d ^= y

	x, y = bits.Mul32(b, c)
	a ^= x
	c = bits.RotateLeft32(c, 16)
	d += y

	x, y = bits.Mul32(a, d)
	b ^= x
	c += y
	d = bits.RotateLeft32(d, 16)

	x, y = bits.Mul32(c, d)
	a += x
	b += y

	mixture = a + b + c + d
	s[i[0]], s[i[1]], s[i[2]], s[i[3]] = mixture-d, mixture-c, mixture-b, mixture-a
	return s
}

func applyQRs512(s Twister.State512, n int) Twister.State512 {
	for i := 0; i < n; i++ {
		s = quarter512(s, qrIndices[i%16])
	}
	return s
}

func applyPermuteCalls512(s Twister.State512, n int) Twister.State512 {
	for i := 0; i < n; i++ {
		for j := (i % 4) * 4; j < (i%4)*4+4; j++ {
			s = quarter512(s, qrIndices[j])
		}
	}
	return s
}

func permuteFull512(s Twister.State512) Twister.State512 {
	return applyPermuteCalls512(s, 16)
}
