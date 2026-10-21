package perm

import (
	"math"
	"testing"
)

func TestAvalanche(t *testing.T) {
	r := rngSeed(0xaa2)
	T := 800
	if fastMode() {
		T = 200
	}
	checkpoints := []int{1, 2, 3, 4, 6, 8, 12, 16}

	t.Logf("Avalanche: 1-bit input diff, T=%d, ideal mean=512.0 std=16.0", T)
	for _, n := range checkpoints {
		sum, sum2 := int64(0), int64(0)
		for tr := 0; tr < T; tr++ {
			s1 := randState(r)
			s2 := s1
			flip(&s2, r.Intn(1024))
			s1 = applyPermuteCalls(s1, n)
			s2 = applyPermuteCalls(s2, n)
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
		t.Logf("after %2d permute-calls: mean HW=%7.2f  std=%6.2f", n, mean, std)
	}
}

// WordDiffusion reports how many of the 16 words have nonzero diff
// after N permute calls. This is distinct from Avalanche: it tells us
// when the diffusion has reached every word, not just how many total bits flipped.
func TestWordDiffusion(t *testing.T) {
	r := rngSeed(0xdd2)
	T := 500
	if fastMode() {
		T = 100
	}
	checkpoints := []int{1, 2, 3, 4}
	for _, n := range checkpoints {
		sumWords := int64(0)
		for tr := 0; tr < T; tr++ {
			s1 := randState(r)
			s2 := s1
			flip(&s2, r.Intn(1024))
			s1 = applyPermuteCalls(s1, n)
			s2 = applyPermuteCalls(s2, n)
			d := xor(s1, s2)
			nz := 0
			for i := 0; i < 16; i++ {
				if d[i] != 0 {
					nz++
				}
			}
			sumWords += int64(nz)
		}
		t.Logf("after %2d calls: %.2f/16 words active (avg)", n, float64(sumWords)/float64(T))
	}
}

// QRAvalanche measures avalanche at finer granularity — 1, 2, 4, 8 QRs —
// to show how quickly the 256-bit QR diffuses a 1-bit flip through the
// full 1024-bit state.
func TestQRAvalanche(t *testing.T) {
	r := rngSeed(0xee3)
	T := 500
	if fastMode() {
		T = 100
	}
	for _, n := range []int{1, 2, 4, 8, 16, 32, 64} {
		sum, sum2 := int64(0), int64(0)
		for tr := 0; tr < T; tr++ {
			s1 := randState(r)
			s2 := s1
			flip(&s2, r.Intn(1024))
			s1 = applyQRs(s1, n)
			s2 = applyQRs(s2, n)
			v := int64(hw(xor(s1, s2)))
			sum += v
			sum2 += v * v
		}
		mean := float64(sum) / float64(T)
		variance := float64(sum2)/float64(T) - mean*mean
		std := 0.0
		if variance > 0 {
			std = math.Sqrt(variance)
		}
		t.Logf("after %2d QRs: mean HW=%7.2f  std=%6.2f", n, mean, std)
	}
}
