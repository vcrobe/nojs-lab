package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/vcrobe/nojs/compiler"
)

func main() {
	// --- CLI Flags ---
	// The '-in' flag specifies the source directory to scan for components.
	inDir := flag.String("in", ".", "The source directory to scan for *.gt.html files.")
	// The '-dev' flag enables development mode (warnings, verbose errors, panic on lifecycle failures).
	devMode := flag.Bool("dev", false, "Enable development mode (warnings, verbose errors, panic on lifecycle failures)")
	flag.Parse()

	// The CLI's job is now to pass the directory to the core compiler logic.
	fmt.Printf("Starting compilation...\nSource directory: %s\n", *inDir)
	if *devMode {
		fmt.Printf("Development mode: ENABLED\n")
	}
	err := compiler.Compile(*inDir, *devMode)
	if err != nil {
		log.Fatalf("Compilation failed: %v", err)
	}

	// Success! Let the user know.
	fmt.Printf("🎉 Compilation completed successfully!\n")
}
