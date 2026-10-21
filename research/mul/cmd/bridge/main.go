package main

import (
	"encoding/json"
	"os"

	"github.com/MHSarmadi/Twister/research/mul"
)

// Request defines the JSON input structure.
type Request struct {
	Rounds int      `json:"rounds"`
	State  []uint64 `json:"state"` // Expects 4 words
}

// Response defines the JSON output structure.
type Response struct {
	State []uint64 `json:"state"`
}

func main() {
	var req Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		panic(err)
	}

	s := mul.State256{req.State[0], req.State[1], req.State[2], req.State[3]}
	for i := 0; i < req.Rounds; i++ {
		s.MultPermuter() // Assuming you've exported the method
	}

	// Return the final state
	if err := json.NewEncoder(os.Stdout).Encode(Response{
		State: []uint64{s[0], s[1], s[2], s[3]},
	}); err != nil {
		panic(err)
	}
}