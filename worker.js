importScripts("wasm_exec.js");
importScripts("message_types.js");

function setImageSource(base64Image) {
  postMessage({ type: MESSAGE_TYPES.SET_SOURCE, src: base64Image });
}

self.onmessage = (message) => {
  switch (message.data.type) {
    case MESSAGE_TYPES.INIT:
      const go = new Go();

      WebAssembly.instantiateStreaming(
        fetch("main.wasm"),
        go.importObject
      ).then(async (result) => {
        await go.run(result.instance);
      });

      return;

    case MESSAGE_TYPES.CARVE:
      const { src, targetHeight, targetWidth } = message.data.params;
      goCarve(src, targetHeight, targetWidth);

      return;

    default:
      return;
  }
};
