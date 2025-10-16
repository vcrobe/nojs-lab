package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/tools/go/packages"
)

// componentSchema holds the type information for a component's props.
type componentSchema struct {
	Props   map[string]propertyDescriptor // Map of Prop name to its Go type (e.g., "Title": "string")
	Methods map[string]bool               // Set of available method names for event handlers
}

type propertyDescriptor struct {
	Name          string
	LowercaseName string
	GoType        string
}

// componentInfo holds all discovered information about a component.
type componentInfo struct {
	Path          string
	PascalName    string
	LowercaseName string
	PackageName   string
	Schema        componentSchema
}

// Regex to find data binding expressions like {FieldName}
var dataBindingRegex = regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)

// Compile is the main entry point for the AOT compiler.
func compile(srcDir, outDir string) error {
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
		if err := compileComponentTemplate(comp, componentMap, outDir); err != nil {
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
					Methods: make(map[string]bool),
				}
			}

			components = append(components, componentInfo{
				Path:          templatePath,
				PascalName:    pascalName,
				LowercaseName: strings.ToLower(pascalName),
				PackageName:   pkg.Name, // Use the package name from the loader.
				Schema:        schema,
			})
		}
	}

	if len(components) == 0 {
		fmt.Println("Warning: No component templates (*.gt.html) were found in any Go packages.")
	}

	return components, nil
}

// inspectGoFile parses a Go file and extracts the prop schema for a given struct.
func inspectGoFile(path, structName string) (componentSchema, error) {
	schema := componentSchema{
		Props:   make(map[string]propertyDescriptor),
		Methods: make(map[string]bool),
	}
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return schema, err
	}

	ast.Inspect(node, func(n ast.Node) bool {
		// Inspect for struct fields (Props)
		if typeSpec, ok := n.(*ast.TypeSpec); ok && typeSpec.Name.Name == structName {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				for _, field := range structType.Fields.List {
					if len(field.Names) > 0 && field.Names[0].IsExported() {
						fieldName := field.Names[0].Name
						if typeIdent, ok := field.Type.(*ast.Ident); ok {
							schema.Props[strings.ToLower(fieldName)] = propertyDescriptor{
								Name:          fieldName,
								LowercaseName: strings.ToLower(fieldName),
								GoType:        typeIdent.Name,
							}
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
					schema.Methods[funcDecl.Name.Name] = true
				}
			}
		}

		return true
	})

	return schema, nil
}

// compileComponentTemplate handles the code generation for a single component.
func compileComponentTemplate(comp componentInfo, componentMap map[string]componentInfo, outDir string) error {
	htmlContent, err := os.ReadFile(comp.Path)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", comp.Path, err)
	}
	htmlString := string(htmlContent)

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
	var generatedCode string
	generatedCode = generateNodeCode(rootElement, "c", componentMap, comp, htmlString)

	template := `//go:build js || wasm
// +build js wasm

// Code generated by the nojs AOT compiler. DO NOT EDIT.
package %[2]s

import (
	"fmt"
	"github.com/vcrobe/nojs/vdom"
	"github.com/vcrobe/nojs/runtime"
	"strconv" // Added for type conversions
)

// Render generates the VNode tree for the %[1]s component.
func (c *%[1]s) Render(r *runtime.Renderer) *vdom.VNode {
	_ = strconv.Itoa // Suppress unused import error if no props are converted
	_ = fmt.Sprintf  // Suppress unused import error if no bindings are used
	return %[3]s
}
`

	source := fmt.Sprintf(template, comp.PascalName, comp.PackageName, generatedCode)
	outFileName := fmt.Sprintf("%s.generated.go", comp.PascalName)
	outFilePath := filepath.Join(outDir, outFileName)
	return os.WriteFile(outFilePath, []byte(source), 0644)
}

// generateAttributesMap is a helper to create the Go map literal for an element's attributes.
func generateAttributesMap(n *html.Node, receiver string, currentComp componentInfo, htmlSource string) string {
	var attrs, events []string
	for _, a := range n.Attr {
		if strings.HasPrefix(a.Key, "@") {
			eventName := strings.TrimPrefix(a.Key, "@")
			handlerName := a.Val
			// Compile-time safety check!
			if _, ok := currentComp.Schema.Methods[handlerName]; !ok {
				// Find the line number for this event attribute
				lineNumber := findEventLineNumber(n, eventName, htmlSource)
				availableMethods := getAvailableMethodNames(currentComp.Schema.Methods)
				contextLines := getContextLines(htmlSource, lineNumber, 2)
				fmt.Fprintf(os.Stderr, "Compilation Error in %s:%d: Method '%s' not found on component '%s'. Available methods: [%s]\n%s",
					currentComp.Path, lineNumber, handlerName, currentComp.PascalName, availableMethods, contextLines)
				os.Exit(1)
			}
			switch eventName {
			case "onclick":
				// Generate the Go code to reference the component's method.
				//events = append(events, fmt.Sprintf(`"onClick": %s.%s`, receiver, handlerName))
				handler := fmt.Sprintf(`%s.%s`, receiver, handlerName)
				events = append(events, fmt.Sprintf(`"onClick": %s`, handler))
			default:
				fmt.Printf("Warning: Unknown event directive '@%s' in %s.\n", eventName, currentComp.Path)
			}
		} else {
			attrs = append(attrs, fmt.Sprintf(`"%s": "%s"`, a.Key, a.Val))
		}
	}

	if len(attrs) == 0 && len(events) == 0 {
		return "nil"
	}
	allProps := append(attrs, events...)
	return fmt.Sprintf("map[string]any{%s}", strings.Join(allProps, ", "))
}

