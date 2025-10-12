# Project Compilation Instructions

To compile this project, please follow these steps:

1. Open a terminal and navigate to the root directory of the project.
2. Run the following command to build the project for WebAssembly:

   ``` bash
   $ GOOS=js GOARCH=wasm go build -o main.wasm
   ```

   ``` PowerShell
   PS> env:GOOS="js"; $env:GOARCH="wasm"; go build -o main.wasm
   ```

This command sets the target operating system to JavaScript (`GOOS=js`) and the architecture to WebAssembly (`GOARCH=wasm`). The output will be a `main.wasm` file, which can be used in web environments that support WebAssembly.

# Running the Project

To run the project after compilation, follow these steps:

1. In the root directory of the project, start a static file web server. For example, you can use Python's built-in HTTP server:

   ``` bash
   $ python3 -m http.server 9090
   ```

   ``` PowerShell
   PS> python -m http.server 9090
   ```

2. Open your web browser and navigate to `http://localhost:9090` to access the project.

This will serve the compiled `main.wasm` and any other static files in the project directory, allowing you to run and test the application in your browser.

> Note: In your browser's DevTools, enable "Disable cache" to force loading WebAssembly modules (e.g., main.wasm) on every refresh. For Chrome/Edge, open DevTools, go to the Network tab, and check "Disable cache" (applies while DevTools is open).

# Using the AOT Compiler (HTML Template to Go Component)

The framework includes an Ahead-of-Time (AOT) compiler for converting HTML templates into Go component code. This enables automatic generation of `Render()` methods from declarative templates.

### Workflow

1. **Create your template:**  
   Place your HTML template in the `compiler` directory. The source file must be named `input.gt.html`.

2. **Run the compiler:**  
   Use the following command from the project root to generate the Go component file:

   ```PowerShell
   PS> go run ./compiler -in compiler\input.gt.html -out ..\generated.go
   ```

   - `-in` specifies the input template file.
   - `-out` specifies the output Go file (e.g., `generated.go`).

3. **Integrate the generated component:**  
   The output file will contain a Go component with a `Render()` method based on your template. You can import and use this component in your application as usual.

> **Note:** The AOT compiler is under active development. Template syntax and features may change. See repo documentation for supported bindings and events.