package main

import (
	"log"
)

// Main function just executes root command `ts`
// this project structure is inspired from `cobra` package
func main() {
	if err := RootCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
