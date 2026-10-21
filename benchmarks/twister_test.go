package benchmarks

import (
	"testing"

	"github.com/MHSarmadi/Twister"
)

// BenchmarkTwisterPermute measures the full 1024-bit Permute.
// SetBytes(128) = 1024 bits per op, so MB/s is directly comparable
// to ChaCha20's block function on a per-state-output basis.
func BenchmarkTwisterPermute(b *testing.B) {
	var s Twister.State1024
	for i := range s {
		s[i] = uint64(i) * 0x9E3779B97F4A7C15
	}
	b.SetBytes(128) // 1024 bits = 128 bytes
	b.ResetTimer()
	for b.Loop() {
		s.Permute()
	}
	benchmarkSink1024 = s // prevent dead-code elimination
}

// BenchmarkTwisterPermuteParallel measures multi-core throughput.
func BenchmarkTwisterPermuteParallel(b *testing.B) {
	b.SetBytes(128)
	b.RunParallel(func(pb *testing.PB) {
		var s Twister.State1024
		for i := range s {
			s[i] = uint64(i) * 0x9E3779B97F4A7C15
		}
		for pb.Next() {
			s.Permute()
		}
		benchmarkSink1024 = s
	})
}

var benchmarkSink1024 Twister.State1024
