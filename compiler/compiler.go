package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/vcrobe/nojs/events"
	"golang.org/x/net/html"
	"golang.org/x/tools/go/packages"
)

// componentSchema holds the type information for a component's props.
type componentSchema struct {
	Props   map[string]propertyDescriptor // Map of Prop name to its Go type (e.g., "Title": "string")
	State   map[string]propertyDescriptor // Map of State name to its Go type (internal component state)
	Methods map[string]methodDescriptor   // Map of method names to their signatures
	Slot    *propertyDescriptor           // Optional: single content slot field ([]*vdom.VNode)
}

type propertyDescriptor struct {
	Name          string
	LowercaseName string
	GoType        string
}

// methodDescriptor holds the signature information for a component method.
type methodDescriptor struct {
	Name    string            // Method name (e.g., "HandleClick")
	Params  []paramDescriptor // Parameter list
	Returns []string          // Return type names (currently unused, reserved for future)
}

// paramDescriptor describes a single parameter in a method signature.
type paramDescriptor struct {
	Name string // Parameter name (e.g., "e")
	Type string // Fully-qualified type (e.g., "events.ChangeEventArgs", "string")
}

// componentInfo holds all discovered information about a component.
type componentInfo struct {
	Path          string
	PascalName    string
	LowercaseName string
	PackageName   string
	Schema        componentSchema
}

// compileOptions holds compiler-wide options passed from CLI flags.
type compileOptions struct {
	DevMode bool // Enable development mode (warnings, verbose errors, panic on lifecycle failures)
}

// loopContext holds information about variables available in a loop scope.
type loopContext struct {
	IndexVar string // e.g., "i" or "_"
	ValueVar string // e.g., "user"
}

// textNodePosition tracks the location of an unwrapped text node in slot content.
type textNodePosition struct {
	lineNum     int
	colNum      int
	textContent string
}

// Regex to find data binding expressions like {FieldName} or {user.Name}
var dataBindingRegex = regexp.MustCompile(`\{([a-zA-Z0-9_.]+)\}`)

// Regex to find ternary expressions like { condition ? 'value1' : 'value2' }
var ternaryExprRegex = regexp.MustCompile(`\{\s*(!?)([a-zA-Z0-9_]+)\s*\?\s*'([^']*)'\s*:\s*'([^']*)'\s*\}`)

// Regex to find boolean shorthand like {condition} or {!condition}
var booleanShorthandRegex = regexp.MustCompile(`^\{\s*(!?)([a-zA-Z0-9_]+)\s*\}$`)

// Standard HTML boolean attributes
var standardBooleanAttrs = map[string]bool{
	"disabled":       true,
	"checked":        true,
	"readonly":       true,
	"required":       true,
	"autofocus":      true,
	"autoplay":       true,
	"controls":       true,
	"loop":           true,
	"muted":          true,
	"selected":       true,
	"hidden":         true,
	"multiple":       true,
	"novalidate":     true,
	"open":           true,
	"reversed":       true,
	"scoped":         true,
	"seamless":       true,
	"sortable":       true,
	"truespeed":      true,
	"default":        true,
	"ismap":          true,
	"formnovalidate": true,
}

// problematicHTMLTags lists HTML tags that conflict with component names.
// The Go html parser treats these case-insensitively and applies HTML5 semantics
// (e.g., <link> becomes self-closing and moves to <head>).
// Component names matching these will cause parsing issues.
var problematicHTMLTags = map[string]bool{
	// Void/self-closing elements (no children allowed)
	"area":   true,
	"base":   true,
	"br":     true,
	"col":    true,
	"embed":  true,
	"hr":     true,
	"img":    true,
	"input":  true,
	"link":   true, // Common conflict: Link component
	"meta":   true,
	"param":  true,
	"source": true,
	"track":  true,
	"wbr":    true,

	// Elements with special parsing rules
	"script":   true,
	"style":    true,
	"title":    true,
	"textarea": true,
	"select":   true,
	"option":   true,
	"optgroup": true,
	"template": true,
	"iframe":   true,
	"object":   true,
	"canvas":   true,
	"audio":    true,
	"video":    true,
	"form":     true, // Common conflict: Form component
	"button":   true, // Common conflict: Button component
	"label":    true,
	"fieldset": true,
	"legend":   true,
	"table":    true,
	"thead":    true,
	"tbody":    true,
	"tfoot":    true,
	"tr":       true,
	"td":       true,
	"th":       true,
	"caption":  true,
	"colgroup": true,

	// Commonly used semantic elements that could conflict
	"main":    true,
	"nav":     true,
	"header":  true,
	"footer":  true,
	"section": true,
	"article": true,
	"aside":   true,
	"details": true,
	"summary": true,
	"dialog":  true,
	"menu":    true,

	// Other potentially problematic tags
	"html": true,
	"head": true,
	"body": true,
	"div":  true,
	"span": true,
	"a":    true,
	"p":    true,
	"h1":   true,
	"h2":   true,
	"h3":   true,
	"h4":   true,
	"h5":   true,
	"h6":   true,
	"ul":   true,
	"ol":   true,
	"li":   true,
	"dl":   true,
	"dt":   true,
	"dd":   true,
}

// preprocessFor preprocesses template source to extract for-loop blocks and replace them with placeholder nodes.
// It validates that every {@for} has a matching {@endfor} and that trackBy is specified.
// Syntax: {@for index, value := range SliceName trackBy uniqueKeyExpression}{@endfor}
// The index can be _ to ignore it: {@for _, value := range SliceName trackBy uniqueKeyExpression}
func preprocessFor(src string, templatePath string) (string, error) {
	// Regex to match ONLY: {@for i, user := range Users trackBy user.ID} or {@for _, user := range Users trackBy user.ID}
	reFor := regexp.MustCompile(`\{\@for\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*,\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*:=\s*range\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+trackBy\s+([a-zA-Z0-9_.]+)\}`)

	// Regex to detect INVALID syntax: {@for user := range Users trackBy user.ID} (missing index/underscore)
	reForInvalid := regexp.MustCompile(`\{\@for\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*:=\s*range\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+trackBy\s+([a-zA-Z0-9_.]+)\}`)

	reEndFor := regexp.MustCompile(`\{\@endfor\}`)

	// Check for invalid syntax (missing index/underscore)
	if invalidMatches := reForInvalid.FindAllString(src, -1); len(invalidMatches) > 0 {
		lines := strings.Split(src, "\n")
		var invalidLines []int

		for i, line := range lines {
			if reForInvalid.MatchString(line) && !reFor.MatchString(line) {
				invalidLines = append(invalidLines, i+1)
			}
		}

		if len(invalidLines) > 0 {
			return "", fmt.Errorf("template syntax error in %s: Invalid {@for} syntax at line(s): %v\n"+
				"  The {@for} directive requires both index and value variables.\n"+
				"  Correct syntax: {@for index, value := range Slice trackBy value.Field}\n"+
				"  To ignore the index, use underscore: {@for _, value := range Slice trackBy value.Field}\n"+
				"  Example: {@for _, user := range Users trackBy user.ID}",
				templatePath, invalidLines)
		}
	}

	// Count directives to validate structure
	forCount := len(reFor.FindAllString(src, -1))
	endForCount := len(reEndFor.FindAllString(src, -1))

	if forCount != endForCount {
		// Find the line numbers to help the developer
		lines := strings.Split(src, "\n")
		var forLines []int
		var endForLines []int

		for i, line := range lines {
			if reFor.MatchString(line) {
				forLines = append(forLines, i+1)
			}
			if reEndFor.MatchString(line) {
				endForLines = append(endForLines, i+1)
			}
		}

		if forCount > endForCount {
			return "", fmt.Errorf("template validation error in %s: found %d {@for} directive(s) but only %d {@endfor} directive(s).\n"+
				"  {@for} found at line(s): %v\n"+
				"  {@endfor} found at line(s): %v\n"+
				"  Missing %d {@endfor} directive(s).",
				templatePath, forCount, endForCount, forLines, endForLines, forCount-endForCount)
		} else {
			return "", fmt.Errorf("template validation error in %s: found %d {@endfor} directive(s) but only %d {@for} directive(s).\n"+
				"  {@for} found at line(s): %v\n"+
				"  {@endfor} found at line(s): %v\n"+
				"  Extra %d {@endfor} directive(s) without matching {@for}.",
				templatePath, endForCount, forCount, forLines, endForLines, endForCount-forCount)
		}
	}

	// Transform {@for i, user := range Users trackBy user.ID} to placeholder elements
	src = reFor.ReplaceAllStringFunc(src, func(m string) string {
		matches := reFor.FindStringSubmatch(m)
		indexVar := matches[1]
		valueVar := matches[2]
		rangeExpr := matches[3]
		trackByExpr := matches[4]
		return fmt.Sprintf(`<go-for data-index="%s" data-value="%s" data-range="%s" data-trackby="%s">`,
			indexVar, valueVar, rangeExpr, trackByExpr)
	})

	src = reEndFor.ReplaceAllString(src, "</go-for>")
	return src, nil
}

// preprocessConditionals preprocesses template source to extract conditional blocks and replace them with placeholder nodes.
// It validates that every {@if} has a matching {@endif}.
func preprocessConditionals(src string, templatePath string) (string, error) {
	reIf := regexp.MustCompile(`\{\@if ([^}]+)\}`)
	reElseIf := regexp.MustCompile(`\{\@else if ([^}]+)\}`)
	reElse := regexp.MustCompile(`\{\@else\}`)
	reEndIf := regexp.MustCompile(`\{\@endif\}`)

	// Count directives to validate structure
	ifCount := len(reIf.FindAllString(src, -1))
	endifCount := len(reEndIf.FindAllString(src, -1))

	if ifCount != endifCount {
		// Find the line numbers to help the developer
		lines := strings.Split(src, "\n")
		var ifLines []int
		var endifLines []int

		for i, line := range lines {
			if reIf.MatchString(line) {
				ifLines = append(ifLines, i+1)
			}
			if reEndIf.MatchString(line) {
				endifLines = append(endifLines, i+1)
			}
		}

		if ifCount > endifCount {
			return "", fmt.Errorf("template validation error in %s: found %d {@if} directive(s) but only %d {@endif} directive(s).\n"+
				"  {@if} found at line(s): %v\n"+
				"  {@endif} found at line(s): %v\n"+
				"  Missing %d {@endif} directive(s).",
				templatePath, ifCount, endifCount, ifLines, endifLines, ifCount-endifCount)
		} else {
			return "", fmt.Errorf("template validation error in %s: found %d {@endif} directive(s) but only %d {@if} directive(s).\n"+
				"  {@if} found at line(s): %v\n"+
				"  {@endif} found at line(s): %v\n"+
				"  Extra %d {@endif} directive(s) without matching {@if}.",
				templatePath, endifCount, ifCount, ifLines, endifLines, endifCount-ifCount)
		}
	}

	src = reIf.ReplaceAllStringFunc(src, func(m string) string {
		cond := reIf.FindStringSubmatch(m)[1]
		return fmt.Sprintf("<go-conditional><go-if data-cond=\"%s\">", cond)
	})
	src = reElseIf.ReplaceAllStringFunc(src, func(m string) string {
		cond := reElseIf.FindStringSubmatch(m)[1]
		return fmt.Sprintf("</go-if><go-elseif data-cond=\"%s\">", cond)
	})
	// Handle {@else} - it closes the previous branch and opens go-else
	src = reElse.ReplaceAllString(src, func() string {
		// Check if the previous element is go-if or go-elseif
		// We need to close whichever was opened
		return "</go-if></go-elseif><go-else>"
	}())
	// {@endif} closes the last opened branch and the wrapper
	src = reEndIf.ReplaceAllString(src, "</go-if></go-elseif></go-else></go-conditional>")
	return src, nil
}

