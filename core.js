const go = new Go();
    WebAssembly
        .instantiateStreaming(fetch("main.wasm"), go.importObject)
        .then(result => {
            go.run(result.instance);
            console.log("WebAssembly module loaded.");
        });


function loadComponent(elementId, filePath) {
    fetch(filePath)
        .then(response => {
            if (!response.ok) {
                throw new Error('Network response was not ok ' + response.statusText);
            }
            return response.text();
        })
        .then(data => {
            document.getElementById(elementId).innerHTML = data;
        })
        .catch(error => {
            console.error('There has been a problem with your fetch operation:', error);
        });
}