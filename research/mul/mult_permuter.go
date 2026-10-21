package mul

import "math/bits"

type State256 [4]uint64

const c0, c1 uint64 = 0x5477697374657221, 0x205361726d616469

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
