package mul

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

// ================================================================
// Test 1: Integral / Cube Attack
// ================================================================
//
// For a cube of k input bits, XOR the outputs over all 2^k values of
// those bits (keeping non-cube bits fixed). If the sum is 0 across
// multiple random non-cube fillings, there is an integral property
// (distinguisher). This is the empirical division-property test.
func TestIntegral(t *testing.T) {
	fmt.Println()
	fmt.Println("=== Integral / Cube Attack ===")
	fmt.Println("Cube: first k bits of word 0. Sum outputs over 2^k values,")
	fmt.Println("repeat with random non-cube bits. Sum==0 across all trials")
	fmt.Println("means an integral property exists (BAD — a distinguisher).")
	fmt.Println()

	rng := rand.New(rand.NewSource(0x1A7))
	const trials = 8

	fmt.Printf("%-3s | %-4s | %-12s | verdict\n", "k", "rd", "sum-of-sums")
	fmt.Println("------------------------------------------------")

	for _, k := range []int{8, 12, 16} {
		nCube := 1 << k
		for _, rounds := range []int{1, 2, 3, 4} {
			zeroSums := 0
			for trial := 0; trial < trials; trial++ {
				var xorSum [4]uint64
				base := State256{rng.Uint64() &^ ((1 << uint(k)) - 1),
					rng.Uint64(), rng.Uint64(), rng.Uint64()}
				for v := 0; v < nCube; v++ {
					x := base
					x[0] |= uint64(v)
					y := permuteN(x, rounds)
					xorSum[0] ^= y[0]
					xorSum[1] ^= y[1]
					xorSum[2] ^= y[2]
					xorSum[3] ^= y[3]
				}
				if xorSum[0] == 0 && xorSum[1] == 0 &&
					xorSum[2] == 0 && xorSum[3] == 0 {
					zeroSums++
				}
			}
			verdict := "excellent"
			if zeroSums == trials {
				verdict = "★ INTEGRAL — BAD"
			} else if zeroSums > 0 {
				verdict = "partial integral"
			}
			fmt.Printf("%-3d | R%-3d | %-12s | %s\n",
				k, rounds, fmt.Sprintf("%d/%d zero", zeroSums, trials), verdict)
		}
	}
}

// ================================================================
// Test 2: Linear Cryptanalysis (empirical)
// ================================================================
//
// For each input mask a and output mask b, samples x and computes
// bias = |Pr[parity(a·x) == parity(b·P(x))] - 1/2|.
// Uses Bonferroni-corrected significance (same convention as BIC tests).
func TestLinear(t *testing.T) {
	const samples = 8192 // noise floor ≈ 0.0055
	const stride = 4     // sample every 4th bit
	const alpha = 0.01

	rng := rand.New(rand.NewSource(0x11C))

	aBits := make([]int, 0, 64)
	bBits := make([]int, 0, 64)
	for b := 0; b < 256; b += stride {
		aBits = append(aBits, b)
		bBits = append(bBits, b)
	}

	numTests := len(aBits) * len(bBits)
	noise := 1.0 / (2.0 * math.Sqrt(float64(samples)))
	zCrit := bonferroniZ(alpha, numTests)

	fmt.Println()
	fmt.Println("=== Linear Cryptanalysis (empirical) ===")
	fmt.Printf("Masks sampled: %d input × %d output (every %dth bit)\n",
		len(aBits), len(bBits), stride)
	fmt.Printf("Total comparisons per round: %d\n", numTests)
	fmt.Printf("Samples/mask: %d, noise in |bias| ≈ %.4f\n", samples, noise)
	fmt.Printf("Bonferroni (α=%.2f): |bias| > %.4f (%.2fσ) is significant.\n",
		alpha, zCrit*noise, zCrit)
	fmt.Printf("Maxima up to ~%.1fσ are EXPECTED by chance (multiple comparisons).\n\n", zCrit)

	type worstEntry struct{ round, aBit, bBit int }
	worst := make([]worstEntry, 0, 4)

	for _, rounds := range []int{1, 2, 3, 4} {
		maxBias := 0.0
		var maxA, maxB int

		for _, aBit := range aBits {
			aWord, aPos := aBit/64, uint(aBit%64)
			for _, bBit := range bBits {
				bWord, bPos := bBit/64, uint(bBit%64)
				matches := 0
				for s := 0; s < samples; s++ {
					x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
					y := permuteN(x, rounds)
					ap := (x[aWord] >> aPos) & 1
					bp := (y[bWord] >> bPos) & 1
					if ap == bp {
						matches++
					}
				}
				bias := math.Abs(float64(matches)/float64(samples) - 0.5)
				if bias > maxBias {
					maxBias = bias
					maxA, maxB = aBit, bBit
				}
			}
		}

		zObs := maxBias / noise
		pValue := maxZPValue(zObs, numTests)
		verdict := pValueVerdict(pValue, alpha)
		fmt.Printf("rounds=%d: max |bias| = %.4f (%.1fσ, p=%.3f)  worst: (%d → %d)  [%s]\n",
			rounds, maxBias, zObs, pValue, maxA, maxB, verdict)
		worst = append(worst, worstEntry{rounds, maxA, maxB})
	}

	fmt.Printf("\nCross-round consistency (worst mask pair per round):\n")
	for _, e := range worst {
		fmt.Printf("  rounds=%d: (%d → %d)\n", e.round, e.aBit, e.bBit)
	}
	fmt.Printf("  (different pairs each round → noise; same pair → investigate)\n")
}

