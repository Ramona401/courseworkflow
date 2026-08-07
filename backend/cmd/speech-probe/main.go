package main

// speech-probe — TE-DNA语音输入部署后端到端冒烟探针
//
// 本命令只用于部署验证，不作为常驻服务运行。
//
// 验证范围：
//   1. 使用JWT_SECRET生成两分钟有效的临时JWT；
//   2. 从公网wss地址经过Nginx完成WebSocket升级；
//   3. 使用生产Origin通过后端严格Origin校验；
//   4. 发送固定16kHz、16bit、单声道start消息；
//   5. 等待后端完成ASR配置读取和豆包上游WebSocket握手；
//   6. 收到ready后立即发送cancel，不上传音频、不产生识别正文。
//
// 安全边界：
//   - 不打印JWT_SECRET和临时JWT；
//   - 不读取或打印火山Access Token；
//   - 不上传真实音频；
//   - 不写数据库；
//   - 临时JWT只存在于当前进程内存，最长两分钟失效。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	defaultProbeURL    = "wss://workflow.pkuailab.com/api/v1/speech/stream"
	defaultProbeOrigin = "https://workflow.pkuailab.com"
)

type probeClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	IsSuper  bool   `json:"is_super"`
	jwt.RegisteredClaims
}

type speechEvent struct {
	Event     string `json:"event"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	LogID     string `json:"log_id"`
}

func fatalf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, "❌ "+format+"\n", args...)
	os.Exit(1)
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func buildProbeToken(secret string) (string, error) {
	now := time.Now()

	claims := probeClaims{
		UserID:   "speech-probe-" + uuid.NewString(),
		Username: "speech-deploy-probe",
		Role:     "teacher",
		IsSuper:  false,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(2 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			Issuer:    "tedna",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func buildProbeURL(rawURL string, token string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	if parsed.Scheme != "wss" && parsed.Scheme != "ws" {
		return "", fmt.Errorf("探针地址必须使用ws或wss协议")
	}

	query := parsed.Query()
	query.Set("token", token)
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func readHandshakeBody(response *http.Response) string {
	if response == nil || response.Body == nil {
		return ""
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 4096))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(body))
}

func main() {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		fatalf("JWT_SECRET未加载，无法执行语音WebSocket探针")
	}

	token, err := buildProbeToken(secret)
	if err != nil {
		fatalf("生成临时探针JWT失败: %v", err)
	}

	probeURL, err := buildProbeURL(
		envOrDefault("SPEECH_PROBE_URL", defaultProbeURL),
		token,
	)
	if err != nil {
		fatalf("构造语音探针地址失败: %v", err)
	}

	origin := envOrDefault("SPEECH_PROBE_ORIGIN", defaultProbeOrigin)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	headers := http.Header{}
	headers.Set("Origin", origin)
	headers.Set("User-Agent", "TE-DNA-Speech-Deploy-Probe/1.0")

	dialer := websocket.Dialer{
		HandshakeTimeout:  15 * time.Second,
		EnableCompression: false,
	}

	conn, response, err := dialer.DialContext(ctx, probeURL, headers)
	if err != nil {
		status := ""
		if response != nil {
			status = response.Status
		}

		body := readHandshakeBody(response)
		if body != "" {
			fatalf(
				"语音WebSocket升级失败: %v; HTTP=%s; 响应=%s",
				err,
				status,
				body,
			)
		}

		fatalf("语音WebSocket升级失败: %v; HTTP=%s", err, status)
	}
	defer conn.Close()

	if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		fatalf("设置探针写超时失败: %v", err)
	}

	if err := conn.WriteJSON(map[string]interface{}{
		"action":          "start",
		"sample_rate":     16000,
		"bits_per_sample": 16,
		"channels":        1,
	}); err != nil {
		fatalf("发送语音start消息失败: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(20 * time.Second)); err != nil {
		fatalf("设置探针读超时失败: %v", err)
	}

	for {
		messageType, payload, readErr := conn.ReadMessage()
		if readErr != nil {
			fatalf("等待语音ready事件失败: %v", readErr)
		}

		if messageType != websocket.TextMessage {
			continue
		}

		var event speechEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			continue
		}

		switch event.Event {
		case "ready":
			fmt.Printf(
				"✅ 语音WebSocket端到端冒烟通过 request_id=%s log_id=%s\n",
				event.RequestID,
				event.LogID,
			)

			if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err == nil {
				_ = conn.WriteJSON(map[string]string{
					"action": "cancel",
				})
			}

			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(
					websocket.CloseNormalClosure,
					"probe complete",
				),
				time.Now().Add(2*time.Second),
			)
			return

		case "error":
			fatalf(
				"语音服务返回错误 code=%s message=%s request_id=%s log_id=%s",
				event.Code,
				event.Message,
				event.RequestID,
				event.LogID,
			)

		case "closed":
			fatalf("语音连接在ready前关闭: %s", event.Message)
		}
	}
}
