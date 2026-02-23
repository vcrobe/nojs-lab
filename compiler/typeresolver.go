package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// resolveNestedFieldType resolves the type of a nested field (e.g., "Ctx.Title" -> "string")
// Returns the Go type string, or empty string if field not found.
// componentDir is the directory containing the component's Go files.
func resolveNestedFieldType(fieldPath string, comp componentInfo, componentDir string) (string, error) {
	parts := strings.Split(fieldPath, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("not a nested field path: %s", fieldPath)
	}

	// Start with the root field
	rootField := strings.ToLower(parts[0])
	var currentType string

	// Get the root field type from the component
	if desc, exists := comp.Schema.Props[rootField]; exists {
		currentType = desc.GoType
	} else if desc, exists := comp.Schema.State[rootField]; exists {
		currentType = desc.GoType
	} else {
		return "", fmt.Errorf("root field '%s' not found on component '%s'", parts[0], comp.PascalName)
	}

	// Traverse the nested fields
	for i := 1; i < len(parts); i++ {
		fieldName := parts[i]

		// Remove pointer dereference marker if present
		if strings.HasPrefix(currentType, "*") {
			currentType = currentType[1:]
		}

		// Remove slice marker if present
		if strings.HasPrefix(currentType, "[]") {
			currentType = currentType[2:]
		}

		// If it's a simple type (like string, int), we can't access fields
		if isBuiltinType(currentType) {
			return "", fmt.Errorf("cannot access field '%s' on built-in type '%s'", fieldName, currentType)
		}

		// Extract the struct name and potential package prefix
		structName := currentType
		var packagePath string

		if strings.Contains(currentType, ".") {
			parts := strings.Split(currentType, ".")
			packageAlias := parts[0]
			structName = parts[1]

			// Try to resolve the package alias to a full path by inspecting imports in the component file
			packagePath, _ = resolvePackageFromAlias(packageAlias, componentDir)
		}

		// Find the struct definition to get the field type
		fieldType, err := findStructFieldType(componentDir, structName, fieldName, packagePath)
		if err != nil {
			return "", fmt.Errorf("cannot resolve field '%s' on type '%s': %v", fieldName, currentType, err)
		}

		currentType = fieldType
	}

	return currentType, nil
}

// resolvePackageFromAlias looks for import statements in Go files to resolve package aliases.
// Returns the full import path for the package, or empty string if not found.
func resolvePackageFromAlias(alias string, componentDir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(componentDir, "*.go"))
	if err != nil {
		return "", err
	}

	for _, filePath := range matches {
		if strings.Contains(filePath, ".generated.") {
			continue
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			continue
		}

		for _, imp := range node.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")

			// Check if this import has an alias
			if imp.Name != nil {
				if imp.Name.Name == alias {
					return importPath, nil
				}
			} else {
				// No alias, check if the last part of the import path matches the alias
				lastPart := importPath
				if idx := strings.LastIndex(importPath, "/"); idx >= 0 {
					lastPart = importPath[idx+1:]
				}
				if lastPart == alias {
					return importPath, nil
				}
			}
		}
	}

	return "", fmt.Errorf("package alias '%s' not found in imports", alias)
}

// isBuiltinType checks if a type is a Go built-in type.
func isBuiltinType(t string) bool {
	builtins := map[string]bool{
		"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true, "bool": true, "byte": true, "rune": true,
	}
	return builtins[t]
}

// findStructFieldType searches for a struct definition and returns the type of a field.
// Searches in the component directory and optionally in the specified package.
func findStructFieldType(componentDir, structName, fieldName string, packagePath string) (string, error) {
	// First try to find it in the component directory
	if result, err := findStructFieldTypeInDir(componentDir, structName, fieldName); err == nil {
		return result, nil
	}

	// If not found and packagePath is provided, try to find it in that package
	if packagePath != "" {
		// Try searching in the package directory
		if pkgDir := findPackageDir(packagePath); pkgDir != "" {
			if result, err := findStructFieldTypeInDir(pkgDir, structName, fieldName); err == nil {
				return result, nil
			}
		}
	}

	return "", fmt.Errorf("struct '%s' with field '%s' not found in %s", structName, fieldName, componentDir)
}

