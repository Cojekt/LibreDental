// @ts-ignore
import daikon from "daikon";

export function isDicomFile(nameOrType: string, buffer?: ArrayBuffer): boolean {
  const lower = nameOrType.toLowerCase();
  if (
    lower.endsWith(".dcm") ||
    lower.endsWith(".dicom") ||
    lower.includes("application/dicom") ||
    lower.includes("image/dicom") ||
    lower.includes("application/x-dicom")
  ) {
    return true;
  }
  if (buffer && buffer.byteLength >= 132) {
    const view = new DataView(buffer);
    // DICOM format has magic bytes 'DICM' starting at byte offset 128
    const magic = String.fromCharCode(
      view.getUint8(128),
      view.getUint8(129),
      view.getUint8(130),
      view.getUint8(131)
    );
    if (magic === "DICM") {
      return true;
    }
  }
  return false;
}

export function parseDicomToDataUrl(
  arrayBuffer: ArrayBuffer,
  frameIndex: number = 0
): string | null {
  try {
    const dataView = new DataView(arrayBuffer);
    const image = daikon.Series.parseImage(dataView);

    if (!image || !image.hasPixelData()) {
      console.warn("DICOM image parsed but has no pixel data.");
      return null;
    }

    const width = image.getCols();
    const height = image.getRows();
    if (!width || !height) {
      return null;
    }

    const numPixels = width * height;
    const samplesPerPixel = image.getNumberOfSamplesPerPixel() || 1;
    const rawData = image.getInterpretedData(false, false, frameIndex);
    if (!rawData || rawData.length === 0) {
      return null;
    }

    let min = image.getMin();
    let max = image.getMax();
    if (min === max || min === undefined || max === undefined) {
      min = 0;
      max = 255;
    }
    const range = max - min || 1;

    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;
    const ctx = canvas.getContext("2d");
    if (!ctx) return null;

    const imgData = ctx.createImageData(width, height);
    const data = imgData.data;

    const isRGB = samplesPerPixel >= 3;
    for (let i = 0; i < numPixels; i++) {
      const valIndex = i * samplesPerPixel;
      const idx = i * 4;
      if (isRGB) {
        const valR = rawData[valIndex];
        const valG = rawData[valIndex + 1];
        const valB = rawData[valIndex + 2];
        data[idx] = Math.max(0, Math.min(255, Math.floor(valR)));
        data[idx + 1] = Math.max(0, Math.min(255, Math.floor(valG)));
        data[idx + 2] = Math.max(0, Math.min(255, Math.floor(valB)));
      } else {
        const val = rawData[valIndex];
        const norm = Math.max(0, Math.min(255, Math.floor(((val - min) / range) * 255)));
        data[idx] = norm; // R
        data[idx + 1] = norm; // G
        data[idx + 2] = norm; // B
      }
      data[idx + 3] = 255; // A
    }

    ctx.putImageData(imgData, 0, 0);
    return canvas.toDataURL("image/png");
  } catch (err) {
    console.error("Error parsing DICOM with daikon:", err);
    return null;
  }
}
