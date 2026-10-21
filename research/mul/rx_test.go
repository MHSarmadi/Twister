package mul

import (
	"fmt"
	"math/bits"
	"math/rand"
	"testing"
)

// TestRXPenalty: Rotational-XOR cryptanalysis penalty test.
// For each rotation r, HW(c ^ rotl(c, r)) measures how strongly the
// constant injection breaks an RX trail with rotation r.
// HIGH HW = good (breaks trail), LOW HW = bad (trail survives).
// Reference: Ashur & Liu, "Rotational Cryptanalysis in the Presence
// of Constants" (TCHES 2016).
func TestRXPenalty(t *testing.T) {
	constants := []struct {
		name string
		val  uint64
	}{
		{"c0", 0x7b4d4f544946217d},
		{"c1", 0x3a5361726d616469},
	}

	fmt.Println()
	fmt.Println("=== RX Penalty Test ===")
	fmt.Println("For each rotation r, HW(c ^ rotl(c, r)) measures how strongly")
	fmt.Println("the constant injection breaks an RX trail of rotation r.")
	fmt.Println("HIGH HW (≈32) = safe, LOW HW (<16) = suspicious, 0 = catastrophic.")
	fmt.Println()

	const threshSafe = 24
	const threshWarn = 16

	fmt.Printf("%-4s | %-18s | %-8s %-8s %-8s | verdict\n",
		"name", "value", "minHW", "avgHW", "worst_r")
	fmt.Println("----------------------------------------------------------------------")

	allSafe := true
	worstCombined := 128
	worstCombR := 0

	for _, c := range constants {
		minHW := 64
		worstR := 0
		sumHW := 0
		for r := 1; r < 64; r++ {
			hw := bits.OnesCount64(c.val ^ bits.RotateLeft64(c.val, r))
			sumHW += hw
			if hw < minHW {
				minHW = hw
				worstR = r
			}
		}
		avgHW := float64(sumHW) / 63.0
		verdict := "excellent"
		if minHW < threshWarn {
			verdict = "SUSPICIOUS"
			allSafe = false
		} else if minHW < threshSafe {
			verdict = "borderline"
		}
		fmt.Printf("%-4s | 0x%016x | %-8d %-8.2f %-8d | %s\n",
			c.name, c.val, minHW, avgHW, worstR, verdict)
	}

	fmt.Println()
	fmt.Println("Per-rotation HW (only showing r where HW < 24):")
	found := false
	for _, c := range constants {
		for r := 1; r < 64; r++ {
			hw := bits.OnesCount64(c.val ^ bits.RotateLeft64(c.val, r))
			if hw < 24 {
				fmt.Printf("  %s, r=%2d: HW = %d\n", c.name, r, hw)
				found = true
			}
		}
	}
	if !found {
		fmt.Println("  (none — all rotations give HW >= 24, strong)")
	}

	fmt.Println()
	fmt.Println("Combined penalty per rotation (both constants):")
	fmt.Println("(combined HW > 32: safe; > 48: very strong)")
	fmt.Println()
	fmt.Printf("%-4s | %-8s %-8s | %-8s | verdict\n", "r", "HW0", "HW1", "combined")
	fmt.Println("---------------------------------------------")
	for r := 1; r < 64; r++ {
		hw0 := bits.OnesCount64(constants[0].val ^ bits.RotateLeft64(constants[0].val, r))
		hw1 := bits.OnesCount64(constants[1].val ^ bits.RotateLeft64(constants[1].val, r))
		combined := hw0 + hw1
		if combined < worstCombined {
			worstCombined = combined
			worstCombR = r
		}
		if combined < 32 {
			verdict := "SUSPICIOUS"
			if combined < 16 {
				verdict = "CATASTROPHIC"
			}
			fmt.Printf("%-4d | %-8d %-8d | %-8d | %s\n", r, hw0, hw1, combined, verdict)
		}
	}
	fmt.Printf("\nWorst combined HW = %d at r=%d (threshold: >32 safe, >48 excellent)\n",
		worstCombined, worstCombR)

	if allSafe && worstCombined >= 32 {
		fmt.Println("\nVERDICT: constants are RX-safe across all 63 non-identity rotations.")
	} else {
		fmt.Println("\nVERDICT: SOME ROTATIONS ARE WEAK — see above.")
	}
}

