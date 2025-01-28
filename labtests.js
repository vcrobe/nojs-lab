function callAddFunction() {
    const num1 = parseInt(document.getElementById('num1').value);
    const num2 = parseInt(document.getElementById('num2').value);

    console.log(add(num1, num2));
}

function calledFromGoWasm(message) {
    console.log('JavaScript function called from Go wasm: ', message);
}