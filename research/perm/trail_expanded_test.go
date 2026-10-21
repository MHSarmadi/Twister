package perm

import (
	"testing"

	"github.com/MHSarmadi/Twister"
)

// firstTouchedDepth returns the smallest n such that QR#n-1's input set
// contains word w. For word w, this is the first QR that can propagate
// a diff starting in w.
func firstTouchedDepth(word int) int {
	for i, idx := range qrIndices {
		for _, p := range idx {
			if p == word {
				return i + 1
			}
		}
	}
	return 999
}

// TestTrailExpanded only reports cells where the applied QR chain
// actually touches the flipped bit's word. Cells where the word is not yet
// touched are skipped (they trivially produce the input diff unchanged).
func TestTrailExpanded(t *testing.T) {
	r := rngSeed(0xe7a2)
	const baseT = 10000
	const escT = 50000
	qrDepths := []int{1, 2, 4}

	var bits []int
	for w := 0; w < 16; w++ {
		bits = append(bits, w*64+0, w*64+32, w*64+63)
	}

	type stat struct {
		worstInv float64 // smallest non-saturated inverse
		worstBit int
		worstQrs int
		nonSat   int
		sat      int
		skipped  int
	}
	byDepth := make(map[int]*stat)
	for _, n := range qrDepths {
		byDepth[n] = &stat{worstInv: 1e9}
	}

	for _, bit := range bits {
		word := bit / 64
		firstDepth := firstTouchedDepth(word)
		for _, n := range qrDepths {
			s := byDepth[n]
			if n < firstDepth {
				s.skipped++
				continue
			}
			inv, sat := measureTrailAtDepth2(r, bit, n, baseT, escT)
			if sat {
				s.sat++
			} else {
				s.nonSat++
				if inv < s.worstInv {
					s.worstInv = inv
					s.worstBit = bit
					s.worstQrs = n
				}
			}
		}
	}

	t.Logf("Trail summary (cells where the applied QRs actually touch the bit):")
	t.Logf("%-6s | %-9s | %-9s | %-9s | %s",
		"QRs", "touched", "non-sat", "sat", "worst trail")
	for _, n := range qrDepths {
		s := byDepth[n]
		touched := s.nonSat + s.sat
		worstStr := "none above noise floor"
		if s.nonSat > 0 {
			worstStr = "1/" + formatInv(s.worstInv) +
				" (bit " + itoa(s.worstBit) + ")"
		}
		t.Logf("%-6d | %-9d | %-9d | %-9d | %s",
			n, touched, s.nonSat, s.sat, worstStr)
	}

	// Regression check: if any non-saturated cell is worse than 1/30,
	// we're genuinely weaker than the ARX V2 baseline.
	regressed := false
	for _, n := range qrDepths {
		s := byDepth[n]
		if s.nonSat > 0 && s.worstInv < 30 {
			t.Errorf(">> REGRESSION: %d QRs worst trail 1/%.1f below 1/30", n, s.worstInv)
			regressed = true
		}
	}
	if !regressed {
		t.Logf("")
		t.Logf("No cell worse than 1/30 for any sampled bit. All trails are below")
		t.Logf("the noise floor (T=%d, so trail < 2^-13.3).", escT)
	}
}

func measureTrailAtDepth2(r interface {
	Uint64() uint64
	Intn(int) int
}, bit, qrs, baseT, escT int) (float64, bool) {
	counts := make(map[Twister.State1024]int)
	trials := 0
	for trials < escT {
		chunk := baseT
		if trials+chunk > escT {
			chunk = escT - trials
		}
		for tr := 0; tr < chunk; tr++ {
			x := randStateConcrete(r)
			y := x
			flip(&y, bit)
			x = applyQRs(x, qrs)
			y = applyQRs(y, qrs)
			counts[xor(x, y)]++
		}
		trials += chunk
		top := 0
		for _, c := range counts {
			if c > top {
				top = c
			}
		}
		if top >= 5 {
			return float64(trials) / float64(top), false
		}
	}
	top := 0
	for _, c := range counts {
		if c > top {
			top = c
		}
	}
	return float64(escT) / float64(top), true
}

// formatInv compactly formats an inverse probability.
func formatInv(f float64) string {
	if f >= 1000 {
		return "1/" + itoa(int(f/1000)) + "k"
	}
	return "1/" + itoa(int(f))
}

// randStateConcrete is a thin wrapper over the concrete type to satisfy the
// interface above. Assumes r is *rand.Rand (guaranteed by call sites).
func randStateConcrete(r interface {
	Uint64() uint64
	Intn(int) int
}) Twister.State1024 {
	var s Twister.State1024
	for i := range s {
		s[i] = r.Uint64()
	}
	return s
}