// estimateLineNumber tries to find the approximate line number where text appears in HTML source.
// For better accuracy with expressions containing special characters, it searches for distinctive parts.
func estimateLineNumber(htmlSource, text string) int {
	lines := strings.Split(htmlSource, "\n")

	// First try: exact match
	for i, line := range lines {
		if strings.Contains(line, text) {
			return i + 1
		}
	}

	// Second try: if text contains ternary pattern, search for the ternary part
	if strings.Contains(text, "?") && strings.Contains(text, ":") {
		// Extract a distinctive part (the condition before '?')
		if idx := strings.Index(text, "?"); idx > 0 {
			// Get text around the '?' for better matching
			start := strings.LastIndex(text[:idx], "{")
			if start >= 0 {
				searchText := text[start : idx+1] // Include { up to ?
				for i, line := range lines {
					if strings.Contains(line, searchText) {
						return i + 1
					}
				}
			}
		}
	}

	// Fallback: search for any significant substring
	trimmed := strings.TrimSpace(text)
	if len(trimmed) > 10 {
		searchText := trimmed[:10]
		for i, line := range lines {
			if strings.Contains(line, searchText) {
				return i + 1
			}
		}
	}

	return 1 // Default to line 1 if not found
}

// validateComponentName checks if a component name conflicts with HTML tags.
// The Go html.Parse treats tags case-insensitively and applies HTML5 semantics,
// which can cause components to be misparsed (e.g., <Link> becomes self-closing <link>).
func validateComponentName(componentName, templatePath string) error {
	lowerName := strings.ToLower(componentName)
	if problematicHTMLTags[lowerName] {
		return fmt.Errorf(
			"Compilation Error: Component name '%s' in %s conflicts with HTML tag '<%s>'.\n"+
				"\n"+
				"The Go html.Parse library treats component names case-insensitively and applies HTML5 parsing rules.\n"+
				"This causes issues like:\n"+
				"  - <Link> is parsed as <link> (self-closing, no children allowed)\n"+
				"  - <Form> is parsed as <form> (special form parsing rules)\n"+
				"  - <Button> is parsed as <button> (special nesting restrictions)\n"+
				"\n"+
				"Suggested alternatives:\n"+
				"  - Link → RouterLink, NavLink, or AppLink\n"+
				"  - Form → DataForm or AppForm\n"+
				"  - Button → ActionButton or CustomButton\n"+
				"\n"+
				"Use PascalCase names that don't match HTML tags (case-insensitive).",
			componentName, templatePath, lowerName)
	}
	return nil
}

// isBooleanAttribute checks if an attribute name is a standard HTML boolean attribute.
func isBooleanAttribute(attrName string) bool {
	return standardBooleanAttrs[attrName]
}

// validateBooleanCondition validates that a condition references a boolean field on the component.
// Returns the propertyDescriptor if valid, or exits with a compile error.
func validateBooleanCondition(condition string, comp componentInfo, templatePath string, lineNumber int, htmlSource string) propertyDescriptor {
	propDesc, exists := comp.Schema.Props[strings.ToLower(condition)]
	if !exists {
		// Also check state fields
		propDesc, exists = comp.Schema.State[strings.ToLower(condition)]
	}
	if !exists {
		allFields := append(getAvailableFieldNames(comp.Schema.Props), getAvailableFieldNames(comp.Schema.State)...)
		availableFields := strings.Join(allFields, ", ")
		contextLines := getContextLines(htmlSource, lineNumber, 2)
		fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Condition '%s' not found on component '%s'. Available fields: [%s]\n%s",
			templatePath, lineNumber, condition, comp.PascalName, availableFields, contextLines)
		os.Exit(1)
	}
	if propDesc.GoType != "bool" {
		contextLines := getContextLines(htmlSource, lineNumber, 2)
		fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Condition '%s' must be a bool field, found type '%s'.\n%s",
			templatePath, lineNumber, condition, propDesc.GoType, contextLines)
		os.Exit(1)
	}
	return propDesc
}

// validateEventHandler validates that an event handler exists and has the correct signature.
// Returns the methodDescriptor if valid, or exits with a compile error and helpful suggestions.
func validateEventHandler(eventName, handlerName, tagName string, comp componentInfo, templatePath string, lineNumber int, htmlSource string) methodDescriptor {
	// Get the event signature from the registry
	eventSig := events.GetEventSignature(eventName)
	if eventSig == nil {
		contextLines := getContextLines(htmlSource, lineNumber, 2)
		fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Unknown event '@%s'.\n%s\nSupported events: @onclick, @oninput, @onchange, @onkeydown, @onkeyup, @onkeypress, @onfocus, @onblur, @onsubmit, @onmousedown, @onmouseup, @onmousemove\n",
			templatePath, lineNumber, eventName, contextLines)
		os.Exit(1)
	}

	// Check if the event is supported on this HTML tag
	if !events.IsEventSupported(eventName, tagName) {
		contextLines := getContextLines(htmlSource, lineNumber, 2)
		fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Event '@%s' is not supported on <%s>.\n%s\nSupported elements for @%s: %v\n",
			templatePath, lineNumber, eventName, tagName, contextLines, eventName, eventSig.SupportedTags)
		os.Exit(1)
	}

	// Check if the handler method exists
	method, exists := comp.Schema.Methods[handlerName]
	if !exists {
		contextLines := getContextLines(htmlSource, lineNumber, 2)
		availableMethods := getAvailableMethodNames(comp.Schema.Methods)
		fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Handler method '%s' not found on component '%s'.\n%s\nAvailable methods: %s\n",
			templatePath, lineNumber, handlerName, comp.PascalName, contextLines, availableMethods)
		os.Exit(1)
	}

	// Validate the method signature
	// Special case: onclick can accept either func() or func(ClickEventArgs)
	if eventName == "onclick" {
		if len(method.Params) == 0 {
			// func() - valid, will use AdaptNoArgEvent
			return method
		} else if len(method.Params) == 1 && method.Params[0].Type == "events.ClickEventArgs" {
			// func(ClickEventArgs) - valid, will use AdaptClickEvent
			return method
		} else {
			// Invalid signature
			contextLines := getContextLines(htmlSource, lineNumber, 2)
			fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Handler '%s' for '@onclick' has incorrect signature.\n%s\nExpected: func(c *%s) %s() OR func(c *%s) %s(e events.ClickEventArgs)\nFound:    func(c *%s) %s(",
				templatePath, lineNumber, handlerName, contextLines,
				comp.PascalName, handlerName,
				comp.PascalName, handlerName,
				comp.PascalName, handlerName)
			for i, p := range method.Params {
				if i > 0 {
					fmt.Fprintf(os.Stderr, ", ")
				}
				fmt.Fprintf(os.Stderr, "%s %s", p.Name, p.Type)
			}
			fmt.Fprintf(os.Stderr, ")\n")
			os.Exit(1)
		}
	}

	// Standard validation for other events
	if eventSig.RequiresArgs {
		// Event requires arguments - handler must have exactly one parameter of the correct type
		if len(method.Params) != 1 {
			contextLines := getContextLines(htmlSource, lineNumber, 2)
			fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Handler '%s' for '@%s' has incorrect signature.\n%s\nExpected: func(c *%s) %s(e %s)\nFound:    func(c *%s) %s(",
				templatePath, lineNumber, handlerName, eventName, contextLines,
				comp.PascalName, handlerName, eventSig.ArgsType,
				comp.PascalName, handlerName)
			for i, p := range method.Params {
				if i > 0 {
					fmt.Fprintf(os.Stderr, ", ")
				}
				fmt.Fprintf(os.Stderr, "%s %s", p.Name, p.Type)
			}
			fmt.Fprintf(os.Stderr, ")\n")
			os.Exit(1)
		}

		// Check if the parameter type matches
		if method.Params[0].Type != eventSig.ArgsType {
			contextLines := getContextLines(htmlSource, lineNumber, 2)
			fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Handler '%s' for '@%s' has wrong parameter type.\n%s\nExpected: func(c *%s) %s(e %s)\nFound:    func(c *%s) %s(e %s)\n",
				templatePath, lineNumber, handlerName, eventName, contextLines,
				comp.PascalName, handlerName, eventSig.ArgsType,
				comp.PascalName, handlerName, method.Params[0].Type)
			os.Exit(1)
		}
	} else {
		// Event requires no arguments - handler must have zero parameters
		if len(method.Params) != 0 {
			contextLines := getContextLines(htmlSource, lineNumber, 2)
			fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Handler '%s' for '@%s' has incorrect signature.\n%s\nExpected: func(c *%s) %s()\nFound:    func(c *%s) %s(",
				templatePath, lineNumber, handlerName, eventName, contextLines,
				comp.PascalName, handlerName,
				comp.PascalName, handlerName)
			for i, p := range method.Params {
				if i > 0 {
					fmt.Fprintf(os.Stderr, ", ")
				}
				fmt.Fprintf(os.Stderr, "%s %s", p.Name, p.Type)
			}
			fmt.Fprintf(os.Stderr, ")\n\nSuggestion: For '@%s' events on <%s>, the handler should not take any parameters.\n", eventName, tagName)
			os.Exit(1)
		}
	}

	return method
}

// generateTernaryExpression generates Go code for a ternary conditional expression.
// Supports negation operator: if negated is true, inverts the condition.
func generateTernaryExpression(negated bool, condition, trueVal, falseVal, receiver string, propDesc propertyDescriptor) string {
	if negated {
		// Swap true and false values for negation
		trueVal, falseVal = falseVal, trueVal
	}
	return fmt.Sprintf(`func() string {
		if %s.%s {
			return %s
		}
		return %s
	}()`, receiver, propDesc.Name, strconv.Quote(trueVal), strconv.Quote(falseVal))
}

// Compile is the main entry point for the AOT compiler.
func compile(srcDir, outDir string, devMode bool) error {
	opts := compileOptions{DevMode: devMode}

	// Step 1: Discover component templates and inspect their Go structs for props.
	components, err := discoverAndInspectComponents(srcDir)
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
		if err := compileComponentTemplate(comp, componentMap, outDir, opts); err != nil {
			return fmt.Errorf("failed to compile template for %s: %w", comp.PascalName, err)
		}
	}
	return nil
}

