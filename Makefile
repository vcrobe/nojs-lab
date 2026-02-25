.PHONY: help wasm wasm-prod full full-prod clean serve docs-install docs-build docs-serve

# Variables
COMPILER_PATH := github.com/ForgeLogic/nojs-compiler/cmd/nojsc
COMPONENTS_DIR := ./app/internal/app/components
WASM_OUTPUT := ./app/wwwroot/main.wasm
MAIN_PATH := ./app/internal/app
BUILD_TAGS := -tags=dev

# Default serve command (override in Makefile.local)
SERVE_CMD := python3 -m http.server 9090
SERVE_DIR := ./app/wwwroot
DOCS_VENV := ./.venv-docs
DOCS_MKDOCS := $(DOCS_VENV)/bin/mkdocs

# Load local developer overrides if present (gitignored)
-include Makefile.local

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
	@echo "🚀 Starting development server..."
	@cd $(SERVE_DIR) && $(SERVE_CMD)

docs-install:
	@echo "📚 Installing documentation dependencies..."
	@python3 -m venv $(DOCS_VENV)
	@$(DOCS_VENV)/bin/pip install -r requirements-docs.txt

docs-build:
	@echo "📚 Building documentation site..."
	@$(DOCS_MKDOCS) build --strict

docs-serve:
	@echo "📚 Serving documentation site locally..."
	@$(DOCS_MKDOCS) serve