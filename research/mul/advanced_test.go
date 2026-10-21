// advanced_test.go — advanced differential and BIC tests for mult_permuter.
//
// Prerequisites (from avalanche_test.go or elsewhere in this package):
//   func (s *State256) mult_permuter()
//   func permuteN(s State256, n int) State256   // must call mult_permuter, not simple_permuter
//   func avalancheHamming(a, b State256) int
//   func meanStdDev(xs []float64) (float64, float64)

package mul

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

// ================================================================
// Statistical helpers
// ================================================================

// maxZPValue returns the two-sided p-value for observing a maximum
// standard normal |Z| = z across n independent tests.
// p > 0.01 ⟹ the maximum is consistent with noise.
func maxZPValue(z float64, n int) float64 {
	pSingle := math.Erfc(z / math.Sqrt2) // P(|Z| > z) for one test
	if pSingle < 1e-15 {
		return 0
	}
	return 1 - math.Pow(1-pSingle, float64(n))
}

// bonferroniZ returns the z threshold such that the family-wise
// false-alarm rate is ≈ alpha across n independent tests.
func bonferroniZ(alpha float64, n int) float64 {
	target := alpha / float64(n)
	lo, hi := 0.0, 10.0
	for i := 0; i < 60; i++ {
		mid := (lo + hi) / 2
		if math.Erfc(mid/math.Sqrt2) > target {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}

// pValueVerdict maps a p-value to a human verdict.
func pValueVerdict(p, alpha float64) string {
	switch {
	case p > 10*alpha:
		return "excellent"
	case p > alpha:
		return "clean"
	case p > alpha/10:
		return "borderline"
	default:
		return "SUSPICIOUS"
	}
}

// ================================================================
// Differential tests
// ================================================================

// TestMDP: max differential probability over sampled deltas, per round.
// Reports the collision index sumSq/N (≈ 1.0 for random functions, >1
// if output deltas cluster — sensitive to many small trails that a
// max-only statistic would miss).
func TestMDP(t *testing.T) {
	const samples = 4096
	const randomDeltas = 32
	rng := rand.New(rand.NewSource(0xD1FF))

	deltas := make([]State256, 0, 128+randomDeltas)
	for i := 0; i < 256; i += 2 {
		var d State256
		d[i/64] = 1 << uint(i%64)
		deltas = append(deltas, d)
	}
	for i := 0; i < randomDeltas; i++ {
		deltas = append(deltas, State256{
			rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64(),
		})
	}

	fmt.Printf("\n=== MDP Test ===\n")
	fmt.Printf("Deltas: 128 single-bit + %d random\n", randomDeltas)
	fmt.Printf("Samples per Δ: %d\n", samples)
	fmt.Printf("Random-function baseline: max DP ≤ 1/%d, collision index ≈ 1.0\n", samples)
	fmt.Printf("Collision index > 1.0 indicates output deltas clustering.\n\n")

	fmt.Printf("%-7s | %-11s | %-9s %-9s | verdict\n", "rounds", "max DP", "distinct", "collIdx")
	fmt.Println("--------------------------------------------------------------")

	for _, rounds := range []int{1, 2, 3, 4, 6} {
		maxDP := 0.0
		totalDistinct := 0
		totalCollIdx := 0.0

		for _, delta := range deltas {
			counts := make(map[State256]int, samples)
			for s := 0; s < samples; s++ {
				x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
				x2 := State256{
					x[0] ^ delta[0], x[1] ^ delta[1],
					x[2] ^ delta[2], x[3] ^ delta[3],
				}
				y1 := permuteN(x, rounds)
				y2 := permuteN(x2, rounds)
				counts[State256{
					y1[0] ^ y2[0], y1[1] ^ y2[1],
					y1[2] ^ y2[2], y1[3] ^ y2[3],
				}]++
			}
			mode, sumSq := 0, 0
			for _, c := range counts {
				if c > mode {
					mode = c
				}
				sumSq += c * c
			}
			dp := float64(mode) / float64(samples)
			if dp > maxDP {
				maxDP = dp
			}
			totalDistinct += len(counts)
			totalCollIdx += float64(sumSq) / float64(samples)
		}
		avgDistinct := float64(totalDistinct) / float64(len(deltas))
		avgCollIdx := totalCollIdx / float64(len(deltas))

		verdict := "excellent"
		if maxDP > 4.0/float64(samples) || avgCollIdx > 1.05 {
			verdict = "SUSPICIOUS"
		} else if maxDP > 2.0/float64(samples) || avgCollIdx > 1.01 {
			verdict = "borderline"
		}
		fmt.Printf("%-7d | %-11.3e | %-9.2f %-9.4f | %s\n",
			rounds, maxDP, avgDistinct, avgCollIdx, verdict)
	}
}

// TestStructuredDeltas: the exact deltas the design targets — top-bit,
// bottom-bit, alternating patterns — to check the design breaks the
// trivial trails of integer multiply (top-bit DP=1, bottom-bit leak).
func TestStructuredDeltas(t *testing.T) {
	const samples = 8192
	rng := rand.New(rand.NewSource(0xD1F4))

	fmt.Printf("\n=== Structured Delta Test ===\n")
	fmt.Printf("Ideal: mean HW ≈ 128, std ≈ 8 (Binomial(256, 0.5)).\n\n")

	cases := []struct {
		name string
		d    State256
	}{
		{"top-bit (w0 only)", State256{1 << 63, 0, 0, 0}},
		{"bottom-bit (w0 only)", State256{1, 0, 0, 0}},
		{"top-bit (all words)", State256{1 << 63, 1 << 63, 1 << 63, 1 << 63}},
		{"bot-bit (all words)", State256{1, 1, 1, 1}},
		{"top+bot (w0)", State256{(1 << 63) | 1, 0, 0, 0}},
		{"mid-bit (w0, b32)", State256{1 << 32, 0, 0, 0}},
		{"alt 0xAAAA…", State256{0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA, 0xAAAAAAAAAAAAAAAA}},
		{"alt 0x5555…", State256{0x5555555555555555, 0x5555555555555555, 0x5555555555555555, 0x5555555555555555}},
	}

	fmt.Printf("%-22s | %-4s | %-7s %-7s | verdict\n", "Δ", "rd", "mean", "std")
	fmt.Println("---------------------------------------------------------------------")

	for _, c := range cases {
		for _, rounds := range []int{1, 2, 4} {
			hs := make([]float64, samples)
			for s := range hs {
				x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
				x2 := State256{
					x[0] ^ c.d[0], x[1] ^ c.d[1],
					x[2] ^ c.d[2], x[3] ^ c.d[3],
				}
				hs[s] = float64(avalancheHamming(permuteN(x, rounds), permuteN(x2, rounds)))
			}
			m, sd := meanStdDev(hs)
			verdict := "ok"
			if math.Abs(m-128) < 1.0 && sd < 8.5 {
				verdict = "ideal"
			} else if math.Abs(m-128) > 4.0 || sd > 10 {
				verdict = "SUSPICIOUS"
			}
			fmt.Printf("%-22s | R%-3d | %-7.2f %-7.2f | %s\n",
				c.name, rounds, m, sd, verdict)
		}
	}
}

// TestDiffSpread: fraction of distinct output deltas out of N samples.
func TestDiffSpread(t *testing.T) {
	const samples = 1024
	rng := rand.New(rand.NewSource(0xD1F3))

	fmt.Printf("\n=== Differential Spread ===\n")
	fmt.Printf("Fraction of distinct output deltas out of %d samples.\n", samples)
	fmt.Printf("Ideal ≈ 1.0 (no dominant trail).\n\n")

	deltas := []State256{
		{1 << 63, 0, 0, 0},
		{1, 0, 0, 0},
		{1 << 32, 0, 0, 0},
		{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()},
	}

	fmt.Printf("%-20s | %-4s | %-8s %-8s\n", "Δ", "rd", "distinct", "fraction")
	fmt.Println("------------------------------------------------------")
	for _, delta := range deltas {
		for _, rounds := range []int{1, 2, 4} {
			seen := make(map[State256]struct{}, samples)
			for s := 0; s < samples; s++ {
				x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
				x2 := State256{
					x[0] ^ delta[0], x[1] ^ delta[1],
					x[2] ^ delta[2], x[3] ^ delta[3],
				}
				y1 := permuteN(x, rounds)
				y2 := permuteN(x2, rounds)
				seen[State256{
					y1[0] ^ y2[0], y1[1] ^ y2[1],
					y1[2] ^ y2[2], y1[3] ^ y2[3],
				}] = struct{}{}
			}
			frac := float64(len(seen)) / float64(samples)
			verdict := ""
			if frac < 0.99 {
				verdict = "  ← possible trail"
			}
			fmt.Printf("[%016x..]          | R%-3d | %-8d %-8.4f%s\n",
				delta[0], rounds, len(seen), frac, verdict)
		}
	}
}

// TestHigherOrderDiff: flip TWO random input bits simultaneously.
func TestHigherOrderDiff(t *testing.T) {
	const samples = 4096
	rng := rand.New(rand.NewSource(0xD1F5))

	fmt.Printf("\n=== Higher-Order Differential (2-bit flip) ===\n")
	fmt.Printf("Ideal: same as single-bit — mean HW ≈ 128, std ≈ 8.\n\n")

	for _, rounds := range []int{1, 2, 4} {
		hs := make([]float64, samples)
		for s := range hs {
			x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
			x2 := x
			b1 := rng.Intn(256)
			b2 := rng.Intn(256)
			for b1 == b2 {
				b2 = rng.Intn(256)
			}
			x2[b1/64] ^= 1 << uint(b1%64)
			x2[b2/64] ^= 1 << uint(b2%64)
			hs[s] = float64(avalancheHamming(
				permuteN(x, rounds), permuteN(x2, rounds)))
		}
		m, sd := meanStdDev(hs)
		fmt.Printf("rounds=%d: 2-bit flip → mean HW = %.2f, std = %.2f\n",
			rounds, m, sd)
	}
}

// ================================================================
// BIC tests (with multiple-comparisons correction)
// ================================================================

// TestBIC: Bit Independence Criterion.
// For each input-bit flip and each output-bit pair, compute the Pearson
// correlation of the two output difference bits. Under the null, the
// maximum |ρ| follows the extreme-value distribution of N normals;
// Bonferroni-corrected p-values eliminate false alarms.
func TestBIC(t *testing.T) {
	const samples = 2048
	const numRandomPairs = 500
	const alpha = 0.01
	rng := rand.New(rand.NewSource(0xB1C))

	pairs := make([][2]int, 0, 255+numRandomPairs)
	for j := 0; j < 255; j++ {
		pairs = append(pairs, [2]int{j, j + 1})
	}
	for len(pairs) < 255+numRandomPairs {
		j := rng.Intn(256)
		k := rng.Intn(256)
		if j == k {
			continue
		}
		if j > k {
			j, k = k, j
		}
		pairs = append(pairs, [2]int{j, k})
	}

	numTests := 256 * len(pairs)
	noise := 1.0 / math.Sqrt(float64(samples))
	zCrit := bonferroniZ(alpha, numTests)

	fmt.Printf("\n=== BIC Test ===\n")
	fmt.Printf("Pairs: %d (255 adjacent + %d random)\n", len(pairs), numRandomPairs)
	fmt.Printf("Total comparisons: 256 input bits × %d pairs = %d\n", len(pairs), numTests)
	fmt.Printf("Samples per input bit: %d, noise in |ρ| ≈ %.4f\n", samples, noise)
	fmt.Printf("Bonferroni (α=%.2f): |ρ| > %.4f (%.2fσ) is significant.\n",
		alpha, zCrit*noise, zCrit)
	fmt.Printf("Maxima up to ~%.1fσ are EXPECTED by chance (multiple comparisons).\n\n", zCrit)

	type entry struct{ rounds, inBit, outJ, outK int }
	worst := make([]entry, 0, 4)

	for _, rounds := range []int{1, 2, 3, 4} {
		maxCorr := 0.0
		var maxBit, maxJ, maxK int

		for inputBit := 0; inputBit < 256; inputBit++ {
			w := inputBit / 64
			b := inputBit % 64
			fJ := make([]int, len(pairs))
			fK := make([]int, len(pairs))
			both := make([]int, len(pairs))

			for s := 0; s < samples; s++ {
				x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
				x2 := x
				x2[w] ^= 1 << uint(b)
				y1 := permuteN(x, rounds)
				y2 := permuteN(x2, rounds)
				diff := [4]uint64{
					y1[0] ^ y2[0], y1[1] ^ y2[1],
					y1[2] ^ y2[2], y1[3] ^ y2[3],
				}
				for p, pr := range pairs {
					dj := int((diff[pr[0]/64] >> uint(pr[0]%64)) & 1)
					dk := int((diff[pr[1]/64] >> uint(pr[1]%64)) & 1)
					fJ[p] += dj
					fK[p] += dk
					both[p] += dj & dk
				}
			}

			for p := range pairs {
				pJ := float64(fJ[p]) / float64(samples)
				pK := float64(fK[p]) / float64(samples)
				// Skip degenerate balance: near-0 or near-1 denominators
				// amplify noise and create spurious high |ρ|.
				if pJ < 0.05 || pJ > 0.95 || pK < 0.05 || pK > 0.95 {
					continue
				}
				pJK := float64(both[p]) / float64(samples)
				denom := math.Sqrt(pJ * (1 - pJ) * pK * (1 - pK))
				rho := (pJK - pJ*pK) / denom
				if math.Abs(rho) > maxCorr {
					maxCorr = math.Abs(rho)
					maxBit = inputBit
					maxJ = pairs[p][0]
					maxK = pairs[p][1]
				}
			}
		}

		zObs := maxCorr / noise
		pValue := maxZPValue(zObs, numTests)
		verdict := pValueVerdict(pValue, alpha)

		fmt.Printf("rounds=%d: max |ρ| = %.4f (%.1fσ, p=%.3f)  worst: in-bit %d, out-pair (%d,%d)  [%s]\n",
			rounds, maxCorr, zObs, pValue, maxBit, maxJ, maxK, verdict)
		worst = append(worst, entry{rounds, maxBit, maxJ, maxK})
	}

	// Cross-round consistency: a real structural correlation would be
	// flagged on the SAME (input, output-pair) at multiple rounds.
	// Different triples each round = noise.
	fmt.Printf("\nCross-round consistency (worst triple per round):\n")
	for _, e := range worst {
		fmt.Printf("  rounds=%d: in-bit %d → out-pair (%d,%d)\n",
			e.rounds, e.inBit, e.outJ, e.outK)
	}
	fmt.Printf("  (different triples each round → noise; same triple → investigate)\n")
}

// TestBICBalance: prerequisite for BIC — P(Δ_j = 1) should be ≈ 0.5
// for every (input bit, output bit) pair.
func TestBICBalance(t *testing.T) {
	const samples = 1024
	const alpha = 0.01
	rng := rand.New(rand.NewSource(0xB2C))

	numTests := 256 * 256
	noise := 0.5 / math.Sqrt(float64(samples))
	zCrit := bonferroniZ(alpha, numTests)

	fmt.Printf("\n=== BIC Balance ===\n")
	fmt.Printf("For each (input bit, output bit) pair, P(Δ_j=1) should be ≈ 0.5.\n")
	fmt.Printf("Total comparisons: %d\n", numTests)
	fmt.Printf("Samples per input bit: %d, noise in P ≈ ±%.4f\n", samples, noise)
	fmt.Printf("Bonferroni (α=%.2f): |P-0.5| > %.4f (%.2fσ) is significant.\n\n",
		alpha, zCrit*noise, zCrit)

	for _, rounds := range []int{1, 2} {
		maxDev := 0.0
		var worstIn, worstOut int

		for inputBit := 0; inputBit < 256; inputBit++ {
			w := inputBit / 64
			b := inputBit % 64
			counts := make([]int, 256)
			for s := 0; s < samples; s++ {
				x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
				x2 := x
				x2[w] ^= 1 << uint(b)
				y1 := permuteN(x, rounds)
				y2 := permuteN(x2, rounds)
				diff := [4]uint64{
					y1[0] ^ y2[0], y1[1] ^ y2[1],
					y1[2] ^ y2[2], y1[3] ^ y2[3],
				}
				for j := 0; j < 256; j++ {
					counts[j] += int((diff[j/64] >> uint(j%64)) & 1)
				}
			}
			for j := 0; j < 256; j++ {
				p := float64(counts[j]) / float64(samples)
				dev := math.Abs(p - 0.5)
				if dev > maxDev {
					maxDev = dev
					worstIn, worstOut = inputBit, j
				}
			}
		}

		zObs := maxDev / noise
		pValue := maxZPValue(zObs, numTests)
		verdict := pValueVerdict(pValue, alpha)

		fmt.Printf("rounds=%d: max |P-0.5| = %.4f (%.2fσ, p=%.3f)  worst: in-bit %d → out-bit %d  [%s]\n",
			rounds, maxDev, zObs, pValue, worstIn, worstOut, verdict)
	}
}

// TestBICAdjacentOnly: tighter BIC focused on adjacent output bits —
// the pairs most likely to be correlated if the multiply's local
// structure leaks through. Same multiple-comparisons correction.
func TestBICAdjacentOnly(t *testing.T) {
	const samples = 4096
	const alpha = 0.01
	rng := rand.New(rand.NewSource(0xB3C))

	numTests := 256 * 255
	noise := 1.0 / math.Sqrt(float64(samples))
	zCrit := bonferroniZ(alpha, numTests)

	fmt.Printf("\n=== BIC (Adjacent Output Bits Only) ===\n")
	fmt.Printf("Pairs: 255 adjacent, %d samples per input bit.\n", samples)
	fmt.Printf("Total comparisons: 256 × 255 = %d\n", numTests)
	fmt.Printf("Noise in |ρ| ≈ %.4f\n", noise)
	fmt.Printf("Bonferroni (α=%.2f): |ρ| > %.4f (%.2fσ) is significant.\n\n",
		alpha, zCrit*noise, zCrit)

	for _, rounds := range []int{1, 2} {
		maxCorr := 0.0
		var maxBit, maxJ int

		for inputBit := 0; inputBit < 256; inputBit++ {
			w := inputBit / 64
			b := inputBit % 64
			fJ := make([]int, 255)
			fK := make([]int, 255)
			both := make([]int, 255)

			for s := 0; s < samples; s++ {
				x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
				x2 := x
				x2[w] ^= 1 << uint(b)
				y1 := permuteN(x, rounds)
				y2 := permuteN(x2, rounds)
				diff := [4]uint64{
					y1[0] ^ y2[0], y1[1] ^ y2[1],
					y1[2] ^ y2[2], y1[3] ^ y2[3],
				}
				for j := 0; j < 255; j++ {
					dj := int((diff[j/64] >> uint(j%64)) & 1)
					dk := int((diff[(j+1)/64] >> uint((j+1)%64)) & 1)
					fJ[j] += dj
					fK[j] += dk
					both[j] += dj & dk
				}
			}

			for j := 0; j < 255; j++ {
				pJ := float64(fJ[j]) / float64(samples)
				pK := float64(fK[j]) / float64(samples)
				if pJ < 0.05 || pJ > 0.95 || pK < 0.05 || pK > 0.95 {
					continue
				}
				pJK := float64(both[j]) / float64(samples)
				denom := math.Sqrt(pJ * (1 - pJ) * pK * (1 - pK))
				rho := (pJK - pJ*pK) / denom
				if math.Abs(rho) > maxCorr {
					maxCorr = math.Abs(rho)
					maxBit = inputBit
					maxJ = j
				}
			}
		}

		zObs := maxCorr / noise
		pValue := maxZPValue(zObs, numTests)
		verdict := pValueVerdict(pValue, alpha)

		fmt.Printf("rounds=%d: max |ρ| = %.4f (%.1fσ, p=%.3f)  worst: in-bit %d, out-pair (%d,%d)  [%s]\n",
			rounds, maxCorr, zObs, pValue, maxBit, maxJ, maxJ+1, verdict)
	}
}
