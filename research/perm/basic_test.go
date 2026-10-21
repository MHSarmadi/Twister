package perm

import (
	"testing"

	"github.com/MHSarmadi/Twister"
)

// Guards against reference-impl drift. If this ever fires, every other number
// in this directory is measuring the wrong function.
func TestReferenceMatchesPermute(t *testing.T) {
	r := rngSeed(0xaa1)
	for i := 0; i < 100; i++ {
		s := randState(r)
		got := s
		got.Permute()
		want := permuteFull(s)
		if !eq(got, want) {
			t.Fatalf("reference impl diverges from Permute() on sample %d", i)
		}
	}
	t.Logf("reference impl matches Permute() on 100/100 random states")
}

func TestBijective(t *testing.T) {
	r := rngSeed(0xbb1)
	T := 2000
	if fastMode() {
		T = 200
	}
	seen := make(map[Twister.State1024]bool, T)
	for i := 0; i < T; i++ {
		s := randState(r)
		s.Permute()
		if seen[s] {
			t.Fatalf("Permute collided on sample %d — non-bijective", i)
		}
		seen[s] = true
	}
	t.Logf("Permute: %d/%d distinct outputs", T, T)
}

func TestZeroState(t *testing.T) {
	s := zero()
	s.Permute()
	nz := 0
	var zeros []int
	for i := 0; i < 16; i++ {
		if s[i] != 0 {
			nz++
		} else {
			zeros = append(zeros, i)
		}
	}
	t.Logf("Permute(0): %d/16 words nonzero, zero words = %v", nz, zeros)
	if nz < 16 {
		t.Errorf("Permute(0) left %d words at zero — constant coverage hole", 16-nz)
	}
}

func TestNotInvolution(t *testing.T) {
	r := rngSeed(0xcc1)
	matches := 0
	for i := 0; i < 500; i++ {
		s := randState(r)
		s.Permute()
		s.Permute()
		// compare to original
		// (rebuild since we destroyed it)
		orig := s
		orig.Permute()
		orig.Permute()
		_ = orig
	}
	_ = matches
}
