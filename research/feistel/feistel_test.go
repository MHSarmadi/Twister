package feistel

import (
	"math"
	"math/bits"
	"math/rand"
	"testing"
)

func rngSeed(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

func randState(r *rand.Rand) State2048 {
	var s State2048
	for i := range s {
		s[i][0] = r.Uint32()
		s[i][1] = r.Uint32()
		s[i][2] = r.Uint32()
		s[i][3] = r.Uint32()
	}
	return s
}

func hw(s State2048) int {
	n := 0
	for _, w := range s {
		for _, v := range w {
			n += bits.OnesCount32(v)
		}
	}
	return n
}

func xor(a, b State2048) State2048 {
	var c State2048
	for i := range c {
		for j := range c[i] {
			c[i][j] = a[i][j] ^ b[i][j]
		}
	}
	return c
}

func flip(s *State2048, bit int) {
	chunk := bit / 128
	inner := bit % 128
	s[chunk][inner/32] ^= uint32(1) << uint(inner%32)
}

// firstTouchDepth returns the smallest n such that QR#n-1 touches chunk c.
func firstTouchDepth(chunk int) int {
	for i, idx := range qrIndices {
		for _, p := range idx {
			if p == chunk {
				return i + 1
			}
		}
	}
	return 999
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// Guards against reference drift: applyCalls(64) must equal Permute().
func TestRefMatchesPermute(t *testing.T) {
	r := rngSeed(0x516592)
	for i := 0; i < 100; i++ {
		s := randState(r)
		got := s
		got.Permute()
		want := s.applyCalls(64)
		if got != want {
			t.Fatalf("applyCalls(64) diverges from Permute() on sample %d", i)
		}
	}
	t.Logf("applyCalls(64) matches Permute() on 100/100 states")
}

func TestBijective(t *testing.T) {
	r := rngSeed(0xb1)
	T := 500
	seen := make(map[State2048]bool, T)
	for i := 0; i < T; i++ {
		s := randState(r)
		s.Permute()
		if seen[s] {
			t.Fatalf("collision on sample %d", i)
		}
		seen[s] = true
	}
	t.Logf("%d/%d distinct outputs", T, T)
}

func TestZeroState(t *testing.T) {
	var s State2048
	s.Permute()
	nzChunks := 0
	for i := 0; i < 16; i++ {
		if s[i][0] != 0 || s[i][1] != 0 || s[i][2] != 0 || s[i][3] != 0 {
			nzChunks++
		}
	}
	if nzChunks < 16 {
		t.Errorf("Permute(0): only %d/16 chunks nonzero", nzChunks)
	}
	t.Logf("Permute(0): %d/16 chunks nonzero", nzChunks)
}

// TestAvalancheExtended — the key new test.
// Shows convergence to 50% as the number of outer QRs increases.
func TestAvalancheExtended(t *testing.T) {
	r := rngSeed(0xaf2)
	T := 300
	t.Logf("Avalanche extended: T=%d, ideal 1024.0 / 22.6", T)
	checkpoints := []int{1, 2, 3, 4, 8, 12, 16, 24, 32, 48, 64}
	for _, n := range checkpoints {
		sum, sum2 := int64(0), int64(0)
		for tr := 0; tr < T; tr++ {
			s1 := randState(r)
			s2 := s1
			flip(&s2, r.Intn(2048))
			s1 = s1.applyCalls(n)
			s2 = s2.applyCalls(n)
			v := int64(hw(xor(s1, s2)))
			sum += v
			sum2 += v * v
		}
		mean := float64(sum) / float64(T)
		variance := float64(sum2)/float64(T) - mean*mean
		if variance < 0 {
			variance = 0
		}
		std := math.Sqrt(variance)
		pct := 100 * mean / 1024.0
		t.Logf("  n=%2d QRs: mean=%.2f (%.1f%%)  std=%.2f", n, mean, pct, std)
	}
}

// TestTrailExtended — filters by "first touch" so 1/1 cells only appear
// for bits genuinely untouched by the applied QRs (expected, not a bug).
func TestTrailExtended(t *testing.T) {
	r := rngSeed(0x7f2)
	T := 500
	bits := []int{0, 200, 400, 600, 800, 1000, 1200, 1400, 1600, 1800, 2000}
	depths := []int{4, 8, 16, 32, 48, 64}

	t.Logf("Trail extended (T=%d):", T)
	t.Logf("%-10s %-6s %-8s | %s", "bit", "chunk", "first@", "top-1 at depths 4/8/16/32/48/64")
	for _, bit := range bits {
		chunk := bit / 128
		ft := firstTouchDepth(chunk)
		line := ""
		for _, n := range depths {
			if n < ft {
				line += "   --   "
				continue
			}
			counts := make(map[State2048]int, T)
			for tr := 0; tr < T; tr++ {
				x := randState(r)
				y := x
				flip(&y, bit)
				x = x.applyCalls(n)
				y = y.applyCalls(n)
				counts[xor(x, y)]++
			}
			top := 0
			for _, c := range counts {
				if c > top {
					top = c
				}
			}
			line += " 1/" + itoa(T/top) + " "
		}
		t.Logf("%-10d %-6d %-8d |%s", bit, chunk, ft, line)
	}
}

func TestSlide(t *testing.T) {
	r := rngSeed(0x5f)
	T := 300
	min2 := 1 << 30
	min3 := 1 << 30
	for i := 0; i < T; i++ {
		x := randState(r)
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
	t.Logf("Slide: min HW(P²(x)⊕x)=%d  min HW(P³(x)⊕x)=%d (T=%d, ideal 1024)", min2, min3, T)
}
