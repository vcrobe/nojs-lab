package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/tools/go/packages"
)

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
					Methods: make(map[string]methodDescriptor),
				}
			}

			components = append(components, componentInfo{
				Path:          templatePath,
				PascalName:    pascalName,
				LowercaseName: strings.ToLower(pascalName),
				PackageName:   pkg.Name,    // Use the package name from the loader.
				ImportPath:    pkg.PkgPath, // Full import path (e.g., "github.com/ForgeLogic/nojs/appcomponents")
				Schema:        schema,
			})

			// Validate that component name doesn't conflict with HTML tags
			if err := validateComponentName(pascalName, templatePath); err != nil {
				return nil, err
			}
		}
	}

	if len(components) == 0 {
		fmt.Println("Warning: No component templates (*.gt.html) were found in any Go packages.")
	}

	return components, nil
}

// collectUsedComponents walks the HTML tree and collects all components used from other packages.
// Returns a map of package name to import path.
func collectUsedComponents(n *html.Node, componentMap map[string]componentInfo, currentComp componentInfo) map[string]string {
	usedPackages := make(map[string]string)

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			tagName := node.Data
			// Check if this is a component
			if compInfo, isComponent := componentMap[tagName]; isComponent {
				// Check if it's from a different package
				if compInfo.PackageName != currentComp.PackageName {
					// Need to import this package
					// Store mapping: package name -> full import path
					usedPackages[compInfo.PackageName] = compInfo.ImportPath
				}
			}
		}

		// Recurse into children
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(n)
	return usedPackages
}

// extractTypeName extracts the type name from an AST expression.
// Handles simple types (int, string, bool), slice types ([]User), pointer types (*User), and function types.
func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		// Simple type like "string", "int", "bool"
		return t.Name
	case *ast.ArrayType:
		// Slice or array type like "[]User"
		elemType := extractTypeName(t.Elt)
		return "[]" + elemType
	case *ast.StarExpr:
		// Pointer type like "*User"
		elemType := extractTypeName(t.X)
		return "*" + elemType
	case *ast.SelectorExpr:
		// Qualified type like "time.Time"
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
	case *ast.FuncType:
		// Function type like "func(result ModalResult)" or "func() string"
		// We return a simplified representation that starts with "func"
		// to allow type checking in prop conversion
		var paramTypes []string
		if t.Params != nil {
			for _, param := range t.Params.List {
				paramTypes = append(paramTypes, extractTypeName(param.Type))
			}
		}

		var returnTypes []string
		if t.Results != nil {
			for _, result := range t.Results.List {
				returnTypes = append(returnTypes, extractTypeName(result.Type))
			}
		}

		// Build a string representation like "func(Type1, Type2) ReturnType"
		paramsStr := strings.Join(paramTypes, ", ")
		if len(returnTypes) == 0 {
			return fmt.Sprintf("func(%s)", paramsStr)
		} else if len(returnTypes) == 1 {
			return fmt.Sprintf("func(%s) %s", paramsStr, returnTypes[0])
		} else {
			returnsStr := strings.Join(returnTypes, ", ")
			return fmt.Sprintf("func(%s) (%s)", paramsStr, returnsStr)
		}
	}
	return "unknown"
}

// extractParams extracts parameter descriptors from a function's parameter list.
func extractParams(fields *ast.FieldList) []paramDescriptor {
	if fields == nil {
		return nil
	}
	var params []paramDescriptor
	for _, field := range fields.List {
		typeName := extractTypeName(field.Type)
		// Handle cases where multiple params share the same type: func(a, b string)
		if len(field.Names) > 0 {
			for _, name := range field.Names {
				params = append(params, paramDescriptor{
					Name: name.Name,
					Type: typeName,
				})
			}
		} else {
			// Unnamed parameter (rare but valid Go)
			params = append(params, paramDescriptor{
				Name: "",
				Type: typeName,
			})
		}
	}
	return params
}

