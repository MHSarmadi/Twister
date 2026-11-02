package huge

import (
	"math"
	"math/bits"
	"math/rand"
	"testing"

	"github.com/MHSarmadi/Twister"
)

// ---------------------------------------------------------------------------
// Reference implementations (unexported methods are not accessible from this
// package). Two ref-match tests at the bottom guard against drift.
// ---------------------------------------------------------------------------

// QR indices per layer, same order as Twister's internal chain.
var qrIndices = [16][4]int{
	{0x0, 0x4, 0x8, 0xc}, {0x9, 0xd, 0x1, 0x5}, {0xe, 0x2, 0x6, 0xa}, {0xf, 0x3, 0x7, 0xb},
	{0x0, 0xd, 0x7, 0xa}, {0xb, 0x4, 0xe, 0x1}, {0xf, 0x2, 0x5, 0x8}, {0x6, 0x3, 0x9, 0xc},
	{0x0, 0x3, 0x1, 0x2}, {0x4, 0x6, 0x5, 0x7}, {0xb, 0x9, 0x8, 0xa}, {0xd, 0xf, 0xe, 0xc},
	{0x0, 0xf, 0x5, 0xa}, {0xb, 0x6, 0xc, 0x1}, {0x2, 0xd, 0x7, 0x8}, {0x9, 0x4, 0xe, 0x3},
}

// State512 QR constants (matching 512.go naming; a←c0_hi etc.)
const (
	c512_a uint32 = 0x74657221 // XORed into a (=c0_hi in Twister source)
	c512_b uint32 = 0x54776973 // XORed into b (=c0_lo)
	c512_c uint32 = 0x6d616469 // XORed into c (=c1_hi)
	c512_d uint32 = 0x20536172 // XORed into d (=c1_lo)
)

// State1024 QR constants
const (
	c1024_a uint64 = 0x5477697374657221 // XORed into a
	c1024_b uint64 = 0x205361726d616469 // XORed into b
)

// Inner MARX QR on State512 — mirrors State512.quarter_marx_ab2ab.
func qr512(s Twister.State512, i [4]int) Twister.State512 {
	a, b, c, d := s[i[0]], s[i[1]], s[i[2]], s[i[3]]

	a ^= c512_a
	b ^= c512_b
	c ^= c512_c
	d ^= c512_d

	mixture := a ^ b ^ c ^ d
	a, b, c, d = mixture^d, mixture^c, mixture^b, mixture^a

	x, y := bits.Mul32(a, b)
	a = bits.RotateLeft32(a, 16)
	b = bits.RotateLeft32(b, 16)
	c ^= x
	d ^= y

	x, y = bits.Mul32(b, c)
	a ^= x
	c = bits.RotateLeft32(c, 16)
	d += y

	x, y = bits.Mul32(a, d)
	b ^= x
	c += y
	d = bits.RotateLeft32(d, 16)

	x, y = bits.Mul32(c, d)
	a += x
	b += y

	mixture = a + b + c + d
	s[i[0]], s[i[1]], s[i[2]], s[i[3]] = mixture-d, mixture-c, mixture-b, mixture-a
	return s
}

// Inner MARX QR on State1024 — mirrors State1024.quarter_marx_ab2ab.
func qr1024(s Twister.State1024, i [4]int) Twister.State1024 {
	a, b, c, d := s[i[0]], s[i[1]], s[i[2]], s[i[3]]

	a ^= c1024_a
	b ^= c1024_b

	mixture := a ^ b ^ c ^ d
	a, b, c, d = mixture^d, mixture^c, mixture^b, mixture^a

	x, y := bits.Mul64(a, b)
	a = bits.RotateLeft64(a, 32)
	b = bits.RotateLeft64(b, 32)
	c ^= x
	d ^= y

	x, y = bits.Mul64(b, c)
	a ^= x
	c = bits.RotateLeft64(c, 32)
	d += y

	x, y = bits.Mul64(a, d)
	b ^= x
	c += y
	d = bits.RotateLeft64(d, 32)

	x, y = bits.Mul64(c, d)
	a += x
	b += y

	mixture = a + b + c + d
	s[i[0]], s[i[1]], s[i[2]], s[i[3]] = mixture-d, mixture-c, mixture-b, mixture-a
	return s
}

// permute512N applies n super-rounds (n × 16 QRs) of the State512 chain.
func permute512N(s Twister.State512, n int) Twister.State512 {
	for r := 0; r < n; r++ {
		for j := 0; j < 16; j++ {
			s = qr512(s, qrIndices[j])
		}
	}
	return s
}

// permute1024N applies n super-rounds (n × 16 QRs) of the State1024 chain.
func permute1024N(s Twister.State1024, n int) Twister.State1024 {
	for r := 0; r < n; r++ {
		for j := 0; j < 16; j++ {
			s = qr1024(s, qrIndices[j])
		}
	}
	return s
}

