package compiler

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/net/html"
)

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
