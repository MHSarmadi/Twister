package benchmarks

import (
	"math/bits"
	"testing"
)

// chacha20QR is the standard ChaCha20 quarter round, matching RFC 8439:
//
//	a += b; d ^= a; d <<<= 16;
//	c += d; b ^= c; b <<<= 12;
//	a += b; d ^= a; d <<<= 8;
//	c += d; b ^= c; b <<<= 7;
func chacha20QR(x *[16]uint32, a, b, c, d int) {
	x[a] += x[b]
	x[d] ^= x[a]
	x[d] = bits.RotateLeft32(x[d], 16)

	x[c] += x[d]
	x[b] ^= x[c]
	x[b] = bits.RotateLeft32(x[b], 12)

	x[a] += x[b]
	x[d] ^= x[a]
	x[d] = bits.RotateLeft32(x[d], 8)

	x[c] += x[d]
	x[b] ^= x[c]
	x[b] = bits.RotateLeft32(x[b], 7)
}

// chacha20Block performs the standard 20-round ChaCha20 block function
// (10 double rounds) plus the final feed-forward addition of the input
// state. This is the exact function used inside ChaCha20 (RFC 8439 §2.3).
func chacha20Block(in [16]uint32) [16]uint32 {
	x := in

	for i := 0; i < 10; i++ {
		// column rounds
		chacha20QR(&x, 0, 4, 8, 12)
		chacha20QR(&x, 1, 5, 9, 13)
		chacha20QR(&x, 2, 6, 10, 14)
		chacha20QR(&x, 3, 7, 11, 15)
		// diagonal rounds
		chacha20QR(&x, 0, 5, 10, 15)
		chacha20QR(&x, 1, 6, 11, 12)
		chacha20QR(&x, 2, 7, 8, 13)
		chacha20QR(&x, 3, 4, 9, 14)
	}

	for i := range x {
		x[i] += in[i]
	}
	return x
}

// BenchmarkChaCha20Block measures the full 512-bit block function.
// SetBytes(64) = 512 bits per op.
func BenchmarkChaCha20Block(b *testing.B) {
	var in [16]uint32
	for i := range in {
		in[i] = uint32(i) * 0x9E3779B9
	}
	b.SetBytes(64) // 512 bits = 64 bytes
	
	for b.Loop() {
		benchmarkSinkChacha = chacha20Block(in)
	}
}

// BenchmarkChaCha20BlockParallel measures multi-core throughput.
func BenchmarkChaCha20BlockParallel(b *testing.B) {
	b.SetBytes(64)
	b.RunParallel(func(pb *testing.PB) {
		var in [16]uint32
		for i := range in {
			in[i] = uint32(i) * 0x9E3779B9
		}
		for pb.Next() {
			benchmarkSinkChacha = chacha20Block(in)
		}
	})
}

var benchmarkSinkChacha [16]uint32
