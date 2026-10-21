package perm

import (
	"testing"

	"github.com/MHSarmadi/Twister"
)

// TestTrailDecay measures top-1 differential probability after N QRs.
// The interesting regime is 1–4 QRs; beyond that the trail saturates.
func TestTrailDecay(t *testing.T) {
	r := rngSeed(0x7a1)
	T := 5000
	if fastMode() {
		T = 1000
	}
	bits := []int{0, 100, 400, 700, 1000}

	t.Logf("Trail decay: bit → top-1 after N QRs, T=%d", T)
	for _, bit := range bits {
		line := "bit " + itoa(bit) + ":"
		for _, n := range []int{1, 2, 4, 8, 16, 32, 64} {
			counts := make(map[Twister.State1024]int, T)
			for tr := 0; tr < T; tr++ {
				x := randState(r)
				y := x
				flip(&y, bit)
				x = applyQRs(x, n)
				y = applyQRs(y, n)
				counts[xor(x, y)]++
			}
			top := 0
			for _, c := range counts {
				if c > top {
					top = c
				}
			}
			inv := float64(T) / float64(top)
			line += " 1/" + ftoa(inv)
		}
		t.Logf("%s", line)
	}
}

// TestTwoPhaseAdaptive measures two-phase trail probability with escalation
// until the top output diff repeats ≥ 5 times, or maxT reached.
func TestTwoPhaseAdaptive(t *testing.T) {
	type pair struct {
		name   string
		f1, f2 func(Twister.State1024) Twister.State1024
	}
	pairs := []pair{
		{"col→rdiag", permuteCol2RDiag, permuteRDiag2Row},
		{"rdiag→row", permuteRDiag2Row, permuteRow2LDiag},
		{"row→ldiag", permuteRow2LDiag, permuteLDiag2Col},
		{"ldiag→col", permuteLDiag2Col, permuteCol2RDiag},
	}
	maxT := 100000
	if fastMode() {
		maxT = 20000
	}
	bits := []int{0, 700}

	t.Logf("Two-phase trail (escalated, maxT=%d):", maxT)
	for _, pp := range pairs {
		for _, bit := range bits {
			invProb, saturated := measureAdaptive(bit, pp.f1, pp.f2, maxT, int64(0x9000+bit))
			tag := ""
			if saturated {
				tag = " (saturated)"
			}
			t.Logf("  %-10s bit %4d: ≥ 1/%.0f%s", pp.name, bit, invProb, tag)
		}
	}
}

// measureAdaptive escalates in chunks until the top count reaches 5, or maxT.
// Returns the inverse probability (so higher is better) and whether it saturated.
func measureAdaptive(bit int, f1, f2 func(Twister.State1024) Twister.State1024,
	maxT int, seed int64) (float64, bool) {
	r := rngSeed(seed)
	counts := make(map[Twister.State1024]int)
	const chunk = 5000
	for done := 0; done < maxT; done += chunk {
		for tr := 0; tr < chunk; tr++ {
			x := randState(r)
			y := x
			flip(&y, bit)
			x = f1(x)
			x = f2(x)
			y = f1(y)
			y = f2(y)
			counts[xor(x, y)]++
		}
		top := 0
		for _, c := range counts {
			if c > top {
				top = c
			}
		}
		total := done + chunk
		if top >= 5 || total >= maxT {
			return float64(total) / float64(top), top < 5
		}
	}
	return 1, true
}

// TestCompositionRatio measures top-1 trail at each QR level and computes
// the observed vs. naive product. Meaningful at 1-4 QRs; saturates after.
func TestCompositionRatio(t *testing.T) {
	r := rngSeed(0xcc2)
	T := 5000
	if fastMode() {
		T = 1000
	}
	bits := []int{0, 700}

	for _, bit := range bits {
		probs := make([]float64, 5) // 1, 2, 4, 8, 16 QRs
		for i, n := range []int{1, 2, 4, 8, 16} {
			counts := make(map[Twister.State1024]int, T)
			for tr := 0; tr < T; tr++ {
				x := randState(r)
				y := x
				flip(&y, bit)
				x = applyQRs(x, n)
				y = applyQRs(y, n)
				counts[xor(x, y)]++
			}
			top := 0
			for _, c := range counts {
				if c > top {
					top = c
				}
			}
			probs[i] = float64(top) / float64(T)
		}
		t.Logf("bit %4d composition:", bit)
		t.Logf("  1 QR:  1/%.0f", 1/probs[0])
		t.Logf("  2 QRs: 1/%.0f  (vs naive 1/%.0f)  ratio=%.2f",
			1/probs[1], 1/(probs[0]*probs[0]), probs[1]/(probs[0]*probs[0]))
		t.Logf("  4 QRs: 1/%.0f  (vs naive 1/%.0f)  ratio=%.4f",
			1/probs[2], 1/(probs[0]*probs[0]*probs[0]*probs[0]),
			probs[2]/(probs[0]*probs[0]*probs[0]*probs[0]))
		t.Logf("  8 QRs: 1/%.0f", 1/probs[3])
		t.Logf("  16 QRs: 1/%.0f", 1/probs[4])
	}
}

// simple helpers to avoid importing strconv/fmt in this file
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func ftoa(f float64) string {
	// crude but adequate for log formatting
	if f < 10 {
		return itoa(int(f*10)) + "0"
	}
	return itoa(int(f))
}
