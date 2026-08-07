/**
 * voiceAudio.ts — 浏览器麦克风采集、降采样和PCM分包
 *
 * 模块职责：
 * 1. 优先使用AudioWorklet；
 * 2. 不支持或初始化失败时回退ScriptProcessorNode；
 * 3. 将浏览器原始采样率降为16kHz；
 * 4. 输出单声道16bit小端PCM；
 * 5. 每约200ms输出一包；
 * 6. 停止时输出不足200ms的最后残余；
 * 7. 统一释放麦克风、AudioContext和音频节点。
 *
 * 本模块不建立WebSocket，也不接触JWT或业务输入框。
 */

const TARGET_SAMPLE_RATE = 16000
const CHUNK_DURATION_MS = 200

const PROCESSOR_NAME =
  'tedna-voice-capture-processor'

const WORKLET_SOURCE = `
class TEDNAVoiceCaptureProcessor extends AudioWorkletProcessor {
  process(inputs) {
    const input = inputs[0]
    const channel = input && input[0]

    if (channel && channel.length > 0) {
      this.port.postMessage(channel.slice(0))
    }

    return true
  }
}

registerProcessor(
  '${PROCESSOR_NAME}',
  TEDNAVoiceCaptureProcessor,
)
`

export interface VoicePCMRecorder {
  start: (
    stream: MediaStream,
  ) => Promise<void>
  stop: (
    flush: boolean,
  ) => number
  destroy: () => void
  chunkCount: () => number
}

export interface VoicePCMRecorderOptions {
  onPCM: (
    pcm: ArrayBuffer,
  ) => void
  onError: (
    message: string,
  ) => void
}

export function supportsVoiceInput(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof navigator !== 'undefined' &&
    typeof WebSocket !== 'undefined' &&
    typeof AudioContext !== 'undefined' &&
    Boolean(
      navigator.mediaDevices
        ?.getUserMedia,
    )
  )
}

export function stopMediaStream(
  stream: MediaStream | null,
): void {
  if (!stream) return

  for (
    const track
    of stream.getTracks()
  ) {
    track.stop()
  }
}

export function microphoneErrorMessage(
  error: unknown,
): string {
  if (
    typeof DOMException !==
      'undefined' &&
    error instanceof DOMException
  ) {
    switch (error.name) {
      case 'NotAllowedError':
      case 'SecurityError':
        return '麦克风权限被拒绝，请在浏览器地址栏允许麦克风访问'

      case 'NotFoundError':
        return '没有检测到可用麦克风'

      case 'NotReadableError':
        return '麦克风正被其他程序占用，请关闭后重试'
    }
  }

  if (
    error instanceof Error &&
    error.message.trim()
  ) {
    return error.message
  }

  return '无法启动麦克风，请检查浏览器权限和设备'
}

function float32ToPCM16(
  samples: Float32Array,
): ArrayBuffer {
  const buffer =
    new ArrayBuffer(
      samples.length * 2,
    )

  const view =
    new DataView(buffer)

  for (
    let index = 0;
    index < samples.length;
    index += 1
  ) {
    const sample = Math.max(
      -1,
      Math.min(
        1,
        samples[index] || 0,
      ),
    )

    const value =
      sample < 0
        ? sample * 0x8000
        : sample * 0x7fff

    view.setInt16(
      index * 2,
      Math.round(value),
      true,
    )
  }

  return buffer
}

function downsampleTo16K(
  input: Float32Array,
  sourceSampleRate: number,
): Float32Array {
  if (
    sourceSampleRate ===
    TARGET_SAMPLE_RATE
  ) {
    return input.slice()
  }

  const outputLength =
    Math.max(
      1,
      Math.round(
        input.length *
          TARGET_SAMPLE_RATE /
          sourceSampleRate,
      ),
    )

  const output =
    new Float32Array(
      outputLength,
    )

  const ratio =
    input.length /
    outputLength

  for (
    let outputIndex = 0;
    outputIndex < outputLength;
    outputIndex += 1
  ) {
    const start =
      Math.floor(
        outputIndex * ratio,
      )

    const end =
      Math.max(
        start + 1,
        Math.floor(
          (outputIndex + 1) *
            ratio,
        ),
      )

    let total = 0
    let count = 0

    for (
      let inputIndex = start;
      inputIndex < end &&
      inputIndex < input.length;
      inputIndex += 1
    ) {
      total +=
        input[inputIndex] || 0
      count += 1
    }

    output[outputIndex] =
      count > 0
        ? total / count
        : 0
  }

  return output
}

function takeSamples(
  queue: Float32Array[],
  requested: number,
): Float32Array {
  const result =
    new Float32Array(
      requested,
    )

  let written = 0

  while (
    written < requested &&
    queue.length > 0
  ) {
    const first = queue[0]
    const remaining =
      requested - written

    if (
      first.length <= remaining
    ) {
      result.set(
        first,
        written,
      )

      written +=
        first.length

      queue.shift()
      continue
    }

    result.set(
      first.subarray(
        0,
        remaining,
      ),
      written,
    )

    queue[0] =
      first.slice(remaining)

    written += remaining
  }

  return result
}