// discoverAndInspectComponents finds all *.gt.html files and inspects their corresponding .go files.
func discoverAndInspectComponents(rootDir string) ([]componentInfo, error) {
	var components []componentInfo

	// Step 1: Load all packages in the module, configured for WASM.
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles, // Request file info
		Dir:  rootDir,
		Env:  append(os.Environ(), "GOOS=js", "GOARCH=wasm"),
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	// Step 2: Iterate through the loaded packages.
	for _, pkg := range pkgs {
		if len(pkg.GoFiles) == 0 {
			continue // Skip packages that are empty for the js/wasm target.
		}

		// All files in a package share the same directory.
		packageDir := filepath.Dir(pkg.GoFiles[0])

		// Step 3: Scan the package's directory for component templates (*.gt.html).
		files, err := os.ReadDir(packageDir)
		if err != nil {
			fmt.Printf("Warning: could not read directory %s: %v\n", packageDir, err)
			continue
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".gt.html") {
				continue
			}

			// We found a component template.
			templatePath := filepath.Join(packageDir, file.Name())
			pascalName := strings.TrimSuffix(file.Name(), ".gt.html")
			goFilePath := filepath.Join(packageDir, strings.ToLower(pascalName)+".go")

			schema, err := inspectGoFile(goFilePath, pascalName)
			if err != nil {
				fmt.Printf("Warning: could not inspect Go file %s: %v\n", goFilePath, err)
				schema = componentSchema{
					Props:   make(map[string]propertyDescriptor),
					Methods: make(map[string]methodDescriptor),
				}
			}

			components = append(components, componentInfo{
				Path:          templatePath,
				PascalName:    pascalName,
				LowercaseName: strings.ToLower(pascalName),
				PackageName:   pkg.Name, // Use the package name from the loader.
				Schema:        schema,
			})

			// Validate that component name doesn't conflict with HTML tags
			if err := validateComponentName(pascalName, templatePath); err != nil {
				return nil, err
			}
		}
	}

	if len(components) == 0 {
		fmt.Println("Warning: No component templates (*.gt.html) were found in any Go packages.")
	}

	return components, nil
}

// extractTypeName extracts the type name from an AST expression.
// Handles simple types (int, string, bool), slice types ([]User), and pointer types (*User).
func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		// Simple type like "string", "int", "bool"
		return t.Name
	case *ast.ArrayType:
		// Slice or array type like "[]User"
		elemType := extractTypeName(t.Elt)
		return "[]" + elemType
	case *ast.StarExpr:
		// Pointer type like "*User"
		elemType := extractTypeName(t.X)
		return "*" + elemType
	case *ast.SelectorExpr:
		// Qualified type like "time.Time"
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
	}
	return "unknown"
}

// extractParams extracts parameter descriptors from a function's parameter list.
func extractParams(fields *ast.FieldList) []paramDescriptor {
	if fields == nil {
		return nil
	}
	var params []paramDescriptor
	for _, field := range fields.List {
		typeName := extractTypeName(field.Type)
		// Handle cases where multiple params share the same type: func(a, b string)
		if len(field.Names) > 0 {
			for _, name := range field.Names {
				params = append(params, paramDescriptor{
					Name: name.Name,
					Type: typeName,
				})
			}
		} else {
			// Unnamed parameter (rare but valid Go)
			params = append(params, paramDescriptor{
				Name: "",
				Type: typeName,
			})
		}
	}
	return params
}

// extractReturns extracts return type names from a function's return list.
func extractReturns(fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}
	var returns []string
	for _, field := range fields.List {
		typeName := extractTypeName(field.Type)
		// Multiple return values of the same type: func() (int, int)
		if len(field.Names) > 0 {
			for range field.Names {
				returns = append(returns, typeName)
			}
		} else {
			returns = append(returns, typeName)
		}
	}
	return returns
}

// inspectStructInFile is a helper that inspects a specific struct type in a Go file.
// It returns a schema with the struct's exported fields.
func inspectStructInFile(path, structName string) (componentSchema, error) {
	schema := componentSchema{
		Props:   make(map[string]propertyDescriptor),
		Methods: make(map[string]methodDescriptor),
	}
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return schema, err
	}

	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok && typeSpec.Name.Name == structName {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				found = true
				for _, field := range structType.Fields.List {
					if len(field.Names) > 0 && field.Names[0].IsExported() {
						fieldName := field.Names[0].Name
						goType := extractTypeName(field.Type)
						schema.Props[strings.ToLower(fieldName)] = propertyDescriptor{
							Name:          fieldName,
							LowercaseName: strings.ToLower(fieldName),
							GoType:        goType,
						}
					}
				}
			}
		}
		return true
	})

	if !found {
		return schema, fmt.Errorf("struct '%s' not found in file", structName)
	}

	return schema, nil
}

// inspectGoFile parses a Go file and extracts the prop schema for a given struct.
func inspectGoFile(path, structName string) (componentSchema, error) {
	schema := componentSchema{
		Props:   make(map[string]propertyDescriptor),
		State:   make(map[string]propertyDescriptor),
		Methods: make(map[string]methodDescriptor),
		Slot:    nil,
	}
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return schema, err
	}

	var slotFields []propertyDescriptor // Track all slot fields for validation

	ast.Inspect(node, func(n ast.Node) bool {
		// Inspect for struct fields (Props)
		if typeSpec, ok := n.(*ast.TypeSpec); ok && typeSpec.Name.Name == structName {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				for _, field := range structType.Fields.List {
					if len(field.Names) > 0 && field.Names[0].IsExported() {
						fieldName := field.Names[0].Name
						goType := extractTypeName(field.Type)

						// Check if field is marked as state via struct tag
						isState := false
						if field.Tag != nil {
							tag := field.Tag.Value
							// Parse struct tag - remove surrounding backticks
							if len(tag) >= 2 {
								tag = tag[1 : len(tag)-1]
							}
							// Check for nojs:"state" tag
							if strings.Contains(tag, `nojs:"state"`) {
								isState = true
							}
						}

						propDesc := propertyDescriptor{
							Name:          fieldName,
							LowercaseName: strings.ToLower(fieldName),
							GoType:        goType,
						}

						// Check if this is a content slot field ([]*vdom.VNode)
						if goType == "[]*vdom.VNode" {
							slotFields = append(slotFields, propDesc)
						} else if !isState {
							// Regular prop field - only add if not marked as state
							schema.Props[strings.ToLower(fieldName)] = propDesc
						} else {
							// State field - add to State map for template access
							schema.State[strings.ToLower(fieldName)] = propDesc
						}
					}
				}
			}
		}

		// Inspect for methods (Event Handlers)
		if funcDecl, ok := n.(*ast.FuncDecl); ok && funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
			recv := funcDecl.Recv.List[0].Type
			if starExpr, ok := recv.(*ast.StarExpr); ok {
				recv = starExpr.X
			}
			if typeIdent, ok := recv.(*ast.Ident); ok && typeIdent.Name == structName {
				if funcDecl.Name.IsExported() {
					methodDesc := methodDescriptor{
						Name:    funcDecl.Name.Name,
						Params:  extractParams(funcDecl.Type.Params),
						Returns: extractReturns(funcDecl.Type.Results),
					}
					schema.Methods[funcDecl.Name.Name] = methodDesc
				}
			}
		}

		return true
	})

	// Validate single slot constraint
	if len(slotFields) > 1 {
		var fieldNames []string
		for _, sf := range slotFields {
			fieldNames = append(fieldNames, sf.Name)
		}
		fmt.Fprintf(os.Stderr, "Compilation Error: could not inspect Go file %s: component '%s' has multiple content slot fields: [%s]. Only one []*vdom.VNode field is allowed per component\n",
			path, structName, strings.Join(fieldNames, ", "))
		os.Exit(1)
	}

	// Set the single slot field if found
	if len(slotFields) == 1 {
		schema.Slot = &slotFields[0]
	}

	return schema, nil
}

// compileComponentTemplate handles the code generation for a single component.
func compileComponentTemplate(comp componentInfo, componentMap map[string]componentInfo, outDir string, opts compileOptions) error {
	htmlContent, err := os.ReadFile(comp.Path)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", comp.Path, err)
	}
	htmlString := string(htmlContent)

	// Preprocess conditional blocks with validation
	htmlString, err = preprocessConditionals(htmlString, comp.Path)
	if err != nil {
		return err // Error message already includes template path and details
	}

	// Preprocess for-loop blocks with validation
	htmlString, err = preprocessFor(htmlString, comp.Path)
	if err != nil {
		return err // Error message already includes template path and details
	}

	doc, err := html.Parse(strings.NewReader(htmlString))
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}
	bodyNode := findBody(doc)
	if bodyNode == nil {
		return fmt.Errorf("could not find <body> tag")
	}

	rootElement := findFirstElementChild(bodyNode)
	if rootElement == nil {
		return fmt.Errorf("no element found inside <body> tag to compile")
	}

	// Generate code for a single root node
	generatedCode := generateNodeCode(rootElement, "c", componentMap, comp, htmlString, opts, nil)

	// Generate the ApplyProps method body
	applyPropsBody := generateApplyPropsBody(comp)

	// NOTE: NO build tags! This file must be available to both WASM and test builds.
	// The core types (vdom.VNode, runtime.Renderer, runtime.Component) are now
	// available without build tags, allowing this generated code to work everywhere.
	template := `// Code generated by the nojs AOT compiler. DO NOT EDIT.
package %[2]s

import (
	"fmt"
	"strconv" // Added for type conversions

	"github.com/vcrobe/nojs/console"
	"github.com/vcrobe/nojs/events"
	"github.com/vcrobe/nojs/runtime"
	"github.com/vcrobe/nojs/vdom"
)

// ApplyProps copies props from source to the receiver, preserving internal state.
// This method is generated automatically by the compiler.
func (c *%[1]s) ApplyProps(source runtime.Component) {
	src, ok := source.(*%[1]s)
	if !ok {
		// Type mismatch - this should never happen in normal operation
		return
	}
	_ = src // Suppress unused variable warning if no props to copy
%[4]s
}

// Render generates the VNode tree for the %[1]s component.
func (c *%[1]s) Render(r runtime.Renderer) *vdom.VNode {
	_ = strconv.Itoa // Suppress unused import error if no props are converted
	_ = fmt.Sprintf  // Suppress unused import error if no bindings are used
	_ = console.Log  // Suppress unused import error if no loops use dev warnings
	_ = events.AdaptNoArgEvent // Suppress unused import error if no event handlers are used

	return %[3]s
}
`

	source := fmt.Sprintf(template, comp.PascalName, comp.PackageName, generatedCode, applyPropsBody)

	// Format the generated source code
	formattedSource, err := format.Source([]byte(source))
	if err != nil {
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	outFileName := fmt.Sprintf("%s.generated.go", comp.PascalName)
	outFilePath := filepath.Join(outDir, outFileName)
	return os.WriteFile(outFilePath, formattedSource, 0644)
}