// generateTextExpression handles data binding in text nodes.
func generateTextExpression(text string, receiver string, currentComp componentInfo) string {
	matches := dataBindingRegex.FindAllStringSubmatch(text, -1)

	if len(matches) == 0 {
		return strconv.Quote(text) // It's just a static string
	}

	formatString := dataBindingRegex.ReplaceAllString(text, "%v")
	var args []string

	for _, match := range matches {
		fieldName := match[1]
		// Type-safety check: does the field exist on the component struct?
		if _, ok := currentComp.Schema.Props[strings.ToLower(fieldName)]; !ok {
			fmt.Fprintf(os.Stderr, "Compilation Error in %s: Field '%s' not found on component '%s' for data binding.\n", currentComp.Path, fieldName, currentComp.PascalName)
			os.Exit(1)
		}
		args = append(args, fmt.Sprintf("%s.%s", receiver, fieldName))
	}

	return fmt.Sprintf(`fmt.Sprintf("%s", %s)`, formatString, strings.Join(args, ", "))
}

// generateNodeCode recursively generates Go vdom calls.
func generateNodeCode(n *html.Node, receiver string, componentMap map[string]componentInfo, currentComp componentInfo, htmlSource string) string {
	if n.Type == html.TextNode {
		content := strings.TrimSpace(n.Data)
		if content == "" {
			return ""
		}
		// In a real scenario, you'd handle text content more robustly.
		// For now, we assume text is primarily for simple elements like <p>.
		return ""
	}

	if n.Type == html.ElementNode {
		tagName := n.Data

		// 1. Handle Custom Components
		if compInfo, isComponent := componentMap[tagName]; isComponent {
			propsStr := generateStructLiteral(n, compInfo, htmlSource, currentComp.Path)
			key := fmt.Sprintf("%s_%d", compInfo.PascalName, childCount(n.Parent, n)) // Simple key generation

			return fmt.Sprintf(`r.RenderChild("%s", &%s%s)`, key, compInfo.PascalName, propsStr)
		}

		// 2. Handle Standard HTML Elements
		var childrenCode []string
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			childCode := generateNodeCode(c, receiver, componentMap, currentComp, htmlSource)
			if childCode != "" {
				childrenCode = append(childrenCode, childCode)
			}
		}

		childrenStr := strings.Join(childrenCode, ", ")
		attrsMapStr := generateAttributesMap(n, receiver, currentComp, htmlSource)

		switch tagName {
		case "div":
			return fmt.Sprintf("vdom.Div(%s, %s)", attrsMapStr, childrenStr)
		case "p", "button":
			textContent := ""
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				// Handle data binding in the text content
				textContent = generateTextExpression(n.FirstChild.Data, receiver, currentComp)
			} else {
				textContent = `""` // Default to empty string if no text node
			}

			// The VDOM helpers expect a string, so we pass the generated expression
			if tagName == "p" {
				return fmt.Sprintf("vdom.Paragraph(%s, %s)", textContent, attrsMapStr)
			}
			return fmt.Sprintf("vdom.Button(%s, %s, %s)", textContent, attrsMapStr, childrenStr)
		default:
			return `vdom.Div(nil)` // Default to an empty div for unknown tags
		}
	}

	return ""
}

// generateStructLiteral creates the { Field: value, ... } string.
func generateStructLiteral(n *html.Node, compInfo componentInfo, htmlSource string, templatePath string) string {
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
				valueStr := convertPropValue(attr.Val, propDesc.GoType)
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
			valueStr := convertPropValue(attr.Val, propDesc.GoType)
			props = append(props, fmt.Sprintf("%s: %s", propDesc.Name, valueStr))
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
func getAvailableMethodNames(methods map[string]bool) string {
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
func convertPropValue(value, goType string) string {
	switch goType {
	case "string":
		return strconv.Quote(value)
	case "int":
		// In a real compiler, you'd handle the error. Here we assume valid input.
		return fmt.Sprintf("func() int { i, _ := strconv.Atoi(\"%s\"); return i }()", value)
	case "bool":
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
