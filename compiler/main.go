package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	// Define command-line flags for the input and output file paths.
	// This makes our compiler flexible and easy to use from a script.
	inPath := flag.String("in", "", "The path to the input HTML file.")
	outPath := flag.String("out", "", "The path for the output Go file.")
	flag.Parse()

	// We need both flags to be present to work.
	if *inPath == "" || *outPath == "" {
		log.Fatal("Error: Both -in (input) and -out (output) flags are required.")
	}

	// Call the core Compile function from our other file.
	// This keeps the main function clean and focused on CLI logic.
	err := compile(*inPath, *outPath)
	if err != nil {
		log.Fatalf("Compilation failed: %v", err)
	}

	// Success! Let the user know.
	fmt.Printf("🎉 Successfully compiled %s to %s\n", *inPath, *outPath)
}