// ================================================================
// Test 3: Truncated Differentials (word-level)
// ================================================================
//
// Tracks which words have a nonzero difference, not individual bits.
// A structural weakness shows up as a persistent input-word-active
// pattern → output-word-active pattern that does not spread.
func TestTruncatedDiff(t *testing.T) {
	const samples = 8192
	rng := rand.New(rand.NewSource(0x7D))

	fmt.Println()
	fmt.Println("=== Truncated Differentials ===")
	fmt.Println("Word-level activation. Input: single word active.")
	fmt.Println("Ideal: output ALL words active with probability ≈ 1.")
	fmt.Printf("Samples per input pattern: %d\n\n", samples)

	fmt.Printf("%-4s | %-10s | %-14s %-14s | verdict\n",
		"rd", "inMask", "out=1111 (all)", "out=other-dominant")
	fmt.Println("----------------------------------------------------------------")

	for _, rounds := range []int{1, 2, 3, 4} {
		for inWord := 0; inWord < 4; inWord++ {
			counts := make(map[uint8]int, 16)
			for s := 0; s < samples; s++ {
				x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
				x2 := x
				bit := rng.Intn(64)
				x2[inWord] ^= 1 << uint(bit)
				y1 := permuteN(x, rounds)
				y2 := permuteN(x2, rounds)
				var outMask uint8
				for i := 0; i < 4; i++ {
					if y1[i] != y2[i] {
						outMask |= 1 << uint(i)
					}
				}
				counts[outMask]++
			}

			allActive := float64(counts[0xF]) / float64(samples)
			var domMask uint8
			domCount := 0
			for m, c := range counts {
				if m != 0xF && c > domCount {
					domMask, domCount = m, c
				}
			}
			domFrac := float64(domCount) / float64(samples)

			verdict := "excellent"
			if allActive < 0.999 {
				if domFrac > 0.01 {
					verdict = "SUSPICIOUS"
				} else {
					verdict = "borderline"
				}
			}
			fmt.Printf("%-4d | w%-9d | %-14.4f | 0x%X: %.4f       | %s\n",
				rounds, inWord, allActive, domMask, domFrac, verdict)
		}
	}
}

// ================================================================
// Test 4: Slide Structural Check
// ================================================================
//
// The round XORs identical constants every call. If the round has
// any internal self-similarity, iterating P might reveal it. Check:
// (a) does P have any fixed points up to constant injection?
// (b) does P∘P have low-HW fixed points?
func TestSlideStructural(t *testing.T) {
	const samples = 1 << 18
	rng := rand.New(rand.NewSource(0x51D))

	fmt.Println()
	fmt.Println("=== Slide / Self-Similarity Check ===")
	fmt.Println("Round uses identical constants every call. Check for")
	fmt.Println("any 'shift-invariant' state where P^k(x) ≈ x up to constant.")

	// Near-period-2: HW(P^2(x) ^ x)
	minHW := 256
	hs := make([]float64, samples)
	for i := 0; i < samples; i++ {
		x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
		y := permuteN(x, 2)
		hw := avalancheHamming(x, y)
		hs[i] = float64(hw)
		if hw < minHW {
			minHW = hw
		}
	}
	m, sd := meanStdDev(hs)
	fmt.Printf("\nHW(P²(x) ^ x): mean=%.2f std=%.2f min=%d (ideal ≈128,8,min>80)\n",
		m, sd, minHW)

	// Near-period-3
	minHW3 := 256
	for i := 0; i < samples; i++ {
		x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
		y := permuteN(x, 3)
		hw := avalancheHamming(x, y)
		if hw < minHW3 {
			minHW3 = hw
		}
	}
	fmt.Printf("HW(P³(x) ^ x): min=%d\n", minHW3)

	if minHW < 64 || minHW3 < 64 {
		fmt.Println("  SUSPICIOUS: near-periodic state found — investigate")
	} else {
		fmt.Println("  No near-periodic state found (min HW > 64 for periods 2 and 3)")
	}
}
