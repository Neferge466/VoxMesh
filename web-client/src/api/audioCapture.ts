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
// Posts { speaking: true/false } via MessagePort when gate state changes.
const gateCode = `
class SimpleGateProcessor extends AudioWorkletProcessor {
  constructor() {
    super();
    this.envelope = 0;
    this.noiseFloor = 1e-8;
    this.holdSamples = 0;
    this.onsetFrames = 0;
    this.wasSpeaking = false;
  }

  static get parameterDescriptors() {
    return [
      { name: 'holdMs', defaultValue: 200, min: 0, max: 800 },
      { name: 'threshold', defaultValue: 15, min: 2, max: 80 },
    ];
  }

  process(inputs, outputs, parameters) {
    const src = inputs[0]?.[0];
    const dst = outputs[0]?.[0];
    if (!src || !dst) return true;

    const N = src.length;
    const sr = sampleRate;
    const holdMs = parameters.holdMs[0];
    const threshold = parameters.threshold[0];

    // Block energy
    let energy = 0;
    for (let i = 0; i < N; i++) energy += src[i] * src[i];
    energy /= N;

    // Voice-band energy emphasis (300–3400 Hz approximates via simple bandpass
    // on a parallel path — we use a crude approach: energy > noiseFloor*threshold
    // still gates on total energy, but onset detection filters short transients).
    const thresholdEnergy = this.noiseFloor * threshold;
    const above = energy > thresholdEnergy;

    // Attack onset detection: require N consecutive frames above threshold
    // before opening the gate. Keyboard clicks are single-frame transients
    // (<3ms) that get filtered out; voice onsets last 20–50ms so they pass.
    if (above) {
      this.onsetFrames++;
    } else {
      this.onsetFrames = 0;
    }

    const ATTACK_ONSET_FRAMES = 3; // ~8ms at 48kHz/128
    const gateOpen = this.onsetFrames >= ATTACK_ONSET_FRAMES;

    // Adaptive noise floor — only update during silence so it tracks
    // background noise, not speech energy. Freeze during speech.
    if (!above) {
      this.noiseFloor = 0.998 * this.noiseFloor + 0.002 * energy;
    }

    if (gateOpen) {
      this.holdSamples = (holdMs / 1000) * sr;
    }

    // Soft-knee expander: below threshold, attenuate proportionally
    // instead of hard mute. Avoids jarring cuts and lets borderline
    // signals through at reduced level.
    let gateTarget;
    if (gateOpen || this.holdSamples > 0) {
      gateTarget = 1;
    } else {
      // Ratio: below threshold, gain = (energy/thresholdEnergy)^2
      // Square-law expander approximates 2:1 expansion in linear domain.
      const ratio = energy / Math.max(thresholdEnergy, 1e-12);
      gateTarget = Math.max(0.05, ratio * ratio);
    }

    // Smooth envelope
    if (gateTarget > this.envelope) {
      this.envelope = Math.min(gateTarget, this.envelope + 0.25);
    } else {
      this.envelope = Math.max(gateTarget, this.envelope - 0.04);
    }

    // Detect speaking state transitions
    const isSpeaking = this.envelope > 0.3;
    if (isSpeaking !== this.wasSpeaking) {
      this.wasSpeaking = isSpeaking;
      this.port.postMessage({ speaking: isSpeaking });
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
  lpFilter.frequency.value = 4000;
  lpFilter.Q.value = 0.7;

  rnnoiseNode = new AudioWorkletNode(audioCtx, NoiseSuppressorWorklet_Name);
  gateNode = new AudioWorkletNode(audioCtx, 'simple-gate', {
    parameterData: { holdMs: 200, threshold: 15 },
  });

  gateNode.port.onmessage = (e: MessageEvent) => {
    if (speakingCallback && e.data && typeof e.data.speaking === 'boolean') {
      speakingCallback(e.data.speaking);
    }
  };

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

let speakingCallback: ((speaking: boolean) => void) | null = null;

export function onSpeakingChange(cb: ((speaking: boolean) => void) | null): void {
  speakingCallback = cb;
}

export function setVADParams(threshold: number, holdMs?: number): void {
  if (!gateNode) return;
  const thresh = gateNode.parameters.get('threshold');
  if (thresh) thresh.value = threshold;
  if (holdMs !== undefined) {
    const hold = gateNode.parameters.get('holdMs');
    if (hold) hold.value = holdMs;
  }
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
