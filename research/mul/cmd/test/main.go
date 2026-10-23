package main

import (
	"fmt"

	"github.com/MHSarmadi/Twister"
	"github.com/MHSarmadi/Twister/research/mul"
)

func main() {
	println()
	println("====== State 128 ======")

	q1 := mul.State128{}
	fmt.Println(q1)

	q1.MultPermuter()
	fmt.Printf("%8x\n", q1)

	q1.MultPermuter()
	fmt.Printf("%8x\n", q1)

	println()
	println("====== State 256 ======")

	q2 := mul.State256{}
	fmt.Println(q2)

	q2.MultPermuter()
	fmt.Printf("%16x\n", q2)

	q2.MultPermuter()
	fmt.Printf("%16x\n", q2)

	println()
	println("====== State 512 ======")

	s1 := Twister.State512{}
	fmt.Println(s1)

	s1.Permute()
	fmt.Printf("%8x\n", s1)

	s1.Permute()
	fmt.Printf("%8x\n", s1)

	println()
	println("====== State 1024 ======")

	s2 := Twister.State1024{}
	fmt.Println(s2)

	s2.Permute()
	fmt.Printf("%16x\n", s2)

	s2.Permute()
	fmt.Printf("%16x\n", s2)
}