// generateApplyPropsBody generates the body of the ApplyProps method.
// It creates assignment statements to copy all props from source to receiver.
func generateApplyPropsBody(comp componentInfo) string {
	if len(comp.Schema.Props) == 0 && comp.Schema.Slot == nil {
		return "\t// No props to copy"
	}

	var assignments []string

	// Copy regular props (sorted by name for consistent output)
	propNames := make([]string, 0, len(comp.Schema.Props))
	for propName := range comp.Schema.Props {
		propNames = append(propNames, propName)
	}
	// Sort for deterministic output
	for i := 0; i < len(propNames); i++ {
		for j := i + 1; j < len(propNames); j++ {
			if propNames[i] > propNames[j] {
				propNames[i], propNames[j] = propNames[j], propNames[i]
			}
		}
	}

	for _, propName := range propNames {
		prop := comp.Schema.Props[propName]
		assignments = append(assignments,
			fmt.Sprintf("\tc.%s = src.%s", prop.Name, prop.Name))
	}

	// Copy slot content if exists
	if comp.Schema.Slot != nil {
		assignments = append(assignments,
			fmt.Sprintf("\tc.%s = src.%s", comp.Schema.Slot.Name, comp.Schema.Slot.Name))
	}

	return strings.Join(assignments, "\n")
}

// generateAttributesMap is a helper to create the Go map literal for an element's attributes.
func generateAttributesMap(n *html.Node, receiver string, currentComp componentInfo, htmlSource string) string {
	var attrs, eventHandlers []string
	for _, a := range n.Attr {
		if after, ok := strings.CutPrefix(a.Key, "@"); ok {
			eventName := after
			handlerName := a.Val
			lineNumber := findEventLineNumber(n, eventName, htmlSource)

			// Validate event handler signature (compile-time type safety!)
			method := validateEventHandler(eventName, handlerName, n.Data, currentComp, currentComp.Path, lineNumber, htmlSource)

			// Get the event signature to determine if we need an adapter
			// Note: using full import path since 'events' is also a local variable name
			eventSig := events.GetEventSignature(eventName)

			// Generate the event handler code
			handlerRef := fmt.Sprintf(`%s.%s`, receiver, handlerName)

			// Convert @eventname to camelCase for JavaScript (e.g., "onclick" -> "onClick")
			jsEventName := "on" + strings.ToUpper(eventName[2:3]) + eventName[3:]

			// Determine which adapter to use based on event type and method signature
			if eventName == "onclick" {
				// onclick supports both func() and func(ClickEventArgs)
				if len(method.Params) == 0 {
					// func() - use no-arg adapter
					eventHandlers = append(eventHandlers, fmt.Sprintf(`"%s": events.AdaptNoArgEvent(%s)`, jsEventName, handlerRef))
				} else if len(method.Params) == 1 && method.Params[0].Type == "events.ClickEventArgs" {
					// func(ClickEventArgs) - use click adapter
					eventHandlers = append(eventHandlers, fmt.Sprintf(`"%s": events.AdaptClickEvent(%s)`, jsEventName, handlerRef))
				}
			} else if eventSig.RequiresArgs {
				// Event requires arguments - use the appropriate adapter
				var adapterFunc string
				switch eventSig.ArgsType {
				case "events.ChangeEventArgs":
					adapterFunc = "events.AdaptChangeEvent"
				case "events.KeyboardEventArgs":
					adapterFunc = "events.AdaptKeyboardEvent"
				case "events.MouseEventArgs":
					adapterFunc = "events.AdaptMouseEvent"
				case "events.FocusEventArgs":
					adapterFunc = "events.AdaptFocusEvent"
				case "events.FormEventArgs":
					adapterFunc = "events.AdaptFormEvent"
				default:
					fmt.Fprintf(os.Stderr, "Internal Error: Unknown event args type '%s'\n", eventSig.ArgsType)
					os.Exit(1)
				}
				eventHandlers = append(eventHandlers, fmt.Sprintf(`"%s": %s(%s)`, jsEventName, adapterFunc, handlerRef))
			} else {
				// Event requires no arguments - use the no-arg adapter
				eventHandlers = append(eventHandlers, fmt.Sprintf(`"%s": events.AdaptNoArgEvent(%s)`, jsEventName, handlerRef))
			}

			// Mark that method is used (prevents unused warnings)
			_ = method
		} else {
			// Check for inline conditional expressions in attribute values
			attrValue := a.Val
			lineNum := estimateLineNumber(htmlSource, fmt.Sprintf(`%s="%s"`, a.Key, attrValue))

			// Check for malformed ternary expressions (mismatched braces)
			openBraces := strings.Count(attrValue, "{")
			closeBraces := strings.Count(attrValue, "}")

			if openBraces > closeBraces {
				// Check if this looks like an attempted ternary expression
				if strings.Contains(attrValue, "?") && strings.Contains(attrValue, ":") && strings.Contains(attrValue, "'") {
					contextLines := getContextLines(htmlSource, lineNum, 2)
					fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Malformed expression in attribute '%s' - unclosed braces (found %d opening '{' but %d closing '}')\n%s\n"+
						"This appears to be an incomplete ternary expression.\n"+
						"Ternary expressions must be complete: {condition ? 'true' : 'false'}\n"+
						"Expected format: {FieldName ? 'value1' : 'value2'} or {!FieldName ? 'value1' : 'value2'}\n",
						currentComp.Path, lineNum, a.Key, openBraces, closeBraces, contextLines)
					os.Exit(1)
				}
			}

			// Pattern 1: Check for boolean shorthand syntax for boolean attributes
			// This must come BEFORE general data binding to handle boolean attributes correctly
			if match := booleanShorthandRegex.FindStringSubmatch(attrValue); match != nil && isBooleanAttribute(a.Key) {
				negated := match[1] == "!"
				condition := match[2]

				// Validate condition is a boolean field
				propDesc := validateBooleanCondition(condition, currentComp, currentComp.Path, lineNum, htmlSource)

				// Generate conditional code: if negated, invert the condition
				if negated {
					attrs = append(attrs, fmt.Sprintf(`"%s": !%s.%s`, a.Key, receiver, propDesc.Name))
				} else {
					attrs = append(attrs, fmt.Sprintf(`"%s": %s.%s`, a.Key, receiver, propDesc.Name))
				}
				continue
			}

			// Pattern 2: Ternary expressions in attribute values
			if ternaryExprRegex.MatchString(attrValue) {
				// Replace all ternary expressions in the value
				result := attrValue
				ternaryMatches := ternaryExprRegex.FindAllStringSubmatch(attrValue, -1)

				for _, match := range ternaryMatches {
					fullMatch := match[0]
					negated := match[1] == "!"
					condition := match[2]
					trueVal := match[3]
					falseVal := match[4]

					// Validate condition is a boolean field
					propDesc := validateBooleanCondition(condition, currentComp, currentComp.Path, lineNum, htmlSource)

					// Generate ternary expression
					ternaryCode := generateTernaryExpression(negated, condition, trueVal, falseVal, receiver, propDesc)

					// If the attribute value is only the ternary expression
					if result == fullMatch {
						attrs = append(attrs, fmt.Sprintf(`"%s": %s`, a.Key, ternaryCode))
						result = ""
						break
					}

					// Otherwise, replace in the string (for concatenation)
					result = strings.Replace(result, fullMatch, "%s", 1)
				}

				// If there were other parts, wrap in fmt.Sprintf
				if result != "" {
					var args []string
					for _, match := range ternaryMatches {
						negated := match[1] == "!"
						condition := match[2]
						trueVal := match[3]
						falseVal := match[4]
						propDesc := validateBooleanCondition(condition, currentComp, currentComp.Path, lineNum, htmlSource)
						args = append(args, generateTernaryExpression(negated, condition, trueVal, falseVal, receiver, propDesc))
					}
					attrs = append(attrs, fmt.Sprintf(`"%s": fmt.Sprintf(%s, %s)`, a.Key, strconv.Quote(result), strings.Join(args, ", ")))
				}
				continue
			}

			// Pattern 3: Regular data binding in attribute values (e.g., {FieldName})
			// This handles simple property interpolation for non-boolean attributes
			matches := dataBindingRegex.FindAllStringSubmatch(attrValue, -1)
			if len(matches) > 0 {
				// Check if the entire value is a single data binding (e.g., href='{Href}')
				if len(matches) == 1 && matches[0][0] == attrValue {
					fieldName := matches[0][1]

					// Validate that the field exists (check both Props and State)
					propDesc, exists := currentComp.Schema.Props[strings.ToLower(fieldName)]
					if !exists {
						propDesc, exists = currentComp.Schema.State[strings.ToLower(fieldName)]
					}
					if !exists {
						allFields := append(getAvailableFieldNames(currentComp.Schema.Props), getAvailableFieldNames(currentComp.Schema.State)...)
						availableFields := strings.Join(allFields, ", ")
						contextLines := getContextLines(htmlSource, lineNum, 2)
						fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Property '%s' not found in component struct. Available fields: [%s]\n%s",
							currentComp.Path, lineNum, fieldName, availableFields, contextLines)
						os.Exit(1)
					}

					// Generate direct field reference
					attrs = append(attrs, fmt.Sprintf(`"%s": %s.%s`, a.Key, receiver, propDesc.Name))
					continue
				}

				// Multiple bindings or mixed content (e.g., '{Base}/{Path}')
				formatString := dataBindingRegex.ReplaceAllString(attrValue, "%v")
				var args []string
				for _, match := range matches {
					fieldName := match[1]

					// Validate that the field exists (check both Props and State)
					propDesc, exists := currentComp.Schema.Props[strings.ToLower(fieldName)]
					if !exists {
						propDesc, exists = currentComp.Schema.State[strings.ToLower(fieldName)]
					}
					if !exists {
						allFields := append(getAvailableFieldNames(currentComp.Schema.Props), getAvailableFieldNames(currentComp.Schema.State)...)
						availableFields := strings.Join(allFields, ", ")
						contextLines := getContextLines(htmlSource, lineNum, 2)
						fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Property '%s' not found in component struct. Available fields: [%s]\n%s",
							currentComp.Path, lineNum, fieldName, availableFields, contextLines)
						os.Exit(1)
					}

					args = append(args, fmt.Sprintf("%s.%s", receiver, propDesc.Name))
				}
				attrs = append(attrs, fmt.Sprintf(`"%s": fmt.Sprintf(%s, %s)`, a.Key, strconv.Quote(formatString), strings.Join(args, ", ")))
				continue
			}

			// Pattern 4: Regular static attribute
			attrs = append(attrs, fmt.Sprintf(`"%s": "%s"`, a.Key, a.Val))
		}
	}

	if len(attrs) == 0 && len(eventHandlers) == 0 {
		return "nil"
	}
	allProps := append(attrs, eventHandlers...)
	return fmt.Sprintf("map[string]any{%s}", strings.Join(allProps, ", "))
}

