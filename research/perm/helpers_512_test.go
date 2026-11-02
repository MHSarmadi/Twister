package perm

import (
	"math/bits"
	"math/rand"

	"github.com/MHSarmadi/Twister"
)

func randState512(r *rand.Rand) Twister.State512 {
	var s Twister.State512
	for i := range s {
		s[i] = r.Uint32()
	}
	return s
}

func hw512(s Twister.State512) int {
	n := 0
	for _, w := range s {
		n += bits.OnesCount32(w)
	}
	return n
}

func xor512(a, b Twister.State512) Twister.State512 {
	var c Twister.State512
	for i := range c {
		c[i] = a[i] ^ b[i]
	}
	return c
}

func eq512(a, b Twister.State512) bool { return a == b }

func flip512(s *Twister.State512, bit int) {
	s[bit/32] ^= uint32(1) << uint(bit%32)
}

func rotlState512(s Twister.State512, r int) Twister.State512 {
	var out Twister.State512
	for i := range s {
		out[i] = bits.RotateLeft32(s[i], r)
	}
	return out
}

func parity512(s Twister.State512, mask Twister.State512) int {
	n := 0
	for i := range s {
		n += bits.OnesCount32(s[i] & mask[i])
	}
	return n & 1
}

func sampleBits512(r *rand.Rand, total, n int) []int {
	if n >= total {
		out := make([]int, total)
		for i := range out {
			out[i] = i
		}
		return out
	}
	used := make(map[int]bool, n)
	out := make([]int, 0, n)
	for len(out) < n {
		b := r.Intn(total)
		if !used[b] {
			used[b] = true
			out = append(out, b)
		}
	}
	return out
}