// TestFixedPointsAndCycles: check for fixed points, 2-cycles, and
// near-fixed-points (states where P(x) ^ x has low HW).
func TestFixedPointsAndCycles(t *testing.T) {
	const samples = 1 << 20
	rng := rand.New(rand.NewSource(0xF1E))

	fmt.Println()
	fmt.Println("=== Fixed Points & Short Cycles ===")
	fmt.Printf("Samples: %d\n\n", samples)

	// Near-fixed-points: measure HW(P(x) ^ x)
	hs := make([]float64, samples)
	minHW := 256
	for i := 0; i < samples; i++ {
		x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
		y := permuteN(x, 1)
		hw := avalancheHamming(x, y)
		hs[i] = float64(hw)
		if hw < minHW {
			minHW = hw
		}
	}
	mean, sd := meanStdDev(hs)
	fmt.Printf("HW(P(x) ^ x): mean = %.2f, std = %.2f, min = %d\n", mean, sd, minHW)
	fmt.Printf("Ideal: mean ≈ 128, std ≈ 8, min ≈ 80+ (no near-fixed-points)\n")
	if minHW < 64 {
		fmt.Printf("  WARNING: found state with HW(P(x) ^ x) = %d\n", minHW)
	}

	// 2-cycles
	found := 0
	for i := 0; i < 100000 && found < 3; i++ {
		x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
		if permuteN(x, 2) == x {
			found++
			fmt.Printf("  Found 2-cycle: %v\n", x)
		}
	}
	if found == 0 {
		fmt.Println("2-cycles: none found in 100,000 samples (expected: 0)")
	}

	fmt.Println()
	fmt.Println("Cycle search: iterate P for 256 steps from 100 random starts.")
	fmt.Println("(Any cycle length < 256 would be a structural weakness.)")
	shortCycles := 0
	for start := 0; start < 100; start++ {
		s := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
		seen := make(map[State256]int)
		s = permuteN(s, 1)
		seen[s] = 1
		for step := 2; step <= 256; step++ {
			s = permuteN(s, 1)
			if prev, ok := seen[s]; ok {
				shortCycles++
				fmt.Printf("  Found cycle length %d from start %d\n", step-prev, start)
				break
			}
			seen[s] = step
		}
	}
	if shortCycles == 0 {
		fmt.Println("  No short cycles found (256 steps × 100 starts)")
	}
}

// TestRotationalSymmetry: directly test whether P commutes with rotation:
// P(rotl(x, r)) ==? rotl(P(x), r) for each r.
// If any r (≠ 0) gives low XOR difference for many samples, the design
// has a rotational symmetry.
func TestRotationalSymmetry(t *testing.T) {
	const samples = 1000
	const rounds = 4
	rng := rand.New(rand.NewSource(0x50F))

	fmt.Println()
	fmt.Println("=== Rotational Symmetry Test ===")
	fmt.Printf("For each rotation r, sample %d inputs.\n", samples)
	fmt.Printf("Compute P(rotl(x, r)) ^ rotl(P(x), r) over %d rounds.\n", rounds)
	fmt.Println("Low HW = rotational symmetry (bad). High HW ≈ 128 = no symmetry.")
	fmt.Println()

	fmt.Printf("%-4s | %-8s %-8s | verdict\n", "r", "mean HW", "min HW")
	fmt.Println("------------------------------------------")

	suspect := false
	for r := 1; r < 64; r++ {
		hs := make([]float64, samples)
		minHW := 256
		for i := 0; i < samples; i++ {
			x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
			xr := State256{
				bits.RotateLeft64(x[0], r), bits.RotateLeft64(x[1], r),
				bits.RotateLeft64(x[2], r), bits.RotateLeft64(x[3], r),
			}
			pr := permuteN(xr, rounds)
			px := permuteN(x, rounds)
			pxr := State256{
				bits.RotateLeft64(px[0], r), bits.RotateLeft64(px[1], r),
				bits.RotateLeft64(px[2], r), bits.RotateLeft64(px[3], r),
			}
			hw := avalancheHamming(pr, pxr)
			hs[i] = float64(hw)
			if hw < minHW {
				minHW = hw
			}
		}
		m, _ := meanStdDev(hs)
		verdict := ""
		if m < 64 {
			verdict = "  ← SUSPICIOUS"
			suspect = true
		} else if m < 100 {
			verdict = "  ← borderline"
		}
		fmt.Printf("%-4d | %-8.2f %-8d | %s\n", r, m, minHW, verdict)
	}
	if !suspect {
		fmt.Println("\nNo rotational symmetry detected (all mean HW >= 100).")
	}
}
