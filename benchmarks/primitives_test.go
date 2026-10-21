package benchmarks

import (
	"math/bits"
	"testing"
)

// BenchmarkMul64 measures the raw cost of one 64-bit integer multiplication
// (hi + lo). This establishes the theoretical floor for Twister's 256 muls
// per Permute.
func BenchmarkMul64(b *testing.B) {
	x := uint64(0xDEADBEEFCAFEBABE)
	y := uint64(0x9E3779B97F4A7C15)
	var hi, lo uint64
	b.ResetTimer()
	for b.Loop() {
		hi, lo = bits.Mul64(x, y)
		x, y = hi, lo
	}
	benchSinkU64 = x ^ y
}

// BenchmarkAdd64 measures the raw cost of one 64-bit modular addition
// (Twister has 704 adds per Permute).
func BenchmarkAdd64(b *testing.B) {
	x := uint64(0xDEADBEEFCAFEBABE)
	y := uint64(0x9E3779B97F4A7C15)
	
	for b.Loop() {
		x += y
		y ^= x
	}
	benchSinkU64 = x ^ y
}

// BenchmarkXor64 measures a single XOR (832 per Permute).
func BenchmarkXor64(b *testing.B) {
	x := uint64(0xDEADBEEFCAFEBABE)
	y := uint64(0x9E3779B97F4A7C15)
	
	for b.Loop() {
		x ^= y
		y ^= x
	}
	benchSinkU64 = x ^ y
}

var benchSinkU64 uint64
