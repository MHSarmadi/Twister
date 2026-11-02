package benchmarks

import (
	"testing"

	"github.com/MHSarmadi/Twister"
)

func BenchmarkTwister8192Permute(b *testing.B) {
	var s Twister.State8192
	for i := range s {
		for j := range s[i] {
			s[i][j] = uint32(i*16+j)*0x9E3779B9 + 1
		}
	}
	b.SetBytes(1024) // 8192 bits = 1024 bytes
	
	for b.Loop() {
		s.Permute()
	}
	benchmarkSink8192 = s
}

func BenchmarkTwister8192PermuteParallel(b *testing.B) {
	b.SetBytes(1024)
	b.RunParallel(func(pb *testing.PB) {
		var s Twister.State8192
		for i := range s {
			for j := range s[i] {
				s[i][j] = uint32(i*16+j)*0x9E3779B9 + 1
			}
		}
		for pb.Next() {
			s.Permute()
		}
		benchmarkSink8192 = s
	})
}

var benchmarkSink8192 Twister.State8192