// extractReturns extracts return type names from a function's return list.
func extractReturns(fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}
	var returns []string
	for _, field := range fields.List {
		typeName := extractTypeName(field.Type)
		// Multiple return values of the same type: func() (int, int)
		if len(field.Names) > 0 {
			for range field.Names {
				returns = append(returns, typeName)
			}
		} else {
			returns = append(returns, typeName)
		}
	}
	return returns
}

// inspectStructInFile is a helper that inspects a specific struct type in a Go file.
// It returns a schema with the struct's exported fields.
func inspectStructInFile(path, structName string) (componentSchema, error) {
	schema := componentSchema{
		Props:   make(map[string]propertyDescriptor),
		Methods: make(map[string]methodDescriptor),
	}
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return schema, err
	}

	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok && typeSpec.Name.Name == structName {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				found = true
				for _, field := range structType.Fields.List {
					if len(field.Names) > 0 && field.Names[0].IsExported() {
						fieldName := field.Names[0].Name
						goType := extractTypeName(field.Type)
						schema.Props[strings.ToLower(fieldName)] = propertyDescriptor{
							Name:          fieldName,
							LowercaseName: strings.ToLower(fieldName),
							GoType:        goType,
						}
					}
				}
			}
		}
		return true
	})

	if !found {
		return schema, fmt.Errorf("struct '%s' not found in file", structName)
	}

	return schema, nil
}

// inspectGoFile parses a Go file and extracts the prop schema for a given struct.
func inspectGoFile(path, structName string) (componentSchema, error) {
	schema := componentSchema{
		Props:   make(map[string]propertyDescriptor),
		State:   make(map[string]propertyDescriptor),
		Methods: make(map[string]methodDescriptor),
		Slot:    nil,
	}
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return schema, err
	}

	var slotFields []propertyDescriptor // Track all slot fields for validation

	ast.Inspect(node, func(n ast.Node) bool {
		// Inspect for struct fields (Props)
		if typeSpec, ok := n.(*ast.TypeSpec); ok && typeSpec.Name.Name == structName {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				for _, field := range structType.Fields.List {
					if len(field.Names) > 0 && field.Names[0].IsExported() {
						fieldName := field.Names[0].Name
						goType := extractTypeName(field.Type)

						// Check if field is marked as state via struct tag
						isState := false
						if field.Tag != nil {
							tag := field.Tag.Value
							// Parse struct tag - remove surrounding backticks
							if len(tag) >= 2 {
								tag = tag[1 : len(tag)-1]
							}
							// Check for nojs:"state" tag
							if strings.Contains(tag, `nojs:"state"`) {
								isState = true
							}
						}

						propDesc := propertyDescriptor{
							Name:          fieldName,
							LowercaseName: strings.ToLower(fieldName),
							GoType:        goType,
						}

						// Check if this is a content slot field ([]*vdom.VNode)
						if goType == "[]*vdom.VNode" {
							slotFields = append(slotFields, propDesc)
						} else if !isState {
							// Regular prop field - only add if not marked as state
							schema.Props[strings.ToLower(fieldName)] = propDesc
						} else {
							// State field - add to State map for template access
							schema.State[strings.ToLower(fieldName)] = propDesc
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
					methodDesc := methodDescriptor{
						Name:    funcDecl.Name.Name,
						Params:  extractParams(funcDecl.Type.Params),
						Returns: extractReturns(funcDecl.Type.Results),
					}
					schema.Methods[funcDecl.Name.Name] = methodDesc
				}
			}
		}

		return true
	})

	// Validate single slot constraint
	if len(slotFields) > 1 {
		var fieldNames []string
		for _, sf := range slotFields {
			fieldNames = append(fieldNames, sf.Name)
		}
		fmt.Fprintf(os.Stderr, "Compilation Error: could not inspect Go file %s: component '%s' has multiple content slot fields: [%s]. Only one []*vdom.VNode field is allowed per component\n",
			path, structName, strings.Join(fieldNames, ", "))
		os.Exit(1)
	}

	// Set the single slot field if found
	if len(slotFields) == 1 {
		schema.Slot = &slotFields[0]
	}

	return schema, nil
}
