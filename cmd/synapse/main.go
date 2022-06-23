package main

import (
	"log"

	"github.com/LambdaTest/test-at-scale/pkg/global"
)

// Main function just executes root command `ts`
// this project structure is inspired from `cobra` package
func main() {
	log.Printf("Starting synapse %s", global.SynapseBinaryVersion)
	if err := RootCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
