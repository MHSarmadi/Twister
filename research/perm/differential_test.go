package perm

import (
	"testing"

	"github.com/MHSarmadi/Twister"
)

func TestDifferentialFull(t *testing.T) {
	r := rngSeed(0xdd3)
	T := 10000
	if fastMode() {
		T = 2000
	}
	bits := []int{0, 100, 400, 700, 1000}

	t.Logf("Full differential: T=%d per bit, Permute() at 16 calls", T)
	for _, bit := range bits {
		counts := make(map[Twister.State1024]int, T)
		for tr := 0; tr < T; tr++ {
			x := randState(r)
			y := x
			flip(&y, bit)
			x.Permute()
			y.Permute()
			counts[xor(x, y)]++
		}
		top := 0
		for _, c := range counts {
			if c > top {
				top = c
			}
		}
		t.Logf("  bit %4d: top-1 = 1/%.0f", bit, float64(T)/float64(top))
	}
}

// TestDifferentialLowWeight samples weight-2 and weight-3 input diffs.
// Sometimes a multi-bit diff beats every single-bit diff.
func TestDifferentialLowWeight(t *testing.T) {
	r := rngSeed(0xdd4)
	T := 5000
	if fastMode() {
		T = 1000
	}

	for _, w := range []int{1, 2, 3} {
		numDiffs := 8
		if fastMode() {
			numDiffs = 2
		}
		var globalMax float64
		for d := 0; d < numDiffs; d++ {
			var delta Twister.State1024
			used := map[int]bool{}
			for len(used) < w {
				used[r.Intn(1024)] = true
			}
			for bit := range used {
				flip(&delta, bit)
			}
			counts := make(map[Twister.State1024]int, T)
			for tr := 0; tr < T; tr++ {
				x := randState(r)
				y := x
				for k := 0; k < 16; k++ {
					y[k] ^= delta[k]
				}
				x.Permute()
				y.Permute()
				counts[xor(x, y)]++
			}
			top := 0
			for _, c := range counts {
				if c > top {
					top = c
				}
			}
			p := float64(top) / float64(T)
			if p > globalMax {
				globalMax = p
			}
		}
		t.Logf("weight-%d: max top-1 = 1/%.0f", w, 1/globalMax)
	}
}
