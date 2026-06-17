// Audio capture:
// getUserMedia → HP(80Hz) → LP(6kHz) → RNNoise (WASM denoise) → SimpleGate → MediaStreamDestination → WebRTC

import { NoiseSuppressorWorklet_Name } from '@timephy/rnnoise-wasm';
import NoiseSuppressorWorkletURL from '@timephy/rnnoise-wasm/NoiseSuppressorWorklet?url';

let audioCtx: AudioContext | null = null;
let rnnoiseNode: AudioWorkletNode | null = null;
let gateNode: AudioWorkletNode | null = null;
let destNode: MediaStreamAudioDestinationNode | null = null;
let stream: MediaStream | null = null;
let processedStream: MediaStream | null = null;

// Simple energy gate — runs AFTER RNNoise (audio is already denoised).
// RNNoise suppresses background noise but doesn't mute silence → the gate
// handles the final silence cut with adaptive noise floor tracking.
const gateCode = `
class SimpleGateProcessor extends AudioWorkletProcessor {
  constructor() {
    super();
    this.envelope = 0;
    this.noiseFloor = 1e-8;
    this.holdSamples = 0;
  }

  static get parameterDescriptors() {
    return [
      { name: 'holdMs', defaultValue: 80, min: 0, max: 500 },
    ];
  }

  process(inputs, outputs, parameters) {
    const src = inputs[0]?.[0];
    const dst = outputs[0]?.[0];
    if (!src || !dst) return true;

    const N = src.length;
    const sr = sampleRate;
    const holdMs = parameters.holdMs[0];

    // Block energy
    let energy = 0;
    for (let i = 0; i < N; i++) energy += src[i] * src[i];
    energy /= N;

    // Adaptive noise floor (very slow adaptation — α=0.998)
    this.noiseFloor = 0.998 * this.noiseFloor + 0.002 * energy;

    // Open gate when energy > 5x noise floor
    if (energy > this.noiseFloor * 5) {
      this.holdSamples = (holdMs / 1000) * sr;
    }

    const target = (energy > this.noiseFloor * 5 || this.holdSamples > 0) ? 1 : 0;

    // Smooth envelope (linear ramp for simplicity)
    if (target === 1) {
      this.envelope = Math.min(1, this.envelope + 0.25);
    } else {
      this.envelope = Math.max(0, this.envelope - 0.05);
    }

    for (let i = 0; i < N; i++) {
      dst[i] = src[i] * this.envelope;
      if (this.holdSamples > 0) this.holdSamples--;
    }

    return true;
  }
}
registerProcessor('simple-gate', SimpleGateProcessor);
`;

let gateBlobURL: string | null = null;

export async function startCapture(): Promise<MediaStream> {
  if (processedStream) {
    console.log('[audio] capture already running');
    return processedStream;
  }

  console.log('[audio] starting capture (HP+LP + RNNoise + gate)...');

  // 1. Raw mic — guard against missing API (mobile HTTP, old browsers)
  if (!navigator?.mediaDevices?.getUserMedia) {
    const msg = 'Microphone not available. Mobile browsers require HTTPS.';
    console.warn('[audio]', msg);
    throw new Error(msg);
  }
  try {
    stream = await navigator.mediaDevices.getUserMedia({
      audio: {
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true,
        channelCount: 1,
        // @ts-ignore — Chrome-specific flags
        googNoiseSuppression: true,
        googAutoGainControl: true,
        googEchoCancellation: true,
        googHighpassFilter: true,
      },
    });
    console.log('[audio] getUserMedia OK, sampleRate=' +
      stream.getAudioTracks()[0]?.getSettings()?.sampleRate);
  } catch (err) {
    console.error('[audio] getUserMedia failed:', err);
    throw err;
  }

  // 2. AudioContext
  try {
    audioCtx = new AudioContext();
    if (audioCtx.state === 'suspended') await audioCtx.resume();
    console.log('[audio] AudioContext sampleRate=' + audioCtx.sampleRate);
  } catch (err) {
    console.error('[audio] AudioContext failed:', err);
    stream.getTracks().forEach((t) => t.stop());
    stream = null;
    throw err;
  }

  // 3. Load worklets (RNNoise + gate)
  try {
    await audioCtx.audioWorklet.addModule(NoiseSuppressorWorkletURL);
    console.log('[audio] RNNoise worklet loaded');
  } catch (err) {
    console.error('[audio] RNNoise worklet load failed:', err);
    audioCtx.close();
    audioCtx = null;
    stream.getTracks().forEach((t) => t.stop());
    stream = null;
    throw err;
  }

  try {
    if (!gateBlobURL) {
      gateBlobURL = URL.createObjectURL(
        new Blob([gateCode], { type: 'application/javascript' }),
      );
    }
    await audioCtx.audioWorklet.addModule(gateBlobURL);
    console.log('[audio] gate worklet loaded');
  } catch (err) {
    console.error('[audio] gate worklet load failed:', err);
    audioCtx.close();
    audioCtx = null;
    stream.getTracks().forEach((t) => t.stop());
    stream = null;
    throw err;
  }

  // 4. Build graph: source → HP → LP → RNNoise → gate → dest
  //                                           gate → zeroGain → speakers (silent)
  const source = audioCtx.createMediaStreamSource(stream);
  destNode = audioCtx.createMediaStreamDestination();

  const hpFilter = audioCtx.createBiquadFilter();
  hpFilter.type = 'highpass';
  hpFilter.frequency.value = 80;
  hpFilter.Q.value = 0.7;

  const lpFilter = audioCtx.createBiquadFilter();
  lpFilter.type = 'lowpass';
  lpFilter.frequency.value = 6000;
  lpFilter.Q.value = 0.7;

  rnnoiseNode = new AudioWorkletNode(audioCtx, NoiseSuppressorWorklet_Name);
  gateNode = new AudioWorkletNode(audioCtx, 'simple-gate', {
    parameterData: { holdMs: 80 },
  });

  const zeroGain = audioCtx.createGain();
  zeroGain.gain.value = 0;

  source.connect(hpFilter);
  hpFilter.connect(lpFilter);
  lpFilter.connect(rnnoiseNode);
  rnnoiseNode.connect(gateNode);
  gateNode.connect(destNode);
  gateNode.connect(zeroGain);
  zeroGain.connect(audioCtx.destination);

  processedStream = destNode.stream;
  console.log('[audio] HP+LP → RNNoise → gate active');
  return processedStream;
}

export function getLocalStream(): MediaStream | null {
  return processedStream;
}

export function stopCapture(): void {
  console.log('[audio] stopping...');
  processedStream = null;
  rnnoiseNode?.disconnect();
  rnnoiseNode = null;
  gateNode?.disconnect();
  gateNode = null;
  destNode?.disconnect();
  destNode = null;
  audioCtx?.close();
  audioCtx = null;
  stream?.getTracks().forEach((t) => t.stop());
  stream = null;
  console.log('[audio] stopped');
}