// Outer Feistel constants (matching 8192.go / 16384.go)
const (
	c0f_hi uint32 = 0x46656973
	c0f_lo uint32 = 0x74656c45
	c1f_hi uint32 = 0x7870616e
	c1f_lo uint32 = 0x73696f6e

	c0f_1024 uint64 = 0x4665697374656c45
	c1f_1024 uint64 = 0x7870616e73696f6e
)

// quarterFeistel8192 mirrors State8192.quarter_feistel_ab2ab.
// Rounds 1-2 use XOR, rounds 3-4 use ADD (Lai-Massey half).
func quarterFeistel8192(s Twister.State8192, i [4]int) Twister.State8192 {
	a, b, c, d := s[i[0]], s[i[1]], s[i[2]], s[i[3]]

	a[0] += c0f_hi
	b[1] += c0f_lo
	c[2] += c1f_hi
	d[3] += c1f_lo

	// Round 1: XOR
	for k := range a {
		c[k] ^= a[k]
	}
	for k := range b {
		d[k] ^= b[k]
	}
	c = permute512N(c, 1)
	d = permute512N(d, 1)

	// Round 2: XOR
	for k := range d {
		a[k] ^= d[k]
	}
	for k := range c {
		b[k] ^= c[k]
	}
	a = permute512N(a, 1)
	b = permute512N(b, 1)

	// Round 3: ADD
	for k := range b {
		c[k] += b[k]
	}
	for k := range a {
		d[k] += a[k]
	}
	c = permute512N(c, 1)
	d = permute512N(d, 1)

	// Round 4: ADD
	for k := range c {
		a[k] += c[k]
	}
	for k := range d {
		b[k] += d[k]
	}
	a = permute512N(a, 1)
	b = permute512N(b, 1)

	s[i[0]], s[i[1]], s[i[2]], s[i[3]] = a, b, c, d
	return s
}

// quarterFeistel16384 mirrors State16384.quarter_feistel_ab2ab.
func quarterFeistel16384(s Twister.State16384, i [4]int) Twister.State16384 {
	a, b, c, d := s[i[0]], s[i[1]], s[i[2]], s[i[3]]

	a[0] += c0f_1024
	b[1] += c1f_1024
	c[2] += c1f_1024
	d[3] += c0f_1024

	// Round 1: XOR
	for k := range a {
		c[k] ^= a[k]
	}
	for k := range b {
		d[k] ^= b[k]
	}
	c = permute1024N(c, 1)
	d = permute1024N(d, 1)

	// Round 2: XOR
	for k := range d {
		a[k] ^= d[k]
	}
	for k := range c {
		b[k] ^= c[k]
	}
	a = permute1024N(a, 1)
	b = permute1024N(b, 1)

	// Round 3: ADD
	for k := range b {
		c[k] += b[k]
	}
	for k := range a {
		d[k] += a[k]
	}
	c = permute1024N(c, 1)
	d = permute1024N(d, 1)

	// Round 4: ADD
	for k := range c {
		a[k] += c[k]
	}
	for k := range d {
		b[k] += d[k]
	}
	a = permute1024N(a, 1)
	b = permute1024N(b, 1)

	s[i[0]], s[i[1]], s[i[2]], s[i[3]] = a, b, c, d
	return s
}

// applyCalls8192 applies n outer QRs. n=64 == one full Permute().
func applyCalls8192(s Twister.State8192, n int) Twister.State8192 {
	for i := 0; i < n; i++ {
		s = quarterFeistel8192(s, qrIndices[i%16])
	}
	return s
}