// generateTextExpression handles data binding in text nodes.
// loopCtx can be nil if not inside a loop.
func generateTextExpression(text string, receiver string, currentComp componentInfo, htmlSource string, lineNumber int, loopCtx *loopContext) string {
	// Check for malformed ternary expressions (opening { with ternary pattern but no closing })
	// Count opening and closing braces to detect mismatches
	openBraces := strings.Count(text, "{")
	closeBraces := strings.Count(text, "}")

	// If we have mismatched braces and the text looks like it contains a ternary pattern
	if openBraces > closeBraces {
		// Check if this looks like an attempted ternary expression
		if strings.Contains(text, "?") && strings.Contains(text, ":") && strings.Contains(text, "'") {
			contextLines := getContextLines(htmlSource, lineNumber, 2)
			fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Malformed expression - unclosed braces (found %d opening '{' but %d closing '}')\n%s\n"+
				"This appears to be an incomplete ternary expression.\n"+
				"Ternary expressions must be complete: {condition ? 'true' : 'false'}\n"+
				"Expected format: {FieldName ? 'value1' : 'value2'} or {!FieldName ? 'value1' : 'value2'}\n",
				currentComp.Path, lineNumber, openBraces, closeBraces, contextLines)
			os.Exit(1)
		}
	}

	// Check for ternary expressions first
	ternaryMatches := ternaryExprRegex.FindAllStringSubmatch(text, -1)

	if len(ternaryMatches) > 0 {
		// Handle ternary expressions
		result := text

		for _, match := range ternaryMatches {
			fullMatch := match[0]
			negated := match[1] == "!"
			condition := match[2]
			trueVal := match[3]
			falseVal := match[4]

			// Validate condition is a boolean field
			propDesc := validateBooleanCondition(condition, currentComp, currentComp.Path, lineNumber, htmlSource)

			// Generate ternary expression
			ternaryCode := generateTernaryExpression(negated, condition, trueVal, falseVal, receiver, propDesc)

			// If the text contains only the ternary expression, return it directly
			if result == fullMatch {
				return ternaryCode
			}

			// Otherwise, replace the match with a placeholder for fmt.Sprintf
			result = strings.Replace(result, fullMatch, "%s", 1)
		}

		// If there are other parts of the text, wrap in fmt.Sprintf
		var args []string
		for _, match := range ternaryMatches {
			negated := match[1] == "!"
			condition := match[2]
			trueVal := match[3]
			falseVal := match[4]
			propDesc := validateBooleanCondition(condition, currentComp, currentComp.Path, lineNumber, htmlSource)
			args = append(args, generateTernaryExpression(negated, condition, trueVal, falseVal, receiver, propDesc))
		}

		return fmt.Sprintf(`fmt.Sprintf(%s, %s)`, strconv.Quote(result), strings.Join(args, ", "))
	}

	// Original data binding logic
	matches := dataBindingRegex.FindAllStringSubmatch(text, -1)

	if len(matches) == 0 {
		return strconv.Quote(text) // It's just a static string
	}

	formatString := dataBindingRegex.ReplaceAllString(text, "%v")
	var args []string

	for _, match := range matches {
		fieldName := match[1]

		// Check if this is a loop variable first
		if loopCtx != nil {
			if fieldName == loopCtx.IndexVar {
				// Reference loop index variable
				args = append(args, fieldName)
				continue
			}
			if fieldName == loopCtx.ValueVar {
				// Reference loop value variable
				args = append(args, fieldName)
				continue
			}
			// Check if it's a field access on the loop value variable (e.g., user.Name)
			if strings.Contains(fieldName, ".") {
				parts := strings.SplitN(fieldName, ".", 2)
				varName := parts[0]
				if varName == loopCtx.ValueVar {
					// This is a field access on the loop value variable
					// Just use it as-is (e.g., user.Name)
					args = append(args, fieldName)
					continue
				}
			}
		}

		// Type-safety check: does the field exist on the component struct (props or state)?
		_, inProps := currentComp.Schema.Props[strings.ToLower(fieldName)]
		_, inState := currentComp.Schema.State[strings.ToLower(fieldName)]

		if !inProps && !inState {
			// If we're in a loop, provide more context in the error
			if loopCtx != nil {
				allFields := append(getAvailableFieldNames(currentComp.Schema.Props), getAvailableFieldNames(currentComp.Schema.State)...)
				fmt.Fprintf(os.Stderr, "Compilation Error in %s: Field '%s' not found.\n"+
					"  - Not a loop variable (loop has: %s, %s)\n"+
					"  - Not a component field (available: %s)\n"+
					"  - For loop item fields, use: %s.FieldName\n",
					currentComp.Path, fieldName,
					loopCtx.IndexVar, loopCtx.ValueVar,
					strings.Join(allFields, ", "),
					loopCtx.ValueVar)
			} else {
				fmt.Fprintf(os.Stderr, "Compilation Error in %s: Field '%s' not found on component '%s' for data binding.\n",
					currentComp.Path, fieldName, currentComp.PascalName)
			}
			os.Exit(1)
		}
		args = append(args, fmt.Sprintf("%s.%s", receiver, fieldName))
	}

	return fmt.Sprintf(`fmt.Sprintf(%s, %s)`, strconv.Quote(formatString), strings.Join(args, ", "))
}

// generateForLoopCode generates Go for...range loop code for list rendering.
func generateForLoopCode(n *html.Node, receiver string, componentMap map[string]componentInfo, currentComp componentInfo, htmlSource string, opts compileOptions) string {
	// Extract loop variables from data attributes
	indexVar := ""
	valueVar := ""
	rangeExpr := ""
	trackByExpr := ""

	for _, attr := range n.Attr {
		switch attr.Key {
		case "data-index":
			indexVar = attr.Val
		case "data-value":
			valueVar = attr.Val
		case "data-range":
			rangeExpr = attr.Val
		case "data-trackby":
			trackByExpr = attr.Val
		}
	}

	// Validate that we have the required attributes
	if valueVar == "" || rangeExpr == "" || trackByExpr == "" {
		fmt.Fprintf(os.Stderr, "Compilation Error in %s: Invalid {@for} directive - missing required attributes.\n", currentComp.Path)
		os.Exit(1)
	}

	// Validate that the range expression exists on the component
	propDesc, exists := currentComp.Schema.Props[strings.ToLower(rangeExpr)]
	if !exists {
		// Also check state fields
		propDesc, exists = currentComp.Schema.State[strings.ToLower(rangeExpr)]
	}
	if !exists {
		allFields := append(getAvailableFieldNames(currentComp.Schema.Props), getAvailableFieldNames(currentComp.Schema.State)...)
		availableFields := strings.Join(allFields, ", ")
		fmt.Fprintf(os.Stderr, "Compilation Error in %s: Field '%s' not found on component '%s'. Available fields: [%s]\n",
			currentComp.Path, rangeExpr, currentComp.PascalName, availableFields)
		os.Exit(1)
	}

	// Validate that the field is a slice type
	if !strings.HasPrefix(propDesc.GoType, "[]") {
		fmt.Fprintf(os.Stderr, "Compilation Error in %s: Field '%s' must be a slice or array type for {@for} directive, found type '%s'.\n",
			currentComp.Path, rangeExpr, propDesc.GoType)
		os.Exit(1)
	}

	// Validate trackBy expression
	// Supports two formats:
	// 1. Bare variable: "id" (for primitive types like string, int)
	// 2. Dot-notation: "user.ID" (for struct fields)
	trackByParts := strings.Split(trackByExpr, ".")

	var trackByVar, trackByField string

	if len(trackByParts) == 1 {
		// Bare variable format: trackBy id
		trackByVar = trackByParts[0]

		// Verify the variable matches the loop value variable
		if trackByVar != valueVar {
			fmt.Fprintf(os.Stderr, "Compilation Error in %s: trackBy variable '%s' must match the loop value variable '%s'.\n"+
				"  For bare variables, use: trackBy %s\n"+
				"  For struct fields, use: trackBy %s.FieldName\n",
				currentComp.Path, trackByVar, valueVar, valueVar, valueVar)
			os.Exit(1)
		}
	} else if len(trackByParts) >= 2 {
		// Dot-notation format: trackBy user.ID (or nested: user.Profile.ID)
		trackByVar = trackByParts[0]
		trackByField = strings.Join(trackByParts[1:], ".") // Handle nested fields like user.Profile.ID

		// Verify the variable matches the loop value variable
		if trackByVar != valueVar {
			fmt.Fprintf(os.Stderr, "Compilation Error in %s: trackBy variable '%s' must match the loop value variable '%s'.\n"+
				"  For bare variables, use: trackBy %s\n"+
				"  For struct fields, use: trackBy %s.FieldName\n",
				currentComp.Path, trackByVar, valueVar, valueVar, valueVar)
			os.Exit(1)
		}

		// Extract element type from slice type: "[]User" -> "User"
		elementType := strings.TrimPrefix(propDesc.GoType, "[]")

		// Validate that the trackBy field exists on the element type
		// We need to inspect the element type's struct definition
		goFilePath := filepath.Join(filepath.Dir(currentComp.Path), strings.ToLower(currentComp.PascalName)+".go")
		elementSchema, err := inspectStructInFile(goFilePath, elementType)
		if err != nil {
			// If we can't find the struct in the component file, it might be defined elsewhere
			// For now, we'll skip validation with a warning
			fmt.Fprintf(os.Stderr, "Warning in %s: Could not validate trackBy field '%s' on type '%s': %v\n",
				currentComp.Path, trackByField, elementType, err)
		} else {
			// Check if the trackBy field exists on the element type (case-insensitive lookup)
			// For nested fields, only validate the first part
			firstField := strings.Split(trackByField, ".")[0]
			propDescField, exists := elementSchema.Props[strings.ToLower(firstField)]
			if !exists {
				availableFields := strings.Join(getAvailableFieldNames(elementSchema.Props), ", ")
				fmt.Fprintf(os.Stderr, "Compilation Error in %s: trackBy identifier '%s' not found on type '%s'.\nAvailable fields: [%s]\n",
					currentComp.Path, trackByField, elementType, availableFields)
				os.Exit(1)
			}

			// Verify exact case match - the field name in the template must match the actual struct field
			if propDescField.Name != firstField {
				availableFields := strings.Join(getAvailableFieldNames(elementSchema.Props), ", ")
				fmt.Fprintf(os.Stderr, "Compilation Error in %s: trackBy identifier '%s' not found on type '%s'.\nAvailable fields: [%s]\n",
					currentComp.Path, trackByField, elementType, availableFields)
				os.Exit(1)
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "Compilation Error in %s: trackBy expression '%s' must be in one of these formats:\n"+
			"  - Bare variable: trackBy %s (for primitive types)\n"+
			"  - Struct field: trackBy %s.FieldName (for struct types)\n",
			currentComp.Path, trackByExpr, valueVar, valueVar)
		os.Exit(1)
	}

	// Generate the loop body - collect child VNodes
	var code strings.Builder

	// Generate IIFE that returns a slice of VNodes
	code.WriteString("func() []*vdom.VNode {\n")
	code.WriteString(fmt.Sprintf("\tvar %s_nodes []*vdom.VNode\n", valueVar))

	// Add development warning if enabled
	if opts.DevMode {
		code.WriteString(fmt.Sprintf("\t// Development warning for empty slice\n"))
		code.WriteString(fmt.Sprintf("\tif len(%s.%s) == 0 {\n", receiver, propDesc.Name))
		code.WriteString(fmt.Sprintf("\t\tconsole.Warning(\"[@for] Rendering empty list for '%s' in %s. Consider using {@if} to handle empty state.\")\n",
			propDesc.Name, currentComp.PascalName))
		code.WriteString("\t}\n\n")
	}

	// Generate the for loop
	code.WriteString(fmt.Sprintf("\tfor %s, %s := range %s.%s {\n", indexVar, valueVar, receiver, propDesc.Name))

	// Create loop context for child nodes
	loopCtx := &loopContext{
		IndexVar: indexVar,
		ValueVar: valueVar,
	}

	// Generate code for each child node in the loop body
	// Use a counter to ensure unique variable names for each child element
	childCounter := 0
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode || (c.Type == html.TextNode && strings.TrimSpace(c.Data) != "") {
			childCode := generateNodeCode(c, receiver, componentMap, currentComp, htmlSource, opts, loopCtx)
			if childCode != "" {
				childVarName := fmt.Sprintf("%s_child_%d", valueVar, childCounter)
				code.WriteString(fmt.Sprintf("\t\t%s := %s\n", childVarName, childCode))
				code.WriteString(fmt.Sprintf("\t\tif %s != nil {\n", childVarName))
				code.WriteString(fmt.Sprintf("\t\t\t%s_nodes = append(%s_nodes, %s)\n", valueVar, valueVar, childVarName))
				code.WriteString("\t\t}\n")
				childCounter++
			}
		}
	}

	code.WriteString("\t}\n")
	code.WriteString(fmt.Sprintf("\treturn %s_nodes\n", valueVar))
	code.WriteString("}()")

	return code.String()
}

