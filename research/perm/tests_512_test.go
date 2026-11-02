package perm

import (
	"math"
	"math/rand"
	"testing"

	"github.com/MHSarmadi/Twister"
)

func TestReferenceMatchesPermute512(t *testing.T) {
	r := rngSeed(0xa512)
	for i := 0; i < 100; i++ {
		s := randState512(r)
		got := s
		got.Permute()
		want := permuteFull512(s)
		if !eq512(got, want) {
			t.Fatalf("reference impl diverges from State512.Permute() on sample %d", i)
		}
	}
	t.Logf("reference matches State512.Permute() on 100/100 states")
}

func TestBijective512(t *testing.T) {
	r := rngSeed(0xb512)
	T := 2000
	if fastMode() {
		T = 200
	}
	seen := make(map[Twister.State512]bool, T)
	for i := 0; i < T; i++ {
		s := randState512(r)
		s.Permute()
		if seen[s] {
			t.Fatalf("collision on sample %d", i)
		}
		seen[s] = true
	}
	t.Logf("%d/%d distinct outputs", T, T)
}

func TestZeroState512(t *testing.T) {
	var s Twister.State512
	s.Permute()
	nz := 0
	for i := 0; i < 16; i++ {
		if s[i] != 0 {
			nz++
		}
	}
	if nz < 16 {
		t.Errorf("Permute(0): only %d/16 nonzero words", nz)
	}
	t.Logf("Permute(0): %d/16 words nonzero", nz)
}

func TestAvalanche512(t *testing.T) {
	r := rngSeed(0xaa512)
	T := 500
	if fastMode() {
		T = 100
	}
	t.Logf("Avalanche: 1-bit diff, T=%d, ideal 256.0 / 8.0", T)
	for _, n := range []int{1, 2, 3, 4} {
		sum, sum2 := int64(0), int64(0)
		for tr := 0; tr < T; tr++ {
			s1 := randState512(r)
			s2 := s1
			flip512(&s2, r.Intn(512))
			s1 = applyPermuteCalls512(s1, n)
			s2 = applyPermuteCalls512(s2, n)
			v := int64(hw512(xor512(s1, s2)))
			sum += v
			sum2 += v * v
		}
		mean := float64(sum) / float64(T)
		variance := float64(sum2)/float64(T) - mean*mean
		if variance < 0 {
			variance = 0
		}
		std := math.Sqrt(variance)
		t.Logf("  after %d calls: mean HW=%.2f  std=%.2f", n, mean, std)
	}
}

func TestWordDiffusion512(t *testing.T) {
	r := rngSeed(0xdd512)
	T := 300
	if fastMode() {
		T = 50
	}
	for _, n := range []int{1, 2, 3, 4} {
		sum := int64(0)
		for tr := 0; tr < T; tr++ {
			s1 := randState512(r)
			s2 := s1
			flip512(&s2, r.Intn(512))
			s1 = applyPermuteCalls512(s1, n)
			s2 = applyPermuteCalls512(s2, n)
			d := xor512(s1, s2)
			nz := 0
			for i := 0; i < 16; i++ {
				if d[i] != 0 {
					nz++
				}
			}
			sum += int64(nz)
		}
		t.Logf("  after %d calls: %.2f/16 words active", n, float64(sum)/float64(T))
	}
}

func TestTrailDecay512(t *testing.T) {
	r := rngSeed(0x7a512)
	T := 5000
	if fastMode() {
		T = 1000
	}
	bits := []int{0, 100, 400}
	t.Logf("Trail (T=%d): top-1 per bit, per QR depth", T)
	for _, bit := range bits {
		line := ""
		for _, n := range []int{1, 2, 4} {
			counts := make(map[Twister.State512]int, T)
			for tr := 0; tr < T; tr++ {
				x := randState512(r)
				y := x
				flip512(&y, bit)
				x = applyQRs512(x, n)
				y = applyQRs512(y, n)
				counts[xor512(x, y)]++
			}
			top := 0
			for _, c := range counts {
				if c > top {
					top = c
				}
			}
			line += " " + formatInv(float64(T)/float64(top))
		}
		t.Logf("  bit %4d: %s", bit, line)
	}
}

