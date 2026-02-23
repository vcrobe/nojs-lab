package compiler

import (
	"fmt"
	"os"
	"strings"

	"github.com/ForgeLogic/nojs/events"
)

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

// levenshteinDistance calculates the edit distance between two strings.
// Used for fuzzy matching component name suggestions.
// Returns the minimum number of single-character edits (insertions, deletions, substitutions)
// needed to transform one string into the other.
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Use dynamic programming with space-optimized approach
	// (only need previous row, not full matrix)
	if len(a) > len(b) {
		a, b = b, a
	}

	prevRow := make([]int, len(a)+1)
	currRow := make([]int, len(a)+1)

	// Initialize first row
	for j := 0; j <= len(a); j++ {
		prevRow[j] = j
	}

	// Compute distances
	for i := 1; i <= len(b); i++ {
		currRow[0] = i

		for j := 1; j <= len(a); j++ {
			cost := 0
			if a[j-1] != b[i-1] {
				cost = 1
			}

			currRow[j] = min(
				currRow[j-1]+1,    // insertion
				prevRow[j]+1,      // deletion
				prevRow[j-1]+cost, // substitution
			)
		}

		prevRow, currRow = currRow, prevRow
	}

	return prevRow[len(a)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// findSimilarComponents finds component names similar to the given name using fuzzy matching.
// Returns suggestions with edit distance <= threshold (default 2 for typos).
// Results are sorted by distance (closest first).
func findSimilarComponents(typedName string, availableComponents []componentInfo) []componentInfo {
	const threshold = 2

	type suggestion struct {
		comp     componentInfo
		distance int
	}

	var suggestions []suggestion

	for _, comp := range availableComponents {
		dist := levenshteinDistance(strings.ToLower(typedName), strings.ToLower(comp.PascalName))
		if dist <= threshold {
			suggestions = append(suggestions, suggestion{comp, dist})
		}
	}

	// Sort by distance (closest first)
	for i := 0; i < len(suggestions); i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[j].distance < suggestions[i].distance {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}

	// Return only the componentInfo (up to 3 suggestions)
	var result []componentInfo
	maxSuggestions := 3
	for i := 0; i < len(suggestions) && i < maxSuggestions; i++ {
		result = append(result, suggestions[i].comp)
	}

	return result
}

// generateMissingComponentError generates a detailed error message for an unknown component.
func generateMissingComponentError(tagName string, componentMap map[string]componentInfo, currentComp componentInfo, htmlSource string, templatePath string, lineNumber int) string {
	var errorMsg strings.Builder

	errorMsg.WriteString(fmt.Sprintf("Compilation Error in %s:%d:\n", templatePath, lineNumber))
	errorMsg.WriteString(fmt.Sprintf("Component '<%s>' not found.\n\n", tagName))

	// Collect all available components
	var allComponents []componentInfo
	for _, comp := range componentMap {
		allComponents = append(allComponents, comp)
	}

	// Find similar components for suggestions
	similar := findSimilarComponents(tagName, allComponents)
	if len(similar) > 0 {
		errorMsg.WriteString("Did you mean one of these?\n")
		for _, comp := range similar {
			errorMsg.WriteString(fmt.Sprintf("  - <%s>\n", comp.PascalName))
		}
		errorMsg.WriteString("\n")
	}

	// Show context from the template
	contextLines := getContextLines(htmlSource, lineNumber, 2)
	if contextLines != "" {
		errorMsg.WriteString("Context:\n")
		errorMsg.WriteString(contextLines)
		errorMsg.WriteString("\n")
	}

	// List all available components (first 10)
	if len(allComponents) > 0 {
		errorMsg.WriteString("Available components in this project:\n")
		count := 0
		for _, comp := range allComponents {
			if count >= 10 {
				errorMsg.WriteString(fmt.Sprintf("  ... and %d more\n", len(allComponents)-10))
				break
			}
			errorMsg.WriteString(fmt.Sprintf("  - %s\n", comp.PascalName))
			count++
		}
		errorMsg.WriteString("\n")
	}

	// Helpful notes
	errorMsg.WriteString("Tips to fix this:\n")
	errorMsg.WriteString("  1. Check the component name spelling (PascalCase, e.g., <MyComponent>)\n")
	errorMsg.WriteString("  2. Ensure the component has a *.gt.html template file\n")
	errorMsg.WriteString("  3. If the component is in another package, import it in your Go code\n")

	return errorMsg.String()
}
