package ai

// client_tts_v3.go — 火山豆包语音合成2.0 v3接口实现（S-V1.5新增）
//
// 接口契约（经官方文档与实测资料双重确认，2026-06）：
//   POST {base_url}/api/v3/tts/unidirectional
//   请求头: Content-Type: application/json
//          X-Api-App-Id:      豆包语音应用APP ID
//          X-Api-Access-Key:  应用Access Token
//          X-Api-Resource-Id: 按音色推导（见ResolveTTSResourceID）
//   请求体: {"user":{"uid":"tedna"},
//           "req_params":{"text":"...","speaker":"音色码",
//                         "audio_params":{"format":"mp3","sample_rate":24000,"speech_rate":可选}}}
//
// 响应契约（NDJSON，换行分隔的JSON，不是标准JSON——直接整体解析会失败）：
//   {"code":0,"data":"<base64音频片段1>"}
//   {"code":0,"data":"<base64音频片段2>"}
//   {"code":20000000}            ← 流结束哨兵
//   解析规则：逐行JSON解析；code=0且data非空→base64解码追加；
//            code=20000000→正常结束；其他code→按错误处理（55000000=resource id不匹配）。
//
// 语速映射：v3的audio_params.speech_rate为整数百分比偏移，0=正常，100=2倍速，-50=0.5倍速。
//   前端四档 0.75/1.0/1.25/1.5 → -25/0/25/50，公式 rate=round((speed-1)×100)，钳制[-50,100]。

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ==================== v3 请求/响应结构 ====================

// ttsV3AudioParams v3 音频输出参数
type ttsV3AudioParams struct {
	Format     string `json:"format"`                // 输出格式：mp3
	SampleRate int    `json:"sample_rate"`           // 采样率：24000
	SpeechRate int    `json:"speech_rate,omitempty"` // 语速偏移百分比：-50~100，0省略=正常
}

// ttsV3ReqParams v3 合成参数
type ttsV3ReqParams struct {
	Text        string           `json:"text"`    // 合成文本
	Speaker     string           `json:"speaker"` // 音色码
	AudioParams ttsV3AudioParams `json:"audio_params"`
}

// ttsV3Request v3 请求体
type ttsV3Request struct {
	User struct {
		UID string `json:"uid"` // 调用方标识（计费归属/日志）
	} `json:"user"`
	ReqParams ttsV3ReqParams `json:"req_params"`
}

// ttsV3Line NDJSON响应的单行结构
type ttsV3Line struct {
	Code    int    `json:"code"`    // 0=音频块 20000000=结束 其他=错误
	Message string `json:"message"` // 错误描述（出错时）
	Data    string `json:"data"`    // base64音频片段（code=0时）
}

// ttsV3DoneCode 流结束哨兵码
const ttsV3DoneCode = 20000000

// ==================== 辅助函数 ====================

// ResolveTTSResourceID 按音色码推导 X-Api-Resource-Id
// 路由表：S_开头=声音复刻(seed-icl-2.0)；含_uranus_或saturn_开头=官方2.0(seed-tts-2.0)；
//
//	其余按官方1.0(seed-tts-1.0)。传错会报55000000: resource ID is mismatched。
func ResolveTTSResourceID(voice string) string {
	if strings.HasPrefix(voice, "S_") {
		return "seed-icl-2.0"
	}
	if strings.Contains(voice, "_uranus_") || strings.HasPrefix(voice, "saturn_") {
		return "seed-tts-2.0"
	}
	return "seed-tts-1.0"
}

// ttsSpeedToRate 把倍速(0.5~2.0)映射为v3的speech_rate整数(-50~100)
func ttsSpeedToRate(speed float64) int {
	rate := int(math.Round((speed - 1.0) * 100))
	if rate < -50 {
		rate = -50
	}
	if rate > 100 {
		rate = 100
	}
	return rate
}

// ==================== v3 合成实现 ====================

