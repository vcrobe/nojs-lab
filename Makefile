.PHONY: help wasm wasm-prod full full-prod clean serve

# Variables
COMPILER_PATH := github.com/vcrobe/nojs/cmd/nojs-compiler
COMPONENTS_DIR := ./app/internal/app/components
WASM_OUTPUT := ./app/wwwroot/main.wasm
MAIN_PATH := ./app/internal/app/main.go
BUILD_TAGS := -tags=dev

# Default target
.DEFAULT_GOAL := help

# Help target
help:
	@echo "🛠️  nojs Build Commands"
	@echo ""
	@echo "Development Mode (with -tags=dev):"
	@echo "  make wasm       - Build WASM only (skip templates compilation)"
	@echo "  make full       - Full build (recompile templates and WASM)"
	@echo ""
	@echo "Production Mode (without -tags=dev):"
	@echo "  make wasm-prod  - Build WASM only (skip templates compilation)"
	@echo "  make full-prod  - Full build (recompile templates and WASM)"
	@echo ""
	@echo "Utility:"
	@echo "  make clean      - Remove generated WASM binary"
	@echo ""

# Full build: compile templates + WASM (dev mode)
full: compile wasm
	@echo "✅ Full build complete!"

# Compile templates
compile:
	@echo "🔨 Compiling templates..."
	@go run $(COMPILER_PATH) -in=$(COMPONENTS_DIR)

# Build WASM only (dev mode, templates assumed up-to-date)
wasm:
	@echo "🔨 Building WASM (dev mode)..."
	@GOOS=js GOARCH=wasm go build -o $(WASM_OUTPUT) $(BUILD_TAGS) $(MAIN_PATH)

# Full build: compile templates + WASM (prod mode)
full-prod: compile wasm-prod
	@echo "✅ Full production build complete!"

# Build WASM only (prod mode, templates assumed up-to-date)
wasm-prod:
	@echo "🔨 Building WASM (production mode)..."
	@GOOS=js GOARCH=wasm go build -o $(WASM_OUTPUT) $(MAIN_PATH)

# Clean
clean:
	@echo "🧹 Cleaning..."
	@rm -f $(WASM_OUTPUT)
	@echo "✅ Clean complete!"

serve:
	@echo "🚀 Starting development server on port 9090..."
	@cd ./app/wwwroot && python3 -m http.server 9090