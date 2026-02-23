package compiler

import "regexp"

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
	ImportPath    string // Full import path (e.g., "github.com/ForgeLogic/nojs/appcomponents")
	Schema        componentSchema
}

// compileOptions holds compiler-wide options passed from CLI flags.
type compileOptions struct {
	DevMode          bool           // Enable development mode (warnings, verbose errors, panic on lifecycle failures)
	ComponentCounter map[string]int // Template-wide counter per component type for unique RenderChild keys
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
