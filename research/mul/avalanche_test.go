package mul

import (
	"fmt"
	"math"
	"math/bits"
	"math/rand"
	"testing"
)

// avalancheHamming returns the Hamming distance between two 256-bit states.
func avalancheHamming(a, b State256) int {
	d := 0
	for i := 0; i < 4; i++ {
		d += bits.OnesCount64(a[i] ^ b[i])
	}
	return d
}

// permuteN applies simple_permuter n times.
func permuteN(s State256, n int) State256 {
	for i := 0; i < n; i++ {
		s.mult_permuter()
	}
	return s
}

func meanStdDev(xs []float64) (mean, stddev float64) {
	var sum, sumsq float64
	for _, x := range xs {
		sum += x
		sumsq += x * x
	}
	n := float64(len(xs))
	mean = sum / n
	stddev = math.Sqrt(sumsq/n - mean*mean)
	return
}

// TestAvalanche measures the one-bit avalanche per round count.
func TestAvalanche(t *testing.T) {
	roundsList := []int{1, 2, 3, 4, 6, 8}
	trialsPerBit := 1000
	rng := rand.New(rand.NewSource(0xC0FFEE))

	fmt.Println()
	fmt.Println("=== Avalanche test on clmul.State256 ===")
	fmt.Println("State size: 256 bits; ideal mean = 128, ideal stddev ≈ 8")
	fmt.Printf("Trials per input bit: %d\n", trialsPerBit)
	fmt.Printf("Total trials per round count: %d\n\n", 256*trialsPerBit)

	fmt.Printf("%-7s | %-8s %-8s %-4s %-4s | %-9s | %-10s\n",
		"rounds", "mean", "stddev", "min", "max", "mean−128", "verdict")
	fmt.Println("-------------------------------------------------------------------------")

	for _, n := range roundsList {
		ds := make([]float64, 0, 256*trialsPerBit)
		minD, maxD := 256.0, 0.0
		for bit := 0; bit < 256; bit++ {
			for tr := 0; tr < trialsPerBit; tr++ {
				x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
				x2 := x
				x2[bit/64] ^= 1 << uint(bit%64)
				d := float64(avalancheHamming(permuteN(x, n), permuteN(x2, n)))
				ds = append(ds, d)
				if d < minD {
					minD = d
				}
				if d > maxD {
					maxD = d
				}
			}
		}
		mean, stddev := meanStdDev(ds)
		verdict := "weak"
		switch {
		case math.Abs(mean-128) < 2 && stddev < 9:
			verdict = "excellent"
		case math.Abs(mean-128) < 8 && stddev < 16:
			verdict = "good"
		case math.Abs(mean-128) < 24:
			verdict = "ok"
		}
		fmt.Printf("%-7d | %-8.2f %-8.2f %-4.0f %-4.0f | %+9.2f | %-10s\n",
			n, mean, stddev, minD, maxD, mean-128, verdict)
	}

	// Per-input-word breakdown for 1 and 4 rounds.
	for _, n := range []int{1, 4} {
		fmt.Printf("\n=== Per-input-word avalanche (rounds=%d) ===\n", n)
		for w := 0; w < 4; w++ {
			vals := make([]float64, 0, 64*trialsPerBit)
			for bit := 0; bit < 64; bit++ {
				fullBit := w*64 + bit
				for tr := 0; tr < trialsPerBit; tr++ {
					x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
					x2 := x
					x2[fullBit/64] ^= 1 << uint(fullBit%64)
					vals = append(vals, float64(avalancheHamming(permuteN(x, n), permuteN(x2, n))))
				}
			}
			m, sd := meanStdDev(vals)
			fmt.Printf("  word %d: mean = %6.2f, stddev = %5.2f\n", w, m, sd)
		}
	}
}

// TestStateHW tracks the total Hamming weight over rounds. If it drops or
// drifts, the per-word fixup is compressing the state — see the analysis note.
func TestStateHW(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	fmt.Println("\n=== State HW over rounds (diagnostic for fixup compression) ===")
	fmt.Printf("%-7s %-10s %-8s\n", "rounds", "mean HW", "stddev")
	for _, n := range []int{0, 1, 2, 4, 8, 16} {
		hw := make([]float64, 10000)
		for i := range hw {
			x := State256{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
			y := permuteN(x, n)
			hw[i] = float64(bits.OnesCount64(y[0]) + bits.OnesCount64(y[1]) +
				bits.OnesCount64(y[2]) + bits.OnesCount64(y[3]))
		}
		m, sd := meanStdDev(hw)
		fmt.Printf("%-7d %-10.2f %-8.2f\n", n, m, sd)
	}
}
