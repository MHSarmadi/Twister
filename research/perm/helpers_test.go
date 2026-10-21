package perm

import (
	"math/bits"
	"math/rand"
	"os"

	"github.com/MHSarmadi/Twister"
)

func rngSeed(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

func randState(r *rand.Rand) Twister.State1024 {
	var s Twister.State1024
	for i := range s {
		s[i] = r.Uint64()
	}
	return s
}

func hw(s Twister.State1024) int {
	n := 0
	for _, w := range s {
		n += bits.OnesCount64(w)
	}
	return n
}

func xor(a, b Twister.State1024) Twister.State1024 {
	var c Twister.State1024
	for i := range c {
		c[i] = a[i] ^ b[i]
	}
	return c
}

func eq(a, b Twister.State1024) bool { return a == b }

func zero() Twister.State1024 { return Twister.State1024{} }

func flip(s *Twister.State1024, bit int) {
	s[bit/64] ^= uint64(1) << uint(bit%64)
}

func getBit(s Twister.State1024, bit int) uint64 {
	return (s[bit/64] >> uint(bit%64)) & 1
}

func rotlState(s Twister.State1024, r int) Twister.State1024 {
	var out Twister.State1024
	for i := 0; i < 16; i++ {
		out[i] = bits.RotateLeft64(s[i], r)
	}
	return out
}

func parity(s Twister.State1024, mask Twister.State1024) int {
	n := 0
	for i := 0; i < 16; i++ {
		n += bits.OnesCount64(s[i] & mask[i])
	}
	return n & 1
}

func sampleBits(r *rand.Rand, total, n int) []int {
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

// fastMode shrinks every heavy test when PERM_FAST is set.
func fastMode() bool { return os.Getenv("PERM_FAST") != "" }
