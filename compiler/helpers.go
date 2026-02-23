package compiler

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

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

// getSourceLine returns the source line at the given line number (1-indexed).
func getSourceLine(htmlSource string, lineNum int) string {
	lines := strings.Split(htmlSource, "\n")
	if lineNum > 0 && lineNum <= len(lines) {
		return lines[lineNum-1]
	}
	return ""
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
