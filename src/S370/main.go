package main

import (
	"fmt"

	CPU "github.com/rcornwell/S370/internal/cpu"
)

func main() {
	fmt.Println("Hello S370!")
	CPU.InitializeCPU()
}
