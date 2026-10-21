package perm

import (
	"testing"

	"github.com/MHSarmadi/Twister"
)

// TestCDConstantFreeFmeasures what actually matters: does a structured
// input with zeros in the a/b positions produce weak avalanche after the
// FULL Permute()? Measured over all 16 words (1024 bits), T=500.
func TestCDConstantFree(t *testing.T) {
	r := rngSeed(0xc0d2)
	const T = 500

	cases := []struct {
		name   string
		makeIn func(x, y uint64) Twister.State1024
	}{
		{"[0, 0, x, y] (c, d free)",
			func(x, y uint64) Twister.State1024 {
				var s Twister.State1024
				s[2], s[3] = x, y
				return s
			}},
		{"[0, 0, x, x] (c=d)",
			func(x, _ uint64) Twister.State1024 {
				var s Twister.State1024
				s[2], s[3] = x, x
				return s
			}},
		{"[x, x, x, x] (all equal)",
			func(x, _ uint64) Twister.State1024 {
				var s Twister.State1024
				s[0], s[1], s[2], s[3] = x, x, x, x
				return s
			}},
		{"[x, x, y, y] (pairs)",
			func(x, y uint64) Twister.State1024 {
				var s Twister.State1024
				s[0], s[1], s[2], s[3] = x, x, y, y
				return s
			}},
		{"[x, y, x, y] (alternating)",
			func(x, y uint64) Twister.State1024 {
				var s Twister.State1024
				s[0], s[1], s[2], s[3] = x, y, x, y
				return s
			}},
	}

	// Baseline: fully random inputs.
	baseline := measureAvalancheFull(r, T, func() Twister.State1024 {
		return randState(r)
	})
	t.Logf("Avalanche after FULL Permute (16 calls), ideal 512, σ 16:")
	t.Logf("  random baseline: %6.2f", baseline)

	for _, c := range cases {
		avg := measureAvalancheFull(r, T, func() Twister.State1024 {
			return c.makeIn(r.Uint64(), r.Uint64())
		})
		dev := (avg - 512.0) / 16.0
		tag := ""
		if dev < -3 {
			tag = "  >> WARNING"
		}
		t.Logf("  %-30s mean HW = %6.2f  (%+.2f σ)%s", c.name, avg, dev, tag)
	}

	// Injectivity for [0,0,x,y] under full Permute.
	t.Logf("")
	t.Logf("Injectivity on [0, 0, x, y] subspace (full Permute):")
	seen := make(map[Twister.State1024]bool, T)
	collisions := 0
	for i := 0; i < T; i++ {
		var s Twister.State1024
		s[2], s[3] = r.Uint64(), r.Uint64()
		s.Permute()
		if seen[s] {
			collisions++
		}
		seen[s] = true
	}
	t.Logf("  %d/%d distinct outputs, collisions: %d", len(seen), T, collisions)
	if collisions > 0 {
		t.Errorf(">> FAIL: [0,0,x,y] is not injective under full Permute")
	}
}

// measureAvalancheFull measures mean HW of (s1 ^ s2) over ALL 16 words
// after full Permute(), given a 1-bit input diff.
func measureAvalancheFull(r interface {
	Uint64() uint64
	Intn(int) int
}, T int, makeIn func() Twister.State1024) float64 {
	sum := int64(0)
	for i := 0; i < T; i++ {
		s1 := makeIn()
		s2 := s1
		bit := r.Intn(1024)
		flip(&s2, bit)

		s1.Permute()
		s2.Permute()
		sum += int64(hw(xor(s1, s2)))
	}
	return float64(sum) / float64(T)
}