// applyCalls16384 applies n outer QRs. n=64 == one full Permute().
func applyCalls16384(s Twister.State16384, n int) Twister.State16384 {
	for i := 0; i < n; i++ {
		s = quarterFeistel16384(s, qrIndices[i%16])
	}
	return s
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func rngSeed(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

func rand8192(r *rand.Rand) Twister.State8192 {
	var s Twister.State8192
	for i := range s {
		for j := range s[i] {
			s[i][j] = r.Uint32()
		}
	}
	return s
}

func rand16384(r *rand.Rand) Twister.State16384 {
	var s Twister.State16384
	for i := range s {
		for j := range s[i] {
			s[i][j] = r.Uint64()
		}
	}
	return s
}

func hw8192(s Twister.State8192) int {
	n := 0
	for _, c := range s {
		for _, w := range c {
			n += bits.OnesCount32(w)
		}
	}
	return n
}

func hw16384(s Twister.State16384) int {
	n := 0
	for _, c := range s {
		for _, w := range c {
			n += bits.OnesCount64(w)
		}
	}
	return n
}

func xor8192(a, b Twister.State8192) Twister.State8192 {
	var c Twister.State8192
	for i := range a {
		for j := range a[i] {
			c[i][j] = a[i][j] ^ b[i][j]
		}
	}
	return c
}

func xor16384(a, b Twister.State16384) Twister.State16384 {
	var c Twister.State16384
	for i := range a {
		for j := range a[i] {
			c[i][j] = a[i][j] ^ b[i][j]
		}
	}
	return c
}

func flip8192(s *Twister.State8192, bit int) {
	c := bit / 512
	b := bit % 512
	s[c][b/32] ^= uint32(1) << uint(b%32)
}

func flip16384(s *Twister.State16384, bit int) {
	c := bit / 1024
	b := bit % 1024
	s[c][b/64] ^= uint64(1) << uint(b%64)
}

// ---------------------------------------------------------------------------
// Ref-match guards — MUST pass first, otherwise every other number is wrong.
// ---------------------------------------------------------------------------

func Test8192RefMatchesPermute(t *testing.T) {
	r := rngSeed(0x516192)
	for i := 0; i < 10; i++ {
		s := rand8192(r)
		got := s
		got.Permute()
		want := applyCalls8192(s, 32)
		if got != want {
			t.Fatalf("ref diverges from State8192.Permute() on sample %d", i)
		}
	}
	t.Logf("ref matches State8192.Permute() on 10/10 states")
}

func Test16384RefMatchesPermute(t *testing.T) {
	r := rngSeed(0x51616384)
	for i := 0; i < 10; i++ {
		s := rand16384(r)
		got := s
		got.Permute()
		want := applyCalls16384(s, 32)
		if got != want {
			t.Fatalf("ref diverges from State16384.Permute() on sample %d", i)
		}
	}
	t.Logf("ref matches State16384.Permute() on 10/10 states")
}

// ---------------------------------------------------------------------------
// State8192 tests
// ---------------------------------------------------------------------------

func Test8192Bijective(t *testing.T) {
	r := rngSeed(0xba8192)
	T := 100
	seen := make(map[Twister.State8192]bool, T)
	for i := 0; i < T; i++ {
		s := rand8192(r)
		s.Permute()
		if seen[s] {
			t.Fatalf("collision on %d", i)
		}
		seen[s] = true
	}
	t.Logf("%d/%d distinct outputs", T, T)
}

func Test8192ZeroState(t *testing.T) {
	var s Twister.State8192
	s.Permute()
	nz := 0
	for i := 0; i < 16; i++ {
		for _, w := range s[i] {
			if w != 0 {
				nz++
				break
			}
		}
	}
	if nz < 16 {
		t.Errorf("only %d/16 chunks nonzero", nz)
	}
	t.Logf("Permute(0): %d/16 chunks nonzero", nz)
}

func Test8192Avalanche(t *testing.T) {
	r := rngSeed(0xa8192)
	T := 100
	checkpoints := []int{1, 2, 3, 4, 8, 16, 32}
	t.Logf("Avalanche (T=%d, ideal 4096.0 / 45.3):", T)
	for _, n := range checkpoints {
		sum, sum2 := int64(0), int64(0)
		for tr := 0; tr < T; tr++ {
			s1 := rand8192(r)
			s2 := s1
			flip8192(&s2, r.Intn(8192))
			s1 = applyCalls8192(s1, n)
			s2 = applyCalls8192(s2, n)
			v := int64(hw8192(xor8192(s1, s2)))
			sum += v
			sum2 += v * v
		}
		mean := float64(sum) / float64(T)
		variance := float64(sum2)/float64(T) - mean*mean
		if variance < 0 {
			variance = 0
		}
		std := math.Sqrt(variance)
		pct := 100 * mean / 4096.0
		t.Logf("  n=%2d QRs: mean=%.2f (%.1f%%)  std=%.2f", n, mean, pct, std)
	}
}

func Test8192WordDiffusion(t *testing.T) {
	r := rngSeed(0xdd8192)
	T := 100
	for _, n := range []int{1, 2, 3, 4, 8, 16, 32} {
		sum := int64(0)
		for tr := 0; tr < T; tr++ {
			s1 := rand8192(r)
			s2 := s1
			flip8192(&s2, r.Intn(8192))
			s1 = applyCalls8192(s1, n)
			s2 = applyCalls8192(s2, n)
			d := xor8192(s1, s2)
			nz := 0
			for i := 0; i < 16; i++ {
				for _, w := range d[i] {
					if w != 0 {
						nz++
						break
					}
				}
			}
			sum += int64(nz)
		}
		t.Logf("  n=%2d QRs: %.2f/16 chunks active", n, float64(sum)/float64(T))
	}
}

func Test8192Slide(t *testing.T) {
	r := rngSeed(0x518192)
	T := 50
	min2 := 1 << 30
	for i := 0; i < T; i++ {
		x := rand8192(r)
		y := x
		y.Permute()
		y.Permute()
		if v := hw8192(xor8192(y, x)); v < min2 {
			min2 = v
		}
	}
	t.Logf("Slide: min HW(P²(x)⊕x)=%d (T=%d, ideal 4096)", min2, T)
}

// ---------------------------------------------------------------------------
// State16384 tests
// ---------------------------------------------------------------------------

func Test16384Bijective(t *testing.T) {
	r := rngSeed(0xba16384)
	T := 50
	seen := make(map[Twister.State16384]bool, T)
	for i := 0; i < T; i++ {
		s := rand16384(r)
		s.Permute()
		if seen[s] {
			t.Fatalf("collision on %d", i)
		}
		seen[s] = true
	}
	t.Logf("%d/%d distinct outputs", T, T)
}

func Test16384ZeroState(t *testing.T) {
	var s Twister.State16384
	s.Permute()
	nz := 0
	for i := 0; i < 16; i++ {
		for _, w := range s[i] {
			if w != 0 {
				nz++
				break
			}
		}
	}
	if nz < 16 {
		t.Errorf("only %d/16 chunks nonzero", nz)
	}
	t.Logf("Permute(0): %d/16 chunks nonzero", nz)
}

func Test16384Avalanche(t *testing.T) {
	r := rngSeed(0xa16384)
	T := 50
	checkpoints := []int{1, 2, 3, 4, 8, 16, 32}
	t.Logf("Avalanche (T=%d, ideal 8192.0 / 64.0):", T)
	for _, n := range checkpoints {
		sum, sum2 := int64(0), int64(0)
		for tr := 0; tr < T; tr++ {
			s1 := rand16384(r)
			s2 := s1
			flip16384(&s2, r.Intn(16384))
			s1 = applyCalls16384(s1, n)
			s2 = applyCalls16384(s2, n)
			v := int64(hw16384(xor16384(s1, s2)))
			sum += v
			sum2 += v * v
		}
		mean := float64(sum) / float64(T)
		variance := float64(sum2)/float64(T) - mean*mean
		if variance < 0 {
			variance = 0
		}
		std := math.Sqrt(variance)
		pct := 100 * mean / 8192.0
		t.Logf("  n=%2d QRs: mean=%.2f (%.1f%%)  std=%.2f", n, mean, pct, std)
	}
}

func Test16384WordDiffusion(t *testing.T) {
	r := rngSeed(0xdd16384)
	T := 50
	for _, n := range []int{1, 2, 3, 4, 8, 16, 32} {
		sum := int64(0)
		for tr := 0; tr < T; tr++ {
			s1 := rand16384(r)
			s2 := s1
			flip16384(&s2, r.Intn(16384))
			s1 = applyCalls16384(s1, n)
			s2 = applyCalls16384(s2, n)
			d := xor16384(s1, s2)
			nz := 0
			for i := 0; i < 16; i++ {
				for _, w := range d[i] {
					if w != 0 {
						nz++
						break
					}
				}
			}
			sum += int64(nz)
		}
		t.Logf("  n=%2d QRs: %.2f/16 chunks active", n, float64(sum)/float64(T))
	}
}

func Test16384Slide(t *testing.T) {
	r := rngSeed(0x5116384)
	T := 25
	min2 := 1 << 30
	for i := 0; i < T; i++ {
		x := rand16384(r)
		y := x
		y.Permute()
		y.Permute()
		if v := hw16384(xor16384(y, x)); v < min2 {
			min2 = v
		}
	}
	t.Logf("Slide: min HW(P²(x)⊕x)=%d (T=%d, ideal 8192)", min2, T)
}

// ---------------------------------------------------------------------------
// Seed constructors
// ---------------------------------------------------------------------------

func TestSeedConstructors(t *testing.T) {
	s8192 := Twister.NewState8192_32bitSeed(1)
	nz := 0
	for i := 0; i < 16; i++ {
		for _, w := range s8192[i] {
			if w != 0 {
				nz++
				break
			}
		}
	}
	if nz < 16 {
		t.Errorf("8192 seed: only %d/16 chunks nonzero", nz)
	}
	t.Logf("8192 seed 1: %d/16 chunks nonzero", nz)

	s16384 := Twister.NewState16384_64bitSeed(1)
	nz = 0
	for i := 0; i < 16; i++ {
		for _, w := range s16384[i] {
			if w != 0 {
				nz++
				break
			}
		}
	}
	if nz < 16 {
		t.Errorf("16384 seed: only %d/16 chunks nonzero", nz)
	}
	t.Logf("16384 seed 1: %d/16 chunks nonzero", nz)
}
