package compiler

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ForgeLogic/nojs/events"
	"golang.org/x/net/html"
)

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

	if len(matchIndex) < 4 {
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

// convertPropValue generates the Go code to convert a string to the target type.
// It handles data binding expressions in attribute values, respecting loop context.
func convertPropValue(value, goType string, receiver string, currentComp componentInfo, htmlSource string, lineNumber int, loopCtx *loopContext) string {
	// Debug: uncomment to see what values are being converted
	// fmt.Fprintf(os.Stderr, "[convertPropValue] value=%q goType=%q\n", value, goType)

	// First, check if value is wrapped in braces {}: if so, extract and handle as expression
	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		// Extract the Go code from within braces
		goCode := strings.TrimSuffix(strings.TrimPrefix(value, "{"), "}")

		// For boolean literals, use as-is
		if goCode == "true" || goCode == "false" {
			return goCode
		}

		// For qualified names (e.g., modal.Information), use as-is
		if strings.Contains(goCode, ".") && !strings.Contains(goCode, "(") {
			return goCode
		}

		// For simple identifiers (component fields or loop variables)
		if !strings.Contains(goCode, " ") && !strings.Contains(goCode, "(") {
			// Check if this is a loop variable
			if loopCtx != nil && (goCode == loopCtx.ValueVar || goCode == loopCtx.IndexVar || strings.HasPrefix(goCode, loopCtx.ValueVar+".")) {
				return goCode
			}

			// Check if it's a component field (props or state)
			propDesc, inProps := currentComp.Schema.Props[strings.ToLower(goCode)]
			if !inProps {
				propDesc, inProps = currentComp.Schema.State[strings.ToLower(goCode)]
			}
			if inProps {
				// It's a component field - add receiver prefix
				return fmt.Sprintf("%s.%s", receiver, propDesc.Name)
			}
		}

		// For everything else (e.g., method names, complex expressions), use as-is
		return goCode
	}

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
		// For unknown/custom types (enums, custom structs, etc.):
		// - If value is a simple identifier, check if it's a method name (for function types)
		if !strings.Contains(value, ".") && !strings.Contains(value, "(") && !strings.Contains(value, " ") {
			// Check if this is a function type - if so, treat as method name
			if strings.HasPrefix(goType, "func") {
				// It's a function type - convert method name to receiver reference
				return fmt.Sprintf("%s.%s", receiver, value)
			}

			// For non-function types, it might be a constant - use as-is
			// (e.g., mypackage.SomeConstant will work, but SomeConstant alone might not)
			return value
		}

		// If it contains a dot, it's likely a qualified name or constant - use as-is
		if strings.Contains(value, ".") {
			return value
		}

		// Default to string for unknown types
		return strconv.Quote(value)
	}
}
