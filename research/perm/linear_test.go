package perm

import (
	"math"
	"testing"

	"github.com/MHSarmadi/Twister"
)

func TestLinearFull(t *testing.T) {
	r := rngSeed(0x11f)
	T := 3000
	if fastMode() {
		T = 500
	}
	const numIn = 20
	const numOut = 20

	// Sample masks
	inBits := sampleBits(r, 1024, numIn)
	outBits := sampleBits(r, 1024, numOut)
	inMasks := make([]Twister.State1024, numIn)
	for i, bit := range inBits {
		flip(&inMasks[i], bit)
	}
	outMasks := make([]Twister.State1024, numOut)
	for i, bit := range outBits {
		flip(&outMasks[i], bit)
	}

	// Precompute P(x) for all samples — 3000 permutes, not 1.2M.
	samples := make([]Twister.State1024, T)
	outputs := make([]Twister.State1024, T)
	for i := 0; i < T; i++ {
		samples[i] = randState(r)
		outputs[i] = samples[i]
		outputs[i].Permute()
	}

	// For each mask pair, compute bias.
	noise := 1.0 / math.Sqrt(float64(T))
	var maxBias float64
	var maxInBit, maxOutBit int
	for ii := 0; ii < numIn; ii++ {
		for oi := 0; oi < numOut; oi++ {
			sum := 0
			for j := 0; j < T; j++ {
				b := parity(samples[j], inMasks[ii]) ^ parity(outputs[j], outMasks[oi])
				if b == 0 {
					sum++
				} else {
					sum--
				}
			}
			bias := math.Abs(float64(sum) / float64(T))
			if bias > maxBias {
				maxBias = bias
				maxInBit = inBits[ii]
				maxOutBit = outBits[oi]
			}
		}
	}

	t.Logf("Linear full: T=%d, noise=%0.4f", T, noise)
	t.Logf("  max |bias| = %.4f (%.2f× noise) at in_bit=%d, out_bit=%d",
		maxBias, maxBias/noise, maxInBit, maxOutBit)
	if maxBias > 5*noise {
		t.Logf("  >> WARNING: bias above 5σ — investigate")
	}
}
