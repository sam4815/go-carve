const widthRangeEl = document.getElementById("width-range");
const heightRangeEl = document.getElementById("height-range");
const imageEl = document.getElementById("output");
const overlayEl = document.getElementById("overlay");

const setImgSrcFromBuffer = (buffer) => {
  const base64String = btoa(
    new Uint8Array(buffer)
      .reduce((data, byte) => data + String.fromCharCode(byte), '')
  );
  imageEl.src = `data:image/jpeg;base64,${base64String}`;
}

const onBufferLoad = (buffer) => {
  const arrayBuffer = new Uint8Array(buffer.target.result);
  setImgSrcFromBuffer(arrayBuffer);
};

const readFile = (event) => {
  const file = event.target.files[0];
  if (!file) return;

  const reader = new FileReader();
  reader.onload = onBufferLoad;
  reader.readAsArrayBuffer(file);
};

const setRanges = () => {
  widthRangeEl.value = 50;
  heightRangeEl.value = 50;
  widthRangeEl.max = Math.round(imageEl.naturalWidth) - 100;
  heightRangeEl.max = Math.round(imageEl.naturalHeight) - 100;
  updateOverlay();
};

const updateOverlay = () => {
  const widthMax = imageEl.naturalWidth;
  const widthValue = widthRangeEl.value;
  const widthPercent = ((widthValue / widthMax) * 100) / 2;

  const heightMax = imageEl.naturalHeight;
  const heightValue = heightRangeEl.value;
  const heightPercent = ((heightValue / heightMax) * 100) / 2;

  const background = `
    linear-gradient(90deg, black ${widthPercent}%, transparent ${widthPercent}%),
    linear-gradient(-90deg, black ${widthPercent}%, transparent ${widthPercent}%),
    linear-gradient(0deg, black ${heightPercent}%, transparent ${heightPercent}%),
    linear-gradient(180deg, black ${heightPercent}%, transparent ${heightPercent}%)
  `;
  overlayEl.style.background = background;
};

const onClickCarve = () => {
  const src = imageEl.src.split(',')[1];
  const output = goCarve(src);
  imageEl.src = `data:image/jpeg;base64,${output}`;
};

const onClickDownload = () => {
  const link = document.createElement("a");
  link.setAttribute("href", imageEl.src);
  link.setAttribute("download", "resized.jpg");
  link.click();
};

const initialize = () =>
  fetch("./dali.jpeg")
    .then((r) => r.arrayBuffer())
    .then((buffer) => {
      const arrayBuffer = new Uint8Array(buffer);
      setImgSrcFromBuffer(arrayBuffer);
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
