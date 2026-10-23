package mul

import "math/bits"

type State256 [4]uint64
type State128 [4]uint32

const c0, c1 uint64 = 0x5477697374657221, 0x205361726d616469

const c0_lo, c0_hi uint32 = 0x54776973, 0x74657221
const c1_lo, c1_hi uint32 = 0x20536172, 0x6d616469

func (s *State256) MultPermuter() {
	var x, y uint64

	// Injecting "Twister! Sarmadi"
	s[0] ^= c0
	s[1] ^= c1

	// Pre-round mixture
	mixture := s[0] ^ s[1] ^ s[2] ^ s[3]
	s[0], s[1], s[2], s[3] = mixture^s[3], mixture^s[2], mixture^s[1], mixture^s[0]

	// Quarter round 1
	x, y = bits.Mul64(s[0], s[1])

	s[0] = bits.RotateLeft64(s[0], 32)
	s[1] = bits.RotateLeft64(s[1], 32)
	s[2] ^= x
	s[3] ^= y

	// Quarter round 2
	x, y = bits.Mul64(s[1], s[2])

	s[0] ^= x
	s[2] = bits.RotateLeft64(s[2], 32)
	s[3] += y

	// Quarter round 3
	x, y = bits.Mul64(s[0], s[3])

	s[1] ^= x
	s[2] += y
	s[3] = bits.RotateLeft64(s[3], 32)

	// Quarter round 4
	x, y = bits.Mul64(s[2], s[3])

	s[0] += x
	s[1] += y

	// Post-round mixture
	mixture = s[0] + s[1] + s[2] + s[3]
	s[0], s[1], s[2], s[3] = mixture-s[3], mixture-s[2], mixture-s[1], mixture-s[0]
}

func (s *State128) MultPermuter() {
	var x, y uint32

	// Injecting "Twister! Sarmadi"
	s[0] ^= c0_hi
	s[1] ^= c0_lo
	s[2] ^= c1_hi
	s[3] ^= c1_lo

	// Pre-round mixture
	mixture := s[0] ^ s[1] ^ s[2] ^ s[3]
	s[0], s[1], s[2], s[3] = mixture^s[3], mixture^s[2], mixture^s[1], mixture^s[0]

	// Quarter round 1
	x, y = bits.Mul32(s[0], s[1])

	s[0] = bits.RotateLeft32(s[0], 16)
	s[1] = bits.RotateLeft32(s[1], 16)
	s[2] ^= x
	s[3] ^= y

	// Quarter round 2
	x, y = bits.Mul32(s[1], s[2])

	s[0] ^= x
	s[2] = bits.RotateLeft32(s[2], 16)
	s[3] += y

	// Quarter round 3
	x, y = bits.Mul32(s[0], s[3])

	s[1] ^= x
	s[2] += y
	s[3] = bits.RotateLeft32(s[3], 16)

	// Quarter round 4
	x, y = bits.Mul32(s[2], s[3])

	s[0] += x
	s[1] += y

	// Post-round mixture
	mixture = s[0] + s[1] + s[2] + s[3]
	s[0], s[1], s[2], s[3] = mixture-s[3], mixture-s[2], mixture-s[1], mixture-s[0]
}