package compiler

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

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
					// Fallback: use a template-wide counter so keys are unique across the whole template
					count := opts.ComponentCounter[compInfo.PascalName]
					opts.ComponentCounter[compInfo.PascalName]++
					key = fmt.Sprintf("%s_%d", compInfo.PascalName, count)
				}
			} else {
				// Not in a loop: use a template-wide counter so keys are unique across the whole template
				// (sibling-position would give the same key to components at the same depth in different
				// parent containers, e.g. multiple RouterLinks each at position 3 all become RouterLink_3)
				count := opts.ComponentCounter[compInfo.PascalName]
				opts.ComponentCounter[compInfo.PascalName]++
				key = fmt.Sprintf("%s_%d", compInfo.PascalName, count)
			}

			// Determine if we need a qualified name (cross-package reference)
			var componentRef string
			if compInfo.PackageName != currentComp.PackageName {
				// Cross-package: use qualified name
				componentRef = fmt.Sprintf("%s.%s", compInfo.PackageName, compInfo.PascalName)
			} else {
				// Same package: use unqualified name
				componentRef = compInfo.PascalName
			}

			return fmt.Sprintf(`r.RenderChild("%s", &%s%s)`, key, componentRef, propsStr)
		}

		// 1.5. Check if this is a PascalCase tag that looks like a component but wasn't found
		// Note: The HTML parser lowercases tag names, so we need to find the original casing in htmlSource
		originalTagName := findOriginalTagName(n, tagName, htmlSource)
		if isComponentTag(originalTagName) {
			lineNumber := estimateLineNumber(htmlSource, fmt.Sprintf("<%s", originalTagName))
			errorMsg := generateMissingComponentError(originalTagName, componentMap, currentComp, htmlSource, currentComp.Path, lineNumber)
			fmt.Fprint(os.Stderr, errorMsg)
			os.Exit(1)
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
							warningCode := fmt.Sprintf("func() []*vdom.VNode {\nif len(%s.%s) == 0 {\nconsole.Warn(\"[Slot] Rendering empty content slot '%s' in component '%s'. Parent provided no content.\")\n}\nreturn %s.%s\n}()...",
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
								warningCode := fmt.Sprintf("func() []*vdom.VNode {\nif len(%s.%s) == 0 {\nconsole.Warn(\"[Slot] Rendering empty content slot '%s' in component '%s'. Parent provided no content.\")\n}\nreturn %s.%s\n}()...",
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
			hasElementChildren := false
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.TextNode {
					textBuilder.WriteString(c.Data)
				} else if c.Type == html.ElementNode {
					hasElementChildren = true
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
				// If there are child elements (e.g. <span>), render as a full VNode with children
				// so that inline elements are not silently dropped.
				if hasElementChildren {
					if childrenStr == "" {
						return fmt.Sprintf("vdom.NewVNode(\"p\", %s, nil, \"\")", attrsMapStr)
					}
					return fmt.Sprintf("vdom.NewVNode(\"p\", %s, []*vdom.VNode{%s}, \"\")", attrsMapStr, childrenStr)
				}
				return fmt.Sprintf("vdom.Paragraph(%s, %s)", textContent, attrsMapStr)
			case "button":
				// Always use children for button content (childrenStr already contains vdom.Text(...)
				// for any plain-text children). Passing textContent as well would create redundant
				// dual-storage (both Content and Children set), which breaks patch updates.
				if childrenStr == "" {
					return fmt.Sprintf("vdom.Button(\"\", %s)", attrsMapStr)
				}
				return fmt.Sprintf("vdom.Button(\"\", %s, %s)", attrsMapStr, childrenStr)
			case "li":
				// For li elements, check if there are child components/elements (not just text)
				hasElementChildren := false
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode {
						hasElementChildren = true
						break
					}
				}

				if !hasElementChildren {
					// Only text content - render with text parameter
					return fmt.Sprintf("vdom.NewVNode(%s, %s, nil, %s)", strconv.Quote(tagName), attrsMapStr, textContent)
				}

				// Has component or element children - render them properly
				if hasForLoop || hasSlotSpread {
					return fmt.Sprintf("vdom.NewVNode(%s, %s, %s, \"\")", strconv.Quote(tagName), attrsMapStr, strings.TrimSuffix(childrenStr, "..."))
				} else {
					return fmt.Sprintf("vdom.NewVNode(%s, %s, []*vdom.VNode{%s}, \"\")", strconv.Quote(tagName), attrsMapStr, childrenStr)
				}
			default:
				// For h1-h6, use NewVNode directly with text content
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
		case "img", "br", "hr", "wbr":
			// Void elements — no children or text content
			return fmt.Sprintf("vdom.NewVNode(%s, %s, nil, \"\")", strconv.Quote(tagName), attrsMapStr)
		default:
			return `vdom.Div(nil)` // Default to an empty div for unknown tags
		}
	}

	return ""
}

// isComponentTag checks if a tag name follows the component naming convention (PascalCase).
// Component names must start with an uppercase letter to distinguish them from HTML elements.
func isComponentTag(tagName string) bool {
	if len(tagName) == 0 {
		return false
	}
	// Component tags start with uppercase letter
	return tagName[0] >= 'A' && tagName[0] <= 'Z'
}

// findOriginalTagName finds the original tag name (with original casing) from the HTML source.
// The Go html parser lowercases all tag names, so we need to search the source to find the original.
func findOriginalTagName(n *html.Node, lowercasedName string, htmlSource string) string {
	// Search for the tag opening in the HTML source, case-insensitively
	// Look for patterns like <TagName or <Tagname...
	searchPatternAny := fmt.Sprintf("(?i)<%s[>\\s]", lowercasedName)

	// Use regex to find the pattern case-insensitively
	re := regexp.MustCompile(searchPatternAny)
	matches := re.FindAllString(htmlSource, -1)

	if len(matches) > 0 {
		// Extract the tag name from the first match
		// Format: "<TagName" or "<Tagname "
		match := matches[0]
		endIdx := strings.IndexAny(match, "> \t\n")
		if endIdx > 0 {
			return match[1:endIdx]
		}
		return match[1 : len(match)-1]
	}

	// Fallback: return the lowercased name
	return lowercasedName
}