export function createVoicePCMRecorder(
  options: VoicePCMRecorderOptions,
): VoicePCMRecorder {
  let stream:
    MediaStream | null = null

  let audioContext:
    AudioContext | null = null

  let sourceNode:
    MediaStreamAudioSourceNode | null =
      null

  let workletNode:
    AudioWorkletNode | null =
      null

  let processorNode:
    ScriptProcessorNode | null =
      null

  let silentGain:
    GainNode | null = null

  let sourceSampleRate = 48000
  let queue: Float32Array[] = []
  let queuedSamples = 0
  let emittedChunks = 0
  let active = false
  let released = false

  const emit = (
    samples: Float32Array,
  ) => {
    if (
      samples.length === 0
    ) {
      return
    }

    const downsampled =
      downsampleTo16K(
        samples,
        sourceSampleRate,
      )

    options.onPCM(
      float32ToPCM16(
        downsampled,
      ),
    )

    emittedChunks += 1
  }

  const processQueue = () => {
    const chunkSamples =
      Math.max(
        1,
        Math.round(
          sourceSampleRate *
            CHUNK_DURATION_MS /
            1000,
        ),
      )

    while (
      active &&
      queuedSamples >=
        chunkSamples
    ) {
      const chunk =
        takeSamples(
          queue,
          chunkSamples,
        )

      queuedSamples -=
        chunkSamples

      try {
        emit(chunk)
      } catch (cause) {
        active = false

        options.onError(
          cause instanceof Error
            ? cause.message
            : '发送语音数据失败',
        )

        return
      }
    }
  }

  const acceptFrame = (
    frame: Float32Array,
  ) => {
    if (
      !active ||
      frame.length === 0
    ) {
      return
    }

    queue.push(
      frame.slice(),
    )

    queuedSamples +=
      frame.length

    processQueue()
  }

  const disconnectNodes = () => {
    active = false

    if (workletNode) {
      workletNode.port.onmessage =
        null
    }

    if (processorNode) {
      processorNode.onaudioprocess =
        null
    }

    try {
      workletNode?.disconnect()
    } catch {
      // 已断开。
    }

    try {
      processorNode?.disconnect()
    } catch {
      // 已断开。
    }

    try {
      sourceNode?.disconnect()
    } catch {
      // 已断开。
    }

    try {
      silentGain?.disconnect()
    } catch {
      // 已断开。
    }

    workletNode = null
    processorNode = null
    sourceNode = null
    silentGain = null
  }

  const releaseMedia = () => {
    if (released) return
    released = true

    stopMediaStream(stream)
    stream = null

    const context =
      audioContext

    audioContext = null

    if (
      context &&
      context.state !== 'closed'
    ) {
      void context.close()
    }
  }

  const start = async (
    nextStream: MediaStream,
  ) => {
    if (
      active ||
      audioContext
    ) {
      throw new Error(
        '录音器已经启动',
      )
    }

    queue = []
    queuedSamples = 0
    emittedChunks = 0
    released = false
    stream = nextStream

    audioContext =
      new AudioContext({
        latencyHint:
          'interactive',
      })

    sourceSampleRate =
      audioContext.sampleRate

    await audioContext.resume()

    sourceNode =
      audioContext
        .createMediaStreamSource(
          nextStream,
        )

    silentGain =
      audioContext.createGain()

    silentGain.gain.value = 0

    silentGain.connect(
      audioContext.destination,
    )

    active = true

    if (
      audioContext.audioWorklet
    ) {
      try {
        const blob =
          new Blob(
            [WORKLET_SOURCE],
            {
              type:
                'text/javascript',
            },
          )

        const moduleURL =
          URL.createObjectURL(
            blob,
          )

        try {
          await audioContext
            .audioWorklet
            .addModule(moduleURL)
        } finally {
          URL.revokeObjectURL(
            moduleURL,
          )
        }

        workletNode =
          new AudioWorkletNode(
            audioContext,
            PROCESSOR_NAME,
            {
              numberOfInputs: 1,
              numberOfOutputs: 1,
              outputChannelCount:
                [1],
            },
          )

        workletNode.port.onmessage =
          (
            event:
              MessageEvent<Float32Array>,
          ) => {
            if (
              event.data instanceof
              Float32Array
            ) {
              acceptFrame(
                event.data,
              )
            }
          }

        sourceNode.connect(
          workletNode,
        )

        workletNode.connect(
          silentGain,
        )

        return
      } catch {
        if (workletNode) {
          workletNode.port.onmessage =
            null

          try {
            workletNode.disconnect()
          } catch {
            // 初始化失败后已断开。
          }

          workletNode = null
        }

        /**
         * AudioWorklet可能被浏览器策略或CSP拦截。
         * 保留已创建的Source和Gain，继续走兼容采集路径。
         */
      }
    }

    processorNode =
      audioContext
        .createScriptProcessor(
          4096,
          1,
          1,
        )

    processorNode.onaudioprocess =
      (event) => {
        const input =
          event.inputBuffer
            .getChannelData(0)
            .slice()

        event.outputBuffer
          .getChannelData(0)
          .fill(0)

        acceptFrame(input)
      }

    sourceNode.connect(
      processorNode,
    )

    processorNode.connect(
      silentGain,
    )
  }

  const stop = (
    flush: boolean,
  ): number => {
    disconnectNodes()

    if (
      flush &&
      queuedSamples > 0
    ) {
      const remaining =
        takeSamples(
          queue,
          queuedSamples,
        )

      queuedSamples = 0

      try {
        emit(remaining)
      } catch (cause) {
        options.onError(
          cause instanceof Error
            ? cause.message
            : '发送最后一段语音失败',
        )
      }
    }

    queue = []
    queuedSamples = 0

    releaseMedia()

    return emittedChunks
  }

  const destroy = () => {
    disconnectNodes()

    queue = []
    queuedSamples = 0

    releaseMedia()
  }

  return {
    start,
    stop,
    destroy,
    chunkCount: () =>
      emittedChunks,
  }
}
