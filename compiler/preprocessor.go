package compiler

import (
	"fmt"
	"regexp"
	"strings"
)

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
