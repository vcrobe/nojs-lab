package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

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

		// Check if this is a nested field access (e.g., Ctx.Title)
		if strings.Contains(fieldName, ".") {
			rootField := strings.ToLower(strings.SplitN(fieldName, ".", 2)[0])

			// Check if root field exists on component
			if _, inProps := currentComp.Schema.Props[rootField]; inProps {
				// Resolve the nested field type
				componentDir := filepath.Dir(currentComp.Path)
				_, err := resolveNestedFieldType(fieldName, currentComp, componentDir)
				if err != nil {
					// Try to get available fields on the nested type for better error message
					nestedFields := getAvailableNestedFields(fieldName, currentComp, componentDir)
					allFields := append(getAvailableFieldNames(currentComp.Schema.Props), getAvailableFieldNames(currentComp.Schema.State)...)

					var msg string
					if len(nestedFields) > 0 {
						msg = fmt.Sprintf("Compilation Error in %s: Field '%s' not resolvable on component '%s'. %v\n\nAvailable component fields: [%s]\nAvailable fields on %s: [%s]\n",
							currentComp.Path, fieldName, currentComp.PascalName, err, strings.Join(allFields, ", "),
							strings.SplitN(fieldName, ".", 2)[0], strings.Join(nestedFields, ", "))
					} else {
						msg = fmt.Sprintf("Compilation Error in %s: Field '%s' not resolvable on component '%s'. %v\nAvailable fields: [%s]\n",
							currentComp.Path, fieldName, currentComp.PascalName, err, strings.Join(allFields, ", "))
					}
					fmt.Fprint(os.Stderr, msg)
					os.Exit(1)
				}
				// Use nested field access as-is
				args = append(args, fmt.Sprintf("%s.%s", receiver, fieldName))
				continue
			} else if _, inState := currentComp.Schema.State[rootField]; inState {
				// Resolve the nested field type
				componentDir := filepath.Dir(currentComp.Path)
				_, err := resolveNestedFieldType(fieldName, currentComp, componentDir)
				if err != nil {
					// Try to get available fields on the nested type for better error message
					nestedFields := getAvailableNestedFields(fieldName, currentComp, componentDir)
					allFields := append(getAvailableFieldNames(currentComp.Schema.Props), getAvailableFieldNames(currentComp.Schema.State)...)

					var msg string
					if len(nestedFields) > 0 {
						msg = fmt.Sprintf("Compilation Error in %s: Field '%s' not resolvable on component '%s'. %v\n\nAvailable component fields: [%s]\nAvailable fields on %s: [%s]\n",
							currentComp.Path, fieldName, currentComp.PascalName, err, strings.Join(allFields, ", "),
							strings.SplitN(fieldName, ".", 2)[0], strings.Join(nestedFields, ", "))
					} else {
						msg = fmt.Sprintf("Compilation Error in %s: Field '%s' not resolvable on component '%s'. %v\nAvailable fields: [%s]\n",
							currentComp.Path, fieldName, currentComp.PascalName, err, strings.Join(allFields, ", "))
					}
					fmt.Fprint(os.Stderr, msg)
					os.Exit(1)
				}
				// Use nested field access as-is
				args = append(args, fmt.Sprintf("%s.%s", receiver, fieldName))
				continue
			}
		}

		// Type-safety check: does the field exist on the component struct (props or state)?
		propDesc, inProps := currentComp.Schema.Props[strings.ToLower(fieldName)]
		stateDesc, inState := currentComp.Schema.State[strings.ToLower(fieldName)]

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
		// Use the schema's correctly-cased field name, not the raw template expression,
		// so that e.g. {id} in the template correctly emits c.ID (not c.id).
		var resolvedName string
		if inProps {
			resolvedName = propDesc.Name
		} else {
			resolvedName = stateDesc.Name
		}
		args = append(args, fmt.Sprintf("%s.%s", receiver, resolvedName))
	}

	return fmt.Sprintf(`fmt.Sprintf(%s, %s)`, strconv.Quote(formatString), strings.Join(args, ", "))
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
// Returns empty string if no children, otherwise returns Go code for []*vdom.VNode{...}.
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