func TestTwoPhase512(t *testing.T) {
	r := rngSeed(0x90512)
	maxT := 50000
	if fastMode() {
		maxT = 10000
	}
	t.Logf("Two-phase trail (maxT=%d):", maxT)
	for _, bit := range []int{0, 400} {
		inv, sat := measureTwoPhase512(r, bit, maxT)
		tag := ""
		if sat {
			tag = " (saturated)"
		}
		t.Logf("  bit %d: 2-phase top-1 ≥ 1/%.0f%s", bit, inv, tag)
	}
}

func measureTwoPhase512(r *rand.Rand, bit, maxT int) (float64, bool) {
	counts := make(map[Twister.State512]int)
	const chunk = 5000
	for done := 0; done < maxT; done += chunk {
		for tr := 0; tr < chunk; tr++ {
			x := randState512(r)
			y := x
			flip512(&y, bit)
			x = applyPermuteCalls512(x, 2)
			y = applyPermuteCalls512(y, 2)
			counts[xor512(x, y)]++
		}
		top := 0
		for _, c := range counts {
			if c > top {
				top = c
			}
		}
		total := done + chunk
		if top >= 5 {
			return float64(total) / float64(top), false
		}
		if total >= maxT {
			return float64(total) / float64(top), true
		}
	}
	return 1, true
}

func TestDifferential512(t *testing.T) {
	r := rngSeed(0xdd3 + 512)
	T := 10000
	if fastMode() {
		T = 2000
	}
	for _, bit := range []int{0, 100, 400} {
		counts := make(map[Twister.State512]int, T)
		for tr := 0; tr < T; tr++ {
			x := randState512(r)
			y := x
			flip512(&y, bit)
			x.Permute()
			y.Permute()
			counts[xor512(x, y)]++
		}
		top := 0
		for _, c := range counts {
			if c > top {
				top = c
			}
		}
		t.Logf("  bit %d: full Permute top-1 = 1/%.0f", bit, float64(T)/float64(top))
	}
}

func TestLinear512(t *testing.T) {
	r := rngSeed(0x11f + 512)
	T := 3000
	if fastMode() {
		T = 500
	}
	const numIn = 20
	const numOut = 20
	inBits := sampleBits512(r, 512, numIn)
	outBits := sampleBits512(r, 512, numOut)
	inMasks := make([]Twister.State512, numIn)
	for i, bit := range inBits {
		flip512(&inMasks[i], bit)
	}
	outMasks := make([]Twister.State512, numOut)
	for i, bit := range outBits {
		flip512(&outMasks[i], bit)
	}

	samples := make([]Twister.State512, T)
	outputs := make([]Twister.State512, T)
	for i := 0; i < T; i++ {
		samples[i] = randState512(r)
		outputs[i] = samples[i]
		outputs[i].Permute()
	}

	noise := 1.0 / math.Sqrt(float64(T))
	maxBias := 0.0
	var maxInBit, maxOutBit int
	for ii := 0; ii < numIn; ii++ {
		for oi := 0; oi < numOut; oi++ {
			sum := 0
			for j := 0; j < T; j++ {
				b := parity512(samples[j], inMasks[ii]) ^ parity512(outputs[j], outMasks[oi])
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
	t.Logf("Linear: T=%d noise=%.4f, max|bias|=%.4f (%.2f× noise) at in=%d out=%d",
		T, noise, maxBias, maxBias/noise, maxInBit, maxOutBit)
}

func TestSlide512(t *testing.T) {
	r := rngSeed(0x51d + 512)
	T := 500
	if fastMode() {
		T = 100
	}
	min2 := 1 << 30
	min3 := 1 << 30
	for i := 0; i < T; i++ {
		x := randState512(r)
		y := x
		y.Permute()
		y.Permute()
		if v := hw512(xor512(y, x)); v < min2 {
			min2 = v
		}
		y = x
		y.Permute()
		y.Permute()
		y.Permute()
		if v := hw512(xor512(y, x)); v < min3 {
			min3 = v
		}
	}
	t.Logf("Slide: min HW(P²(x)⊕x)=%d  min HW(P³(x)⊕x)=%d (T=%d)", min2, min3, T)
}

func TestRotational512(t *testing.T) {
	r := rngSeed(0x707 + 512)
	T := 100
	if fastMode() {
		T = 20
	}
	minHW := 1 << 30
	worstR := 0
	for rot := 1; rot < 32; rot++ {
		for tr := 0; tr < T; tr++ {
			x := randState512(r)
			xr := rotlState512(x, rot)
			px := x
			px.Permute()
			pxr := rotlState512(px, rot)
			xr.Permute()
			diff := hw512(xor512(xr, pxr))
			if diff < minHW {
				minHW = diff
				worstR = rot
			}
		}
	}
	t.Logf("Rotational: min HW=%d at r=%d (T=%d)", minHW, worstR, T)
}

func TestIntegral512(t *testing.T) {
	r := rngSeed(0x1e6 + 512)
	trials := 4
	if fastMode() {
		trials = 1
	}
	for _, k := range []int{8, 10} {
		for _, numCalls := range []int{1, 2, 4} {
			allZero := 0
			for tr := 0; tr < trials; tr++ {
				basis := make([]int, k)
				used := map[int]bool{}
				for b := 0; b < k; b++ {
					for {
						cand := r.Intn(512)
						if !used[cand] {
							used[cand] = true
							basis[b] = cand
							break
						}
					}
				}
				var acc Twister.State512
				for m := 0; m < (1 << k); m++ {
					x := randState512(r)
					for b := 0; b < k; b++ {
						if m&(1<<b) != 0 {
							flip512(&x, basis[b])
						}
					}
					x = applyPermuteCalls512(x, numCalls)
					acc = xor512(acc, x)
				}
				if hw512(acc) == 0 {
					allZero++
				}
			}
			tag := ""
			if allZero > 0 {
				tag = "  >> WARNING"
			}
			t.Logf("  cube 2^%d, %d calls: zero-sum %d/%d%s", k, numCalls, allZero, trials, tag)
		}
	}
}

func TestCD512(t *testing.T) {
	r := rngSeed(0xc0d + 512)
	T := 300
	if fastMode() {
		T = 100
	}
	cases := []struct {
		name string
		mk   func(x, y uint32) Twister.State512
	}{
		{"[0,0,x,y]", func(x, y uint32) Twister.State512 {
			var s Twister.State512
			s[2], s[3] = x, y
			return s
		}},
		{"[0,0,x,x]", func(x, _ uint32) Twister.State512 {
			var s Twister.State512
			s[2], s[3] = x, x
			return s
		}},
		{"[x,x,x,x]", func(x, _ uint32) Twister.State512 {
			var s Twister.State512
			s[0], s[1], s[2], s[3] = x, x, x, x
			return s
		}},
	}
	baseline := measureAvalFull512(r, T, func() Twister.State512 { return randState512(r) })
	t.Logf("Avalanche full Permute (ideal 256, σ 8): baseline=%.2f", baseline)
	for _, c := range cases {
		avg := measureAvalFull512(r, T, func() Twister.State512 {
			return c.mk(r.Uint32(), r.Uint32())
		})
		dev := (avg - 256.0) / 8.0
		tag := ""
		if dev < -3 {
			tag = "  >> WARNING"
		}
		t.Logf("  %-10s mean HW=%.2f (%+.2f σ)%s", c.name, avg, dev, tag)
	}
}

func measureAvalFull512(r *rand.Rand, T int, mk func() Twister.State512) float64 {
	sum := int64(0)
	for i := 0; i < T; i++ {
		s1 := mk()
		s2 := s1
		flip512(&s2, r.Intn(512))
		s1.Permute()
		s2.Permute()
		sum += int64(hw512(xor512(s1, s2)))
	}
	return float64(sum) / float64(T)
}