// synthesizeSpeechV3 火山豆包语音v3合成实现
// 入参出参与统一入口 SynthesizeSpeech 完全一致，由路由层分流调入。
func synthesizeSpeechV3(ctx context.Context, cfg *TTSConfig, text string, voice string, speed float64, outputDir string, outputName string, traceCtx *TraceContext) (*TTSResult, error) {
	// 1. v3专属配置校验
	if cfg.AppID == "" || cfg.AccessToken == "" {
		return nil, fmt.Errorf("火山TTS未配置APP ID或Access Token（请在AI管理中心或 /api/v1/admin/tts-config 配置）")
	}
	baseURL := cfg.APIBaseURL
	if baseURL == "" {
		baseURL = ttsV3DefaultBaseURL
	}
	apiURL := strings.TrimRight(baseURL, "/") + "/api/v3/tts/unidirectional"
	resourceID := ResolveTTSResourceID(voice)

	// 2. 构建请求体
	reqBody := ttsV3Request{}
	reqBody.User.UID = "tedna"
	reqBody.ReqParams = ttsV3ReqParams{
		Text:    text,
		Speaker: voice,
		AudioParams: ttsV3AudioParams{
			Format:     "mp3",
			SampleRate: 24000,
			SpeechRate: ttsSpeedToRate(speed), // 0时omitempty自动省略=正常语速
		},
	}
	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化TTS v3请求失败: %w", err)
	}

	ttsLog.Info("调用火山TTS v3语音合成",
		"url", apiURL,
		"resource_id", resourceID,
		"voice", voice,
		"text_len", len([]rune(text)),
		"speed", speed,
	)

	// 3. 创建HTTP请求（v3专属鉴权头）
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("创建TTS v3 HTTP请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-App-Id", cfg.AppID)
	httpReq.Header.Set("X-Api-Access-Key", cfg.AccessToken)
	httpReq.Header.Set("X-Api-Resource-Id", resourceID)

	// 4. 发送请求（流式返回，超时给足90秒）
	client := &http.Client{Timeout: 90 * time.Second}
	startTime := time.Now()
	httpResp, err :=
		client.Do(
			httpReq,
		)

	if err != nil {
		latencyMs :=
			time.Since(
				startTime,
			).Milliseconds()

		ttsLog.Error(
			"TTS v3 HTTP请求结果不确定",
			"error",
			err,
			"latency_ms",
			latencyMs,
		)

		return nil,
			&TTSSynthesisError{
				Cause: fmt.Errorf(
					"TTS请求网络结果不确定: %w",
					err,
				),
				Uncertain: true,
			}
	}

	defer httpResp.Body.Close()
	// 5. HTTP层错误（鉴权失败等会直接非200）
	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		latencyMs := time.Since(startTime).Milliseconds()
		ttsLog.Error("TTS v3 API返回HTTP错误",
			"status", httpResp.StatusCode,
			"body", truncateStr(string(respBody), 500),
			"latency_ms", latencyMs,
		)
		return nil, fmt.Errorf("TTS API返回错误(HTTP %d): %s", httpResp.StatusCode, truncateStr(string(respBody), 200))
	}

	// 6. NDJSON逐行解析，拼接base64音频块
	var audioBuf bytes.Buffer
	scanner := bufio.NewScanner(httpResp.Body)
	// base64音频块单行可达数百KB，放大扫描缓冲（初始1MB，上限16MB）
	scanBuf := make([]byte, 0, 1024*1024)
	scanner.Buffer(scanBuf, 16*1024*1024)

	chunkCount := 0
	doneSeen := false
	for scanner.Scan() {
		line :=
			strings.TrimSpace(
				scanner.Text(),
			)

		if line == "" {
			continue
		}

		line =
			strings.TrimSpace(
				strings.TrimPrefix(
					line,
					"data:",
				),
			)

		if line == "" {
			continue
		}

		var item ttsV3Line

		if err :=
			json.Unmarshal(
				[]byte(line),
				&item,
			); err != nil {
			return nil,
				&TTSSynthesisError{
					Cause: fmt.Errorf(
						"解析TTS流式响应失败: %w",
						err,
					),
					Uncertain: true,
				}
		}

		switch {
		case item.Code ==
			ttsV3DoneCode:
			doneSeen = true

		case item.Code == 0 &&
			item.Data != "":
			chunk, decodeErr :=
				base64.StdEncoding.
					DecodeString(
						item.Data,
					)

			if decodeErr != nil {
				return nil,
					&TTSSynthesisError{
						Cause: fmt.Errorf(
							"解码TTS音频块失败: %w",
							decodeErr,
						),
						Uncertain: true,
					}
			}

			audioBuf.Write(chunk)
			chunkCount++

		case item.Code == 0:
			// 心跳或空包。

		default:
			latencyMs :=
				time.Since(
					startTime,
				).Milliseconds()

			ttsLog.Error(
				"TTS v3返回明确业务错误",
				"code",
				item.Code,
				"message",
				item.Message,
				"latency_ms",
				latencyMs,
			)

			return nil,
				fmt.Errorf(
					"TTS合成失败(code %d): %s",
					item.Code,
					item.Message,
				)
		}

		if doneSeen {
			break
		}
	}
	if scanErr :=
		scanner.Err(); scanErr != nil {
		return nil,
			&TTSSynthesisError{
				Cause: fmt.Errorf(
					"读取TTS流式响应失败: %w",
					scanErr,
				),
				Uncertain: true,
			}
	}

	if !doneSeen {
		return nil,
			&TTSSynthesisError{
				Cause: fmt.Errorf(
					"TTS流式响应未收到结束标识",
				),
				Uncertain: true,
			}
	}

	latencyMs :=
		time.Since(
			startTime,
		).Milliseconds()

	if audioBuf.Len() < 100 {
		return nil,
			&TTSSynthesisError{
				Cause: fmt.Errorf(
					"TTS供应商已完成但未返回有效音频（%d块，%d字节）",
					chunkCount,
					audioBuf.Len(),
				),
				ProviderSucceeded: true,
			}
	}
	// 7. 写盘。此时已经收到供应商成功结束标识，
	// 后续本地失败不能释放积分预留。
	if err :=
		os.MkdirAll(
			outputDir,
			0755,
		); err != nil {
		return nil,
			&TTSSynthesisError{
				Cause: fmt.Errorf(
					"创建输出目录失败: %w",
					err,
				),
				ProviderSucceeded: true,
			}
	}

	outputPath :=
		filepath.Join(
			outputDir,
			outputName+".mp3",
		)

	if err :=
		os.WriteFile(
			outputPath,
			audioBuf.Bytes(),
			0644,
		); err != nil {
		return nil,
			&TTSSynthesisError{
				Cause: fmt.Errorf(
					"写入音频文件失败: %w",
					err,
				),
				ProviderSucceeded: true,
			}
	}

	written :=
		int64(
			audioBuf.Len(),
		)
	// 8. ffprobe取时长
	duration := getAudioDuration(outputPath)

	ttsLog.Info("TTS v3合成成功",
		"resource_id", resourceID,
		"voice", voice,
		"chunks", chunkCount,
		"file_size", written,
		"duration", duration,
		"latency_ms", latencyMs,
	)

	// 9. 写入追踪记录（口径与旧路径一致：按字符数粗估tokens）
	if traceCtx != nil {
		go func() {
			estimatedTokens := len([]rune(text))
			emitTrace(
				traceCtx, resourceID,
				estimatedTokens, 0, estimatedTokens,
				latencyMs, "success", "",
				int(written),
				false, false, "",
			)
		}()
	}

	return &TTSResult{
		AudioFilePath: outputPath,
		AudioURL:      "", // 由调用方拼接公网URL
		Duration:      duration,
		ModelUsed:     resourceID,
		FileSize:      written,
	}, nil
}
