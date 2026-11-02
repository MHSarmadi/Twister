package benchmarks

import (
	"testing"

	"github.com/MHSarmadi/Twister"
)

func BenchmarkTwister16384Permute(b *testing.B) {
	var s Twister.State16384
	for i := range s {
		for j := range s[i] {
			s[i][j] = uint64(i*16+j)*0x9E3779B97F4A7C15 + 1
		}
	}
	b.SetBytes(2048) // 16384 bits = 2048 bytes
	
	for b.Loop() {
		s.Permute()
	}
	benchmarkSink16384 = s
}

func BenchmarkTwister16384PermuteParallel(b *testing.B) {
	b.SetBytes(2048)
	b.RunParallel(func(pb *testing.PB) {
		var s Twister.State16384
		for i := range s {
			for j := range s[i] {
				s[i][j] = uint64(i*16+j)*0x9E3779B97F4A7C15 + 1
			}
		}
		for pb.Next() {
			s.Permute()
		}
		benchmarkSink16384 = s
	})
}

var benchmarkSink16384 Twister.State16384
