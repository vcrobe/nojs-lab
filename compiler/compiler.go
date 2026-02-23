package compiler

import (
	"fmt"
	"path/filepath"
)

// Compile is the main entry point for the nojs AOT compiler.
// It discovers all *.gt.html component templates under srcDir, inspects
// their corresponding Go structs, and writes a *.generated.go file next
// to each template.
func Compile(srcDir string, devMode bool) error {
	opts := compileOptions{DevMode: devMode}

	// Convert srcDir to absolute path for consistent path handling
	absSrcDir, err := filepath.Abs(srcDir)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for srcDir: %w", err)
	}

	// Step 1: Discover component templates and inspect their Go structs for props.
	components, err := discoverAndInspectComponents(absSrcDir)
	if err != nil {
		return fmt.Errorf("failed to discover or inspect components: %w", err)
	}
	fmt.Printf("Discovered and inspected %d component templates.\n", len(components))

	componentMap := make(map[string]componentInfo)
	for _, comp := range components {
		componentMap[comp.LowercaseName] = comp
	}

	// Step 2: Loop through each discovered component and compile its template.
	for _, comp := range components {
		if err := compileComponentTemplate(comp, componentMap, absSrcDir, opts); err != nil {
			return fmt.Errorf("failed to compile template for %s: %w", comp.PascalName, err)
		}
	}
	return nil
}
