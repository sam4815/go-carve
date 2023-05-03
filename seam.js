const rootEl = document.getElementById("root");
const widthRangeEl = document.getElementById("width-range");
const heightRangeEl = document.getElementById("height-range");
const inputImageEl = document.getElementById("input");
const outputImageEl = document.getElementById("output");
const overlayEl = document.getElementById("overlay");

const setImgSrcFromBuffer = (buffer) => {
  const base64String = btoa(
    new Uint8Array(buffer).reduce(
      (data, byte) => data + String.fromCharCode(byte),
      ""
    )
  );
  inputImageEl.src = `data:image/jpeg;base64,${base64String}`;
  outputImageEl.src = `data:image/jpeg;base64,${base64String}`;
  outputImageEl.style.maxHeight = "unset";
};

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
  widthRangeEl.value = 0;
  heightRangeEl.value = 0;
  widthRangeEl.max = Math.round(inputImageEl.naturalWidth) - 100;
  heightRangeEl.max = Math.round(inputImageEl.naturalHeight) - 100;
  updateOverlay();
};

const updateOverlay = () => {
  const widthMax = inputImageEl.naturalWidth;
  const widthValue = widthRangeEl.value;
  const widthPercent = ((widthValue / widthMax) * 100) / 2;

  const heightMax = inputImageEl.naturalHeight;
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
  outputImageEl.style.maxHeight = `${inputImageEl.clientHeight}px`;
  const src = inputImageEl.src.split(",")[1];
  const targetHeight = inputImageEl.naturalHeight - heightRangeEl.value;
  const targetWidth = inputImageEl.naturalWidth - widthRangeEl.value;
  worker.postMessage({
    type: MESSAGE_TYPES.CARVE,
    params: { src, targetHeight, targetWidth },
  });
};

const onClickDownload = () => {
  const link = document.createElement("a");
  link.setAttribute("href", outputImageEl.src);
  link.setAttribute("download", "resized.jpg");
  link.click();
};

const initialize = () =>
  fetch("https://picsum.photos/650/450")
    .then((r) => r.arrayBuffer())
    .then((buffer) => {
      const arrayBuffer = new Uint8Array(buffer);
      setImgSrcFromBuffer(arrayBuffer);
      rootEl.style.opacity = 1;
    });

const worker = new Worker("worker.js");

worker.postMessage({ type: MESSAGE_TYPES.INIT });

worker.onmessage = ({ data }) => {
  if (data.type == MESSAGE_TYPES.SET_SOURCE) {
    outputImageEl.src = `data:image/jpeg;base64,${data.src}`;
  }
};

initialize();