// generateConditionalCode generates Go if/else blocks for conditional rendering.
func generateConditionalCode(n *html.Node, receiver string, componentMap map[string]componentInfo, currentComp componentInfo, htmlSource string, opts compileOptions, loopCtx *loopContext) string {
	var code strings.Builder

	// Generate IIFE (Immediately Invoked Function Expression)
	code.WriteString("func() *vdom.VNode {\n")

	// Process children of go-conditional wrapper
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "go-if" {
			// Extract and validate condition
			cond := ""
			for _, attr := range c.Attr {
				if attr.Key == "data-cond" {
					cond = attr.Val
					break
				}
			}

			propDesc, exists := currentComp.Schema.Props[strings.ToLower(cond)]
			if !exists {
				// Also check state fields
				propDesc, exists = currentComp.Schema.State[strings.ToLower(cond)]
			}
			if !exists {
				fmt.Fprintf(os.Stderr, "Compilation Error in %s: Condition '%s' not found on component '%s'.\n", currentComp.Path, cond, currentComp.PascalName)
				os.Exit(1)
			}
			if propDesc.GoType != "bool" {
				fmt.Fprintf(os.Stderr, "Compilation Error in %s: Condition '%s' must be a bool field, found type '%s'.\n", currentComp.Path, cond, propDesc.GoType)
				os.Exit(1)
			}

			code.WriteString(fmt.Sprintf("if %s.%s {\n", receiver, propDesc.Name))
			foundContent := false
			for cc := c.FirstChild; cc != nil; cc = cc.NextSibling {
				childCode := generateNodeCode(cc, receiver, componentMap, currentComp, htmlSource, opts, loopCtx)
				if childCode != "" {
					code.WriteString("return ")
					code.WriteString(childCode)
					code.WriteString("\n")
					foundContent = true
					break
				}
			}
			if !foundContent {
				code.WriteString("return nil\n")
			}
			code.WriteString("}")
		} else if c.Type == html.ElementNode && c.Data == "go-elseif" {
			// Extract and validate condition
			elseifCond := ""
			for _, attr := range c.Attr {
				if attr.Key == "data-cond" {
					elseifCond = attr.Val
					break
				}
			}

			propDesc, exists := currentComp.Schema.Props[strings.ToLower(elseifCond)]
			if !exists {
				// Also check state fields
				propDesc, exists = currentComp.Schema.State[strings.ToLower(elseifCond)]
			}
			if !exists {
				fmt.Fprintf(os.Stderr, "Compilation Error in %s: Condition '%s' not found on component '%s'.\n", currentComp.Path, elseifCond, currentComp.PascalName)
				os.Exit(1)
			}
			if propDesc.GoType != "bool" {
				fmt.Fprintf(os.Stderr, "Compilation Error in %s: Condition '%s' must be a bool field, found type '%s'.\n", currentComp.Path, elseifCond, propDesc.GoType)
				os.Exit(1)
			}

			code.WriteString(fmt.Sprintf(" else if %s.%s {\n", receiver, propDesc.Name))
			foundContent := false
			for cc := c.FirstChild; cc != nil; cc = cc.NextSibling {
				childCode := generateNodeCode(cc, receiver, componentMap, currentComp, htmlSource, opts, loopCtx)
				if childCode != "" {
					code.WriteString("return ")
					code.WriteString(childCode)
					code.WriteString("\n")
					foundContent = true
					break
				}
			}
			if !foundContent {
				code.WriteString("return nil\n")
			}
			code.WriteString("}")
		} else if c.Type == html.ElementNode && c.Data == "go-else" {
			code.WriteString(" else {\n")
			foundContent := false
			for cc := c.FirstChild; cc != nil; cc = cc.NextSibling {
				childCode := generateNodeCode(cc, receiver, componentMap, currentComp, htmlSource, opts, loopCtx)
				if childCode != "" {
					code.WriteString("return ")
					code.WriteString(childCode)
					code.WriteString("\n")
					foundContent = true
					break
				}
			}
			if !foundContent {
				code.WriteString("return nil\n")
			}
			code.WriteString("}\n")
			// Don't add the fallback return nil after else block
			code.WriteString("}()")
			return code.String()
		}
	}

	// Only add fallback return nil if there's no else branch
	code.WriteString("\nreturn nil\n}()")
	return code.String()
}

