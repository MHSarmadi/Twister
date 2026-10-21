package perm

import (
	"testing"

	"github.com/MHSarmadi/Twister"
)

// minHW_over finds the minimum HW of f(x) over samples.
func minHW_over(r_samples []Twister.State1024, f func(Twister.State1024) Twister.State1024) int {
	minHW := 1 << 30
	for _, s := range r_samples {
		v := hw(f(s))
		if v < minHW {
			minHW = v
		}
	}
	return minHW
}

func TestSlide(t *testing.T) {
	r := rngSeed(0x51d)
	T := 500
	if fastMode() {
		T = 100
	}
	samples := make([]Twister.State1024, T)
	for i := range samples {
		samples[i] = randState(r)
	}

	min2 := minHW_over(samples, func(s Twister.State1024) Twister.State1024 {
		s.Permute()
		s.Permute()
		var out Twister.State1024
		_ = out
		// P^2(x) ⊕ x — need original. Rebuild from slice.
		_ = s
		return s
	})
	_ = min2

	// Do it properly.
	min2 = 1 << 30
	min3 := 1 << 30
	for _, x := range samples {
		y := x
		y.Permute()
		y.Permute()
		if v := hw(xor(y, x)); v < min2 {
			min2 = v
		}

		y = x
		y.Permute()
		y.Permute()
		y.Permute()
		if v := hw(xor(y, x)); v < min3 {
			min3 = v
		}
	}
	t.Logf("Slide: min HW(P²(x) ⊕ x) = %d, min HW(P³(x) ⊕ x) = %d (T=%d)", min2, min3, T)
	if min2 < 400 {
		t.Logf("  >> WARNING: min HW for P² below 400 — possible near-periodic structure")
	}
}

func TestRotational(t *testing.T) {
	r := rngSeed(0x707)
	T := 100
	if fastMode() {
		T = 20
	}
	minHW := 1 << 30
	worstR := 0
	for rot := 1; rot < 64; rot++ {
		for tr := 0; tr < T; tr++ {
			x := randState(r)
			xr := rotlState(x, rot)

			px := x
			px.Permute()
			pxr := rotlState(px, rot)

			xr.Permute()
			// Now xr holds P(rotl(x, rot))
			diff := hw(xor(xr, pxr))
			if diff < minHW {
				minHW = diff
				worstR = rot
			}
		}
	}
	t.Logf("Rotational: min HW(P(rotl(x,r)) ⊕ rotl(P(x),r)) = %d at r=%d (T=%d)", minHW, worstR, T)
	if minHW < 400 {
		t.Logf("  >> WARNING: rotational symmetry detected (min HW < 400)")
	}
}

func TestIntegral(t *testing.T) {
	r := rngSeed(0x1e6)
	trials := 4
	if fastMode() {
		trials = 1
	}

	cubeSizes := []int{8, 10, 12}
	if !fastMode() {
		cubeSizes = append(cubeSizes, 14)
	}

	for _, k := range cubeSizes {
		for _, numCalls := range []int{1, 2, 4, 8, 16} {
			allZero := 0
			for tr := 0; tr < trials; tr++ {
				// random k-bit subspace
				basis := make([]int, k)
				used := map[int]bool{}
				for b := 0; b < k; b++ {
					for {
						candidate := r.Intn(1024)
						if !used[candidate] {
							used[candidate] = true
							basis[b] = candidate
							break
						}
					}
				}
				var acc Twister.State1024
				for m := 0; m < (1 << k); m++ {
					x := randState(r)
					for b := 0; b < k; b++ {
						if m&(1<<b) != 0 {
							flip(&x, basis[b])
						}
					}
					x = applyPermuteCalls(x, numCalls)
					acc = xor(acc, x)
				}
				if hw(acc) == 0 {
					allZero++
				}
			}
			tag := ""
			if allZero > 0 {
				tag = "  >> WARNING: integral distinguisher!"
			}
			t.Logf("cube 2^%-2d, %2d calls: zero-sum %d/%d%s", k, numCalls, allZero, trials, tag)
		}
	}
}
