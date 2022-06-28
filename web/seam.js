const initialize = () =>
  fetch("./dali.jpeg")
    .then((r) => r.arrayBuffer())
    .then((buffer) => {
      const arrayBuffer = new Uint8Array(buffer);
      analyze(arrayBuffer);
    });

const loadAndInitWA = (waURL) => {
  const go = new Go();
  WebAssembly.instantiateStreaming(fetch(waURL), go.importObject).then(
    async (result) => {
      initialize();
      await go.run(result.instance);
    }
  );
};

loadAndInitWA("main.wasm");

const widthRangeEl = document.getElementById("width-range");
const heightRangeEl = document.getElementById("height-range");
const imageEl = document.getElementById("output");

const onBufferLoad = (buffer) => {
  const arrayBuffer = new Uint8Array(buffer.target.result);
  analyze(arrayBuffer);
};

const readFileFromEvent = (event) => {
  const file = (event.dataTransfer || event.target).files[0];
  if (!file) return;

  const reader = new FileReader();
  reader.onload = onBufferLoad;
  reader.readAsArrayBuffer(file);
};

const cancelEvent = (event) => {
  event.stopPropagation();
  event.preventDefault();
};

const setRanges = () => {
  widthRangeEl.value = 0;
  heightRangeEl.value = 0;
  widthRangeEl.max = Math.round(imageEl.naturalWidth) - 100;
  heightRangeEl.max = Math.round(imageEl.naturalHeight) - 100;
};

const updateRange = () => {
  const widthMax = imageEl.naturalWidth;
  const widthValue = widthRangeEl.value;
  const widthPercent = ((widthValue / widthMax) * 100) / 2;

  const heightMax = imageEl.naturalHeight;
  const heightValue = heightRangeEl.value;
  const heightPercent = ((heightValue / heightMax) * 100) / 2;

  setOverlay(widthPercent, heightPercent);
};

const setOverlay = (width, height) => {
  const background = `
    linear-gradient(90deg, black ${width}%, transparent ${width}%),
    linear-gradient(-90deg, black ${width}%, transparent ${width}%),
    linear-gradient(0deg, black ${height}%, transparent ${height}%),
    linear-gradient(180deg, black ${height}%, transparent ${height}%)
  `;
  document.getElementById("overlay").style.background = background;
};

const onDropFile = (event) => {
  cancelEvent(event);
  readFileFromEvent(event);
};