// generateNodeCode recursively generates Go vdom calls.
// loopCtx can be nil if not inside a loop.
func generateNodeCode(n *html.Node, receiver string, componentMap map[string]componentInfo, currentComp componentInfo, htmlSource string, opts compileOptions, loopCtx *loopContext) string {
	if n.Type == html.TextNode {
		content := strings.TrimSpace(n.Data)
		if content == "" {
			return ""
		}

		// Generate the text expression (handles data binding, ternaries, static text, etc.)
		lineNum := estimateTextNodeLineNumber(htmlSource, n.Data)
		textExpr := generateTextExpression(content, receiver, currentComp, htmlSource, lineNum, loopCtx)

		// Wrap in vdom.Text() call to create a proper text VNode
		return fmt.Sprintf("vdom.Text(%s)", textExpr)
	}

	if n.Type == html.ElementNode {
		tagName := n.Data

		// 0. Handle conditional placeholder nodes
		if tagName == "go-conditional" {
			return generateConditionalCode(n, receiver, componentMap, currentComp, htmlSource, opts, loopCtx)
		}
		if tagName == "go-if" || tagName == "go-elseif" || tagName == "go-else" {
			// These are handled within go-conditional processing
			return ""
		}

		// 0.5. Handle for-loop placeholder nodes
		if tagName == "go-for" {
			return generateForLoopCode(n, receiver, componentMap, currentComp, htmlSource, opts)
		}

		// 1. Handle Custom Components
		if compInfo, isComponent := componentMap[tagName]; isComponent {
			propsStr := generateStructLiteral(n, compInfo, receiver, componentMap, currentComp, htmlSource, currentComp.Path, opts, loopCtx)

			// Generate key: if inside a loop, include trackBy value for uniqueness
			var key string
			if loopCtx != nil {
				// Inside a loop: use trackBy expression to ensure unique keys
				// Extract trackBy from the parent go-for node
				trackByExpr := extractTrackByFromParent(n)
				if trackByExpr != "" {
					// Use the trackBy value in the key
					key = fmt.Sprintf(`%s_" + fmt.Sprintf("%%v", %s) + "`, compInfo.PascalName, trackByExpr)
				} else {
					// Fallback: use simple counter
					key = fmt.Sprintf("%s_%d", compInfo.PascalName, childCount(n.Parent, n))
				}
			} else {
				// Not in a loop: use simple counter
				key = fmt.Sprintf("%s_%d", compInfo.PascalName, childCount(n.Parent, n))
			}

			return fmt.Sprintf(`r.RenderChild("%s", &%s%s)`, key, compInfo.PascalName, propsStr)
		}

		// 2. Handle Standard HTML Elements
		var childrenCode []string
		hasForLoop := false
		hasSlotSpread := false
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			// Check if this child is a go-for node
			if c.Type == html.ElementNode && c.Data == "go-for" {
				hasForLoop = true
			}
			// Check if this is a slot spread (text node with {SlotField})
			if c.Type == html.TextNode {
				trimmed := strings.TrimSpace(c.Data)
				if matches := dataBindingRegex.FindStringSubmatch(trimmed); len(matches) > 0 {
					fieldName := matches[1]

					// Check if this references the slot field
					if currentComp.Schema.Slot != nil && strings.ToLower(fieldName) == currentComp.Schema.Slot.LowercaseName {
						// This is a slot spread
						hasSlotSpread = true

						// Generate dev warning if enabled
						if opts.DevMode {
							warningCode := fmt.Sprintf("func() []*vdom.VNode {\nif len(%s.%s) == 0 {\nconsole.Warning(\"[Slot] Rendering empty content slot '%s' in component '%s'. Parent provided no content.\")\n}\nreturn %s.%s\n}()...",
								receiver, currentComp.Schema.Slot.Name, currentComp.Schema.Slot.Name, currentComp.PascalName, receiver, currentComp.Schema.Slot.Name)
							childrenCode = append(childrenCode, warningCode)
						} else {
							// No dev warning: just spread the slot
							childrenCode = append(childrenCode, fmt.Sprintf("%s.%s...", receiver, currentComp.Schema.Slot.Name))
						}
						continue
					}

					// Also check regular props and state for backward compatibility
					propDesc, ok := currentComp.Schema.Props[strings.ToLower(fieldName)]
					if !ok {
						propDesc, ok = currentComp.Schema.State[strings.ToLower(fieldName)]
					}
					if ok {
						if propDesc.GoType == "[]*vdom.VNode" {
							// This shouldn't happen anymore since []*vdom.VNode fields are slots
							// But keep this as fallback
							hasSlotSpread = true

							if opts.DevMode {
								warningCode := fmt.Sprintf("func() []*vdom.VNode {\nif len(%s.%s) == 0 {\nconsole.Warning(\"[Slot] Rendering empty content slot '%s' in component '%s'. Parent provided no content.\")\n}\nreturn %s.%s\n}()...",
									receiver, propDesc.Name, propDesc.Name, currentComp.PascalName, receiver, propDesc.Name)
								childrenCode = append(childrenCode, warningCode)
							} else {
								childrenCode = append(childrenCode, fmt.Sprintf("%s.%s...", receiver, propDesc.Name))
							}
							continue
						}
					}
				}
			}
			childCode := generateNodeCode(c, receiver, componentMap, currentComp, htmlSource, opts, loopCtx)
			if childCode != "" {
				childrenCode = append(childrenCode, childCode)
			}
		}

		var childrenStr string
		if hasForLoop || hasSlotSpread {
			// When we have a for loop or slot spread, we need to build children differently
			// Generate code that collects all children into a slice
			childrenStr = "func() []*vdom.VNode {\nvar allChildren []*vdom.VNode\n"
			for _, code := range childrenCode {
				// Check if this looks like a for loop return (starts with "func")
				if strings.HasPrefix(strings.TrimSpace(code), "func()") {
					// For loop or slot with dev warning returns []*vdom.VNode, need spread operator
					if !strings.HasSuffix(code, "...") {
						childrenStr += fmt.Sprintf("allChildren = append(allChildren, %s...)\n", code)
					} else {
						childrenStr += fmt.Sprintf("allChildren = append(allChildren, %s)\n", code)
					}
				} else if strings.HasSuffix(code, "...") {
					// Already has spread operator (e.g., c.SlotField...)
					childrenStr += fmt.Sprintf("allChildren = append(allChildren, %s)\n", code)
				} else {
					// Regular single VNode
					childrenStr += fmt.Sprintf("allChildren = append(allChildren, %s)\n", code)
				}
			}
			childrenStr += "return allChildren\n}()..."
		} else {
			childrenStr = strings.Join(childrenCode, ", ")
		}

		attrsMapStr := generateAttributesMap(n, receiver, currentComp, htmlSource)

		switch tagName {
		case "div":
			return fmt.Sprintf("vdom.Div(%s, %s)", attrsMapStr, childrenStr)
		case "ul", "ol":
			// Handle spread operator for ul/ol elements
			if hasForLoop || hasSlotSpread {
				// childrenStr ends with "...", need to wrap in an IIFE that returns a slice
				return fmt.Sprintf("vdom.NewVNode(%s, %s, %s, \"\")", strconv.Quote(tagName), attrsMapStr, strings.TrimSuffix(childrenStr, "..."))
			} else {
				// Regular children list
				if childrenStr == "" {
					return fmt.Sprintf("vdom.NewVNode(%s, %s, nil, \"\")", strconv.Quote(tagName), attrsMapStr)
				}
				return fmt.Sprintf("vdom.NewVNode(%s, %s, []*vdom.VNode{%s}, \"\")", strconv.Quote(tagName), attrsMapStr, childrenStr)
			}
		case "p", "button", "li", "h1", "h2", "h3", "h4", "h5", "h6":
			textContent := ""
			// Concatenate all text nodes within the element to handle multi-line text
			var textBuilder strings.Builder
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.TextNode {
					textBuilder.WriteString(c.Data)
				}
			}
			fullText := textBuilder.String()
			if fullText != "" {
				// Handle data binding and inline conditionals in the text content
				// Estimate line number by searching for the text in the HTML source
				lineNum := estimateLineNumber(htmlSource, fullText)
				textContent = generateTextExpression(fullText, receiver, currentComp, htmlSource, lineNum, loopCtx)
			} else {
				textContent = `""` // Default to empty string if no text node
			}

			// The VDOM helpers expect a string, so we pass the generated expression
			switch tagName {
			case "p":
				return fmt.Sprintf("vdom.Paragraph(%s, %s)", textContent, attrsMapStr)
			case "button":
				return fmt.Sprintf("vdom.Button(%s, %s, %s)", textContent, attrsMapStr, childrenStr)
			default:
				// For li, h1-h6, use NewVNode directly with text content
				return fmt.Sprintf("vdom.NewVNode(%s, %s, nil, %s)", strconv.Quote(tagName), attrsMapStr, textContent)
			}
		case "input":
			// Handle input element
			return fmt.Sprintf("vdom.NewVNode(%s, %s, nil, \"\")", strconv.Quote(tagName), attrsMapStr)
		case "select":
			// Handle select element with option children
			if hasForLoop || hasSlotSpread {
				return fmt.Sprintf("vdom.NewVNode(%s, %s, %s, \"\")", strconv.Quote(tagName), attrsMapStr, strings.TrimSuffix(childrenStr, "..."))
			} else {
				if childrenStr == "" {
					return fmt.Sprintf("vdom.NewVNode(%s, %s, nil, \"\")", strconv.Quote(tagName), attrsMapStr)
				}
				return fmt.Sprintf("vdom.NewVNode(%s, %s, []*vdom.VNode{%s}, \"\")", strconv.Quote(tagName), attrsMapStr, childrenStr)
			}
		case "option":
			// Handle option element
			textContent := ""
			// Concatenate all text nodes within the element to handle multi-line text
			var textBuilder strings.Builder
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.TextNode {
					textBuilder.WriteString(c.Data)
				}
			}
			fullText := textBuilder.String()
			if fullText != "" {
				lineNum := estimateLineNumber(htmlSource, fullText)
				textContent = generateTextExpression(fullText, receiver, currentComp, htmlSource, lineNum, loopCtx)
			} else {
				textContent = `""`
			}
			return fmt.Sprintf("vdom.NewVNode(%s, %s, nil, %s)", strconv.Quote(tagName), attrsMapStr, textContent)
		case "textarea":
			// Handle textarea element
			return fmt.Sprintf("vdom.NewVNode(%s, %s, nil, \"\")", strconv.Quote(tagName), attrsMapStr)
		case "form":
			// Handle form element with children
			if hasForLoop || hasSlotSpread {
				return fmt.Sprintf("vdom.NewVNode(%s, %s, %s, \"\")", strconv.Quote(tagName), attrsMapStr, strings.TrimSuffix(childrenStr, "..."))
			} else {
				if childrenStr == "" {
					return fmt.Sprintf("vdom.NewVNode(%s, %s, nil, \"\")", strconv.Quote(tagName), attrsMapStr)
				}
				return fmt.Sprintf("vdom.NewVNode(%s, %s, []*vdom.VNode{%s}, \"\")", strconv.Quote(tagName), attrsMapStr, childrenStr)
			}
		case "nav", "a", "span", "section", "article", "header", "footer", "main", "aside":
			// Handle semantic HTML5 elements and inline elements with children
			if hasForLoop || hasSlotSpread {
				return fmt.Sprintf("vdom.NewVNode(%s, %s, %s, \"\")", strconv.Quote(tagName), attrsMapStr, strings.TrimSuffix(childrenStr, "..."))
			} else {
				if childrenStr == "" {
					// Check if there's text content - concatenate all text nodes
					textContent := ""
					var textBuilder strings.Builder
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.TextNode {
							textBuilder.WriteString(c.Data)
						}
					}
					fullText := textBuilder.String()
					if fullText != "" {
						lineNum := estimateLineNumber(htmlSource, fullText)
						textContent = generateTextExpression(fullText, receiver, currentComp, htmlSource, lineNum, loopCtx)
						return fmt.Sprintf("vdom.NewVNode(%s, %s, nil, %s)", strconv.Quote(tagName), attrsMapStr, textContent)
					}
					return fmt.Sprintf("vdom.NewVNode(%s, %s, nil, \"\")", strconv.Quote(tagName), attrsMapStr)
				}
				return fmt.Sprintf("vdom.NewVNode(%s, %s, []*vdom.VNode{%s}, \"\")", strconv.Quote(tagName), attrsMapStr, childrenStr)
			}
		default:
			return `vdom.Div(nil)` // Default to an empty div for unknown tags
		}
	}

	return ""
}

// generateStructLiteral creates the { Field: value, ... } string.
// If the component has a content slot, it collects child nodes and includes them in the struct literal.
func generateStructLiteral(n *html.Node, compInfo componentInfo, receiver string, componentMap map[string]componentInfo, currentComp componentInfo, htmlSource string, templatePath string, opts compileOptions, loopCtx *loopContext) string {
	var props []string

	// Extract the original attribute names from the HTML source
	originalAttrs, lineNumber := extractOriginalAttributesWithLineNumber(n, compInfo.LowercaseName, htmlSource)

	for _, attr := range n.Attr {
		// Get the original casing from the source
		originalKey := attr.Key
		if origName, found := originalAttrs[attr.Key]; found {
			originalKey = origName
		}

		// Check if the ORIGINAL attribute starts with a capital letter
		if len(originalKey) > 0 && originalKey[0] >= 'A' && originalKey[0] <= 'Z' {
			// This is a prop binding - it must match an exported field
			lookupKey := strings.ToLower(originalKey)

			if propDesc, ok := compInfo.Schema.Props[lookupKey]; ok {
				valueStr := convertPropValue(attr.Val, propDesc.GoType, receiver, currentComp, htmlSource, lineNumber, loopCtx)
				props = append(props, fmt.Sprintf("%s: %s", propDesc.Name, valueStr))
			} else {
				// Attribute starts with capital letter but doesn't match any exported field
				availableFields := strings.Join(getAvailableFieldNames(compInfo.Schema.Props), ", ")
				contextLines := getContextLines(htmlSource, lineNumber, 2)
				fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Attribute '%s' does not match any exported field on component '%s'. Available fields: [%s]\n%s",
					templatePath, lineNumber, originalKey, compInfo.PascalName, availableFields, contextLines)
				os.Exit(1)
			}
		} else if propDesc, ok := compInfo.Schema.Props[attr.Key]; ok {
			// Lowercase attribute that happens to match a field
			valueStr := convertPropValue(attr.Val, propDesc.GoType, receiver, currentComp, htmlSource, lineNumber, loopCtx)
			props = append(props, fmt.Sprintf("%s: %s", propDesc.Name, valueStr))
		}
	}

	// Handle content slot if component has one
	if compInfo.Schema.Slot != nil {
		slotContent := collectSlotChildren(n, receiver, componentMap, currentComp, compInfo.PascalName, templatePath, htmlSource, opts, loopCtx)
		if slotContent == "" {
			// Empty slot: compile to nil
			props = append(props, fmt.Sprintf("%s: nil", compInfo.Schema.Slot.Name))
		} else {
			// Has children: compile to VNode slice
			props = append(props, fmt.Sprintf("%s: %s", compInfo.Schema.Slot.Name, slotContent))
		}
	}

	if len(props) == 0 {
		return "{}"
	}

	return fmt.Sprintf("{%s}", strings.Join(props, ", "))
}

// extractOriginalAttributesWithLineNumber extracts the original attribute names and line number from the HTML source.
// This is needed because the HTML parser lowercases all attributes.
func extractOriginalAttributesWithLineNumber(n *html.Node, componentName, htmlSource string) (map[string]string, int) {
	originalAttrs := make(map[string]string)
	lineNumber := 1

	// Find the component tag in the HTML source (case-insensitive tag name)
	// Pattern: <componentName attr1="..." attr2="..." ...>
	pattern := fmt.Sprintf(`(?i)<%s\s+([^>]*)>`, regexp.QuoteMeta(componentName))
	re := regexp.MustCompile(pattern)
	matchIndex := re.FindStringSubmatchIndex(htmlSource)

	if matchIndex == nil || len(matchIndex) < 4 {
		return originalAttrs, lineNumber
	}

	// Calculate line number by counting newlines before the match
	lineNumber = strings.Count(htmlSource[:matchIndex[0]], "\n") + 1

	// Extract the attribute string
	attrString := htmlSource[matchIndex[2]:matchIndex[3]]

	// Extract individual attributes with their original casing
	// Pattern: attrName="value" or attrName='value'
	attrPattern := regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9]*)\s*=\s*["']([^"']*)["']`)
	attrMatches := attrPattern.FindAllStringSubmatch(attrString, -1)

	for _, match := range attrMatches {
		if len(match) >= 2 {
			originalName := match[1]
			lowercaseName := strings.ToLower(originalName)
			originalAttrs[lowercaseName] = originalName
		}
	}

	return originalAttrs, lineNumber
}

// getAvailableFieldNames returns a slice of exported field names for error messages.
func getAvailableFieldNames(props map[string]propertyDescriptor) []string {
	var names []string
	for _, prop := range props {
		names = append(names, prop.Name)
	}
	return names
}

// getAvailableMethodNames returns a comma-separated string of available method names for error messages.
func getAvailableMethodNames(methods map[string]methodDescriptor) string {
	var names []string
	for methodName := range methods {
		names = append(names, methodName)
	}
	return strings.Join(names, ", ")
}

// findEventLineNumber finds the line number where an event attribute is defined.
func findEventLineNumber(n *html.Node, eventName, htmlSource string) int {
	// Look for the event attribute pattern: @eventName="..."
	// We need to find the element's tag and then the specific event attribute
	tagName := n.Data

	// Create a pattern to find this specific element with the event attribute
	// This is a simplified approach - it finds the first occurrence
	pattern := fmt.Sprintf(`(?i)<%s[^>]*@%s\s*=`, regexp.QuoteMeta(tagName), regexp.QuoteMeta(eventName))
	re := regexp.MustCompile(pattern)
	matchIndex := re.FindStringIndex(htmlSource)

	if matchIndex == nil {
		return 1 // Default to line 1 if not found
	}

	// Count newlines before the match to get the line number
	lineNumber := strings.Count(htmlSource[:matchIndex[0]], "\n") + 1
	return lineNumber
}

// getContextLines returns a formatted string with context lines around the error line.
// It shows 'contextSize' lines before and after the target line.
func getContextLines(source string, lineNumber int, contextSize int) string {
	lines := strings.Split(source, "\n")

	// Calculate the range of lines to show
	startLine := lineNumber - contextSize - 1 // -1 for 0-based indexing
	if startLine < 0 {
		startLine = 0
	}

	endLine := lineNumber + contextSize // lineNumber is already the index we want to highlight
	if endLine > len(lines) {
		endLine = len(lines)
	}

	var result strings.Builder
	result.WriteString("\n")

	for i := startLine; i < endLine; i++ {
		lineNum := i + 1
		prefix := "  "

		// Highlight the error line with a marker
		if lineNum == lineNumber {
			prefix = "> "
		}

		result.WriteString(fmt.Sprintf("%s%4d | %s\n", prefix, lineNum, lines[i]))
	}

	return result.String()
}

// convertPropValue generates the Go code to convert a string to the target type.
// It handles data binding expressions in attribute values, respecting loop context.
func convertPropValue(value, goType string, receiver string, currentComp componentInfo, htmlSource string, lineNumber int, loopCtx *loopContext) string {
	switch goType {
	case "string":
		// Check if the value contains data binding expressions
		if dataBindingRegex.MatchString(value) {
			// Use generateTextExpression to handle bindings (including loop variables)
			return generateTextExpression(value, receiver, currentComp, htmlSource, lineNumber, loopCtx)
		}
		return strconv.Quote(value)
	case "int":
		// Check if the value contains data binding expressions (e.g., {UserId})
		if dataBindingRegex.MatchString(value) {
			// Extract the field name from the binding
			matches := dataBindingRegex.FindStringSubmatch(value)
			if len(matches) > 1 {
				fieldName := matches[1]
				// Check if this is a loop variable
				if loopCtx != nil && fieldName == loopCtx.ValueVar {
					// Direct reference to loop value variable
					return fieldName
				} else if loopCtx != nil && strings.HasPrefix(fieldName, loopCtx.ValueVar+".") {
					// Reference to a field of the loop value (e.g., user.ID)
					return fieldName
				} else {
					// Reference to component field
					return fmt.Sprintf("%s.%s", receiver, fieldName)
				}
			}
		}
		// Literal integer value
		return fmt.Sprintf("func() int { i, _ := strconv.Atoi(\"%s\"); return i }()", value)
	case "bool":
		// Check if the value contains data binding expressions (e.g., {IsActive})
		if dataBindingRegex.MatchString(value) {
			// Extract the field name from the binding
			matches := dataBindingRegex.FindStringSubmatch(value)
			if len(matches) > 1 {
				fieldName := matches[1]
				// Check if this is a loop variable
				if loopCtx != nil && fieldName == loopCtx.ValueVar {
					// Direct reference to loop value variable
					return fieldName
				} else if loopCtx != nil && strings.HasPrefix(fieldName, loopCtx.ValueVar+".") {
					// Reference to a field of the loop value
					return fieldName
				} else {
					// Reference to component field
					return fmt.Sprintf("%s.%s", receiver, fieldName)
				}
			}
		}
		// Literal boolean value
		return fmt.Sprintf("func() bool { b, _ := strconv.ParseBool(\"%s\"); return b }()", value)
	default:
		// Default to string for unknown types
		return strconv.Quote(value)
	}
}

// findBody finds the <body> node in the parsed HTML.
func findBody(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "body" {
		return n
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findBody(c); result != nil {
			return result
		}
	}

	return nil
}

// findFirstElementChild finds the first actual element inside a node.
func findFirstElementChild(n *html.Node) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			return c
		}
	}
	return nil
}

// childCount is a helper function to count preceding element siblings for key generation.
func childCount(parent *html.Node, until *html.Node) int {
	count := 0

	if parent == nil {
		return 0
	}

	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if c == until {
			break
		}

		if c.Type == html.ElementNode {
			count++
		}
	}

	return count
}

// extractTrackByFromParent walks up the node tree to find a go-for parent and extracts its trackBy expression.
func extractTrackByFromParent(n *html.Node) string {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == "go-for" {
			// Found the parent loop node, extract trackBy
			for _, attr := range p.Attr {
				if attr.Key == "data-trackby" {
					return attr.Val
				}
			}
		}
	}
	return ""
}

// getSourceLine returns the source line at the given line number (1-indexed).
func getSourceLine(htmlSource string, lineNum int) string {
	lines := strings.Split(htmlSource, "\n")
	if lineNum > 0 && lineNum <= len(lines) {
		return lines[lineNum-1]
	}
	return ""
}

// estimateTextNodeLineNumber estimates the line number where a text node appears in the source.
func estimateTextNodeLineNumber(htmlSource string, textContent string) int {
	// Simple heuristic: find the line containing the trimmed text
	trimmed := strings.TrimSpace(textContent)
	if trimmed == "" {
		return 0
	}

	lines := strings.Split(htmlSource, "\n")
	for i, line := range lines {
		if strings.Contains(line, trimmed) {
			return i + 1 // 1-indexed
		}
	}
	return 0
}

// estimateComponentTagLineNumber finds the line number of a component tag that contains specific text.
// This is more accurate than estimateLineNumber for finding the right occurrence.
func estimateComponentTagLineNumber(htmlSource string, n *html.Node, componentName string) int {
	// Try to find unique characteristics of this specific component usage
	// Look for attributes that make it unique
	var uniqueAttr string
	for _, attr := range n.Attr {
		if attr.Key == "title" || strings.HasPrefix(attr.Key, "Title") {
			uniqueAttr = attr.Val
			break
		}
	}

	searchPattern := fmt.Sprintf("<%s", componentName)
	if uniqueAttr != "" {
		// Search for the tag with this specific attribute
		lines := strings.Split(htmlSource, "\n")
		for i, line := range lines {
			if strings.Contains(line, searchPattern) && strings.Contains(line, uniqueAttr) {
				return i + 1 // 1-indexed
			}
		}
	}

	// Fallback to simple search
	return estimateLineNumber(htmlSource, searchPattern)
}

// generateSlotTextNodeError generates a detailed error message for unwrapped text in slot content.
func generateSlotTextNodeError(
	componentName string,
	templatePath string,
	htmlSource string,
	textNodes []textNodePosition,
	componentTagLine int,
) {
	var errorMsg strings.Builder

	// Header
	firstPos := textNodes[0]
	fmt.Fprintf(&errorMsg, "Compilation Error in %s:%d:%d: Slot content contains unwrapped text in component '%s'\n\n",
		templatePath, firstPos.lineNum, firstPos.colNum, componentName)

	// Show problematic line(s) with context
	for _, pos := range textNodes {
		if pos.lineNum == 0 {
			continue
		}

		line := getSourceLine(htmlSource, pos.lineNum)
		fmt.Fprintf(&errorMsg, "%d | %s\n", pos.lineNum, line)

		// Add caret highlighting
		trimmedText := strings.TrimSpace(pos.textContent)
		colOffset := strings.Index(line, trimmedText)
		if colOffset >= 0 {
			padding := strings.Repeat(" ", len(fmt.Sprintf("%d | ", pos.lineNum))+colOffset)
			carets := strings.Repeat("^", len(trimmedText))
			fmt.Fprintf(&errorMsg, "%s%s\n", padding, carets)
		}
	}

	// Suggestion: multi-line format
	fmt.Fprintf(&errorMsg, "\nSlot content must be wrapped in HTML elements. Wrap the text in a tag:\n\n")
	fmt.Fprintf(&errorMsg, "%d | <%s ...>\n", componentTagLine, componentName)
	fmt.Fprintf(&errorMsg, "%d |   <p>Text content here</p>\n", componentTagLine+1)
	fmt.Fprintf(&errorMsg, "%d | </%s>\n", componentTagLine+2, componentName)

	// Suggestion: inline format
	fmt.Fprintf(&errorMsg, "\nOr use <span> for inline text:\n\n")
	fmt.Fprintf(&errorMsg, "%d | <%s ...><span>Text content</span></%s>\n",
		componentTagLine, componentName, componentName)

	// Print to stderr and exit
	fmt.Fprint(os.Stderr, errorMsg.String())
	os.Exit(1)
}

// collectSlotChildren collects child nodes for content projection and generates VNode slice code.
// Returns empty string if no children, otherwise returns Go code for []*vdom.VNode{...}
// Validates that slot content does not contain unwrapped text nodes.
func collectSlotChildren(n *html.Node, receiver string, componentMap map[string]componentInfo, currentComp componentInfo, componentName string, templatePath string, htmlSource string, opts compileOptions, loopCtx *loopContext) string {
	var childrenCode []string

	// Collect all children (elements and text nodes)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			// Check if this is meaningful text (not just whitespace)
			trimmed := strings.TrimSpace(c.Data)
			if trimmed != "" {
				// Convert text node to pure text VNode using vdom.Text()
				textExpr := generateTextExpression(trimmed, receiver, currentComp, htmlSource, estimateTextNodeLineNumber(htmlSource, c.Data), loopCtx)
				childrenCode = append(childrenCode, fmt.Sprintf(`vdom.Text(%s)`, textExpr))
			}
			// Skip whitespace-only text nodes
			continue
		}

		childCode := generateNodeCode(c, receiver, componentMap, currentComp, htmlSource, opts, loopCtx)
		if childCode != "" {
			childrenCode = append(childrenCode, childCode)
		}
	}

	if len(childrenCode) == 0 {
		return "" // No children, will compile to nil
	}

	return fmt.Sprintf("[]*vdom.VNode{%s}", strings.Join(childrenCode, ", "))
}