// findStructFieldTypeInDir searches for a struct field in a specific directory.
func findStructFieldTypeInDir(dir, structName, fieldName string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return "", err
	}

	for _, filePath := range matches {
		// Skip generated files
		if strings.Contains(filePath, ".generated.") {
			continue
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			continue
		}

		var foundType string
		ast.Inspect(node, func(n ast.Node) bool {
			if typeSpec, ok := n.(*ast.TypeSpec); ok && typeSpec.Name.Name == structName {
				if structType, ok := typeSpec.Type.(*ast.StructType); ok {
					for _, field := range structType.Fields.List {
						if len(field.Names) > 0 && field.Names[0].Name == fieldName && field.Names[0].IsExported() {
							foundType = extractTypeName(field.Type)
							return false
						}
					}
				}
			}
			return true
		})

		if foundType != "" {
			return foundType, nil
		}
	}

	return "", fmt.Errorf("field '%s' not found in %s", fieldName, dir)
}

// findPackageDir tries to locate the directory for a given import path.
func findPackageDir(importPath string) string {
	// Use the Go packages tool to find the package
	cfg := &packages.Config{
		Mode: packages.NeedFiles,
	}
	pkgs, err := packages.Load(cfg, importPath)
	if err == nil && len(pkgs) > 0 && len(pkgs[0].GoFiles) > 0 {
		return filepath.Dir(pkgs[0].GoFiles[0])
	}
	return ""
}

// getAvailableNestedFields returns a list of available exported fields on a nested type.
// For example, if fieldPath is "Ctx.Title1", it returns available fields on the Ctx type.
func getAvailableNestedFields(fieldPath string, comp componentInfo, componentDir string) []string {
	parts := strings.Split(fieldPath, ".")
	if len(parts) < 1 {
		return nil
	}

	rootField := strings.ToLower(parts[0])
	var fieldType string

	// Get the root field type from the component
	if desc, exists := comp.Schema.Props[rootField]; exists {
		fieldType = desc.GoType
	} else if desc, exists := comp.Schema.State[rootField]; exists {
		fieldType = desc.GoType
	} else {
		return nil
	}

	// Remove pointer and slice markers
	if strings.HasPrefix(fieldType, "*") {
		fieldType = fieldType[1:]
	}
	if strings.HasPrefix(fieldType, "[]") {
		fieldType = fieldType[2:]
	}

	// Extract struct name from qualified types (e.g., "context.MainLayoutCtx" -> "MainLayoutCtx")
	structName := fieldType
	var packagePath string
	if strings.Contains(fieldType, ".") {
		typeParts := strings.Split(fieldType, ".")
		packageAlias := typeParts[0]
		structName = typeParts[1]
		packagePath, _ = resolvePackageFromAlias(packageAlias, componentDir)
	}

	// Get available fields from the nested type
	var fields []string

	// First try to find it in the component directory
	if availableFields, err := getStructFields(componentDir, structName); err == nil {
		fields = availableFields
	} else if packagePath != "" {
		// If not found and packagePath is provided, try to find it in that package
		if pkgDir := findPackageDir(packagePath); pkgDir != "" {
			if availableFields, err := getStructFields(pkgDir, structName); err == nil {
				fields = availableFields
			}
		}
	}

	return fields
}

// getStructFields returns a list of exported fields on a struct type.
func getStructFields(dir, structName string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}

	for _, filePath := range matches {
		// Skip generated files
		if strings.Contains(filePath, ".generated.") {
			continue
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			continue
		}

		var foundFields []string
		ast.Inspect(node, func(n ast.Node) bool {
			if typeSpec, ok := n.(*ast.TypeSpec); ok && typeSpec.Name.Name == structName {
				if structType, ok := typeSpec.Type.(*ast.StructType); ok {
					for _, field := range structType.Fields.List {
						if len(field.Names) > 0 && field.Names[0].IsExported() {
							foundFields = append(foundFields, field.Names[0].Name)
						}
					}
					return false
				}
			}
			return true
		})

		if len(foundFields) > 0 {
			return foundFields, nil
		}
	}

	return nil, fmt.Errorf("struct '%s' not found in %s", structName, dir)
}
