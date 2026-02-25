package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

// generateForLoopCode generates Go for...range loop code for list rendering.
func generateForLoopCode(n *html.Node, receiver string, componentMap map[string]componentInfo, currentComp componentInfo, htmlSource string, opts compileOptions) string {
	// Extract loop variables from data attributes
	indexVar := ""
	valueVar := ""
	rangeExpr := ""
	trackByExpr := ""

	error

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
		code.WriteString("\t// Development warning for empty slice\n")
		code.WriteString(fmt.Sprintf("\tif len(%s.%s) == 0 {\n", receiver, propDesc.Name))
		code.WriteString(fmt.Sprintf("\t\tconsole.Warn(\"[@for] Rendering empty list for '%s' in %s. Consider using {@if} to handle empty state.\")\n",
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
