package benchmarks

import (
	"testing"

	"github.com/MHSarmadi/Twister"
)

func BenchmarkTwister512Permute(b *testing.B) {
	var s Twister.State512
	for i := range s {
		s[i] = uint32(i) * 0x9E3779B9
	}
	b.SetBytes(64) // 512 bits = 64 bytes
	
	for b.Loop() {
		s.Permute()
	}
	benchmarkSink512 = s
}

func BenchmarkTwister512PermuteParallel(b *testing.B) {
	b.SetBytes(64)
	b.RunParallel(func(pb *testing.PB) {
		var s Twister.State512
		for i := range s {
			s[i] = uint32(i) * 0x9E3779B9
		}
		for pb.Next() {
			s.Permute()
		}
		benchmarkSink512 = s
	})
}

var benchmarkSink512 Twister.State512
