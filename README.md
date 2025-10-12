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
