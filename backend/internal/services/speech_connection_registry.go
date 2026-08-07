package services

// speech_connection_registry.go — 全平台语音输入WebSocket连接治理
//
// 语音输入会同时占用：
//   1. 浏览器到TE-DNA的WebSocket连接；
//   2. TE-DNA到豆包ASR的上游WebSocket连接；
//   3. 火山小时版语音识别额度。
//
// 因此不能把它当成普通短HTTP请求处理。本文件提供进程级统一注册表：
//   - 同一用户默认只允许1条活动语音连接；
//   - 全进程默认最多64条活动连接；
//   - 部署排空时拒绝新连接并主动关闭全部存量连接；
//   - Acquire、设置关闭函数、Release与BeginDraining均并发安全；
//   - 关闭函数始终在锁外执行，避免网络关闭动作阻塞注册表。
//
// 本注册表只治理连接生命周期，不保存音频、识别文字或业务输入框内容。

import (
	"errors"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// ==================== 公共错误 ====================

var (
	// ErrSpeechConnectionUserRequired 表示调用方没有可信用户ID。
	ErrSpeechConnectionUserRequired = errors.New("语音连接缺少可信用户ID")

	// ErrSpeechConnectionUserLimit 表示同一用户已有活动语音连接。
	ErrSpeechConnectionUserLimit = errors.New("当前用户的语音连接数已达上限")

	// ErrSpeechConnectionGlobalLimit 表示全进程活动连接已达上限。
	ErrSpeechConnectionGlobalLimit = errors.New("语音连接总数已达上限")

	// ErrSpeechConnectionsDraining 表示进程正在部署排空。
	ErrSpeechConnectionsDraining = errors.New("语音连接服务正在排空")
)

// ==================== 默认全局注册表 ====================

const (
	// speechDefaultMaxTotalConnections 是单进程最大活动语音连接数。
	speechDefaultMaxTotalConnections = 64

	// speechDefaultMaxConnectionsPerUser 是单用户最大活动语音连接数。
	speechDefaultMaxConnectionsPerUser = 1
)

// GlobalSpeechConnections 是生产环境统一语音连接注册表。
var GlobalSpeechConnections = NewSpeechConnectionRegistry(
	speechDefaultMaxTotalConnections,
	speechDefaultMaxConnectionsPerUser,
)

// BeginGlobalSpeechDraining 进入全局语音连接排空并关闭存量连接。
func BeginGlobalSpeechDraining() int {
	return GlobalSpeechConnections.BeginDraining()
}

// ==================== 注册表模型 ====================

// speechConnectionRecord 保存一条活动连接的最小治理信息。
type speechConnectionRecord struct {
	userID  string
	closeFn func()
}

// SpeechConnectionRegistry 管理进程内全部语音连接。
type SpeechConnectionRegistry struct {
	mu sync.Mutex

	draining bool

	maxTotal   int
	maxPerUser int

	active  map[string]*speechConnectionRecord
	perUser map[string]int
}

// SpeechConnectionLease 表示调用方成功占用的一个连接名额。
//
// Release是幂等的，Handler可以在多个退出路径中安全重复调用。
type SpeechConnectionLease struct {
	registry     *SpeechConnectionRegistry
	connectionID string
	userID       string
	releaseOnce  sync.Once
}

// NewSpeechConnectionRegistry 创建连接注册表。
//
// 非法上限按最小安全值1收敛，避免0值导致全部请求永久被拒。
func NewSpeechConnectionRegistry(
	maxTotal int,
	maxPerUser int,
) *SpeechConnectionRegistry {
	if maxTotal < 1 {
		maxTotal = 1
	}
	if maxPerUser < 1 {
		maxPerUser = 1
	}
	if maxPerUser > maxTotal {
		maxPerUser = maxTotal
	}

	return &SpeechConnectionRegistry{
		maxTotal:   maxTotal,
		maxPerUser: maxPerUser,
		active:     make(map[string]*speechConnectionRecord),
		perUser:    make(map[string]int),
	}
}

// Acquire 为可信用户预留一条语音连接名额。
//
// 该方法必须在WebSocket升级前调用，使连接上限错误可以继续使用普通HTTP响应。
func (registry *SpeechConnectionRegistry) Acquire(
	userID string,
) (*SpeechConnectionLease, error) {
	if registry == nil {
		return nil, ErrSpeechConnectionGlobalLimit
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrSpeechConnectionUserRequired
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	if registry.draining {
		return nil, ErrSpeechConnectionsDraining
	}

	if len(registry.active) >= registry.maxTotal {
		return nil, ErrSpeechConnectionGlobalLimit
	}

	if registry.perUser[userID] >= registry.maxPerUser {
		return nil, ErrSpeechConnectionUserLimit
	}

	connectionID := uuid.NewString()

	registry.active[connectionID] = &speechConnectionRecord{
		userID: userID,
	}
	registry.perUser[userID]++

	return &SpeechConnectionLease{
		registry:     registry,
		connectionID: connectionID,
		userID:       userID,
	}, nil
}

// SetCloser 登记部署排空时应执行的连接关闭函数。
//
// 如果注册表已经进入排空，或连接在登记前已经释放，closeFn会在锁外立即执行。
func (lease *SpeechConnectionLease) SetCloser(
	closeFn func(),
) {
	if lease == nil || lease.registry == nil || closeFn == nil {
		return
	}

	registry := lease.registry
	closeImmediately := false

	registry.mu.Lock()

	record, exists := registry.active[lease.connectionID]
	switch {
	case !exists:
		closeImmediately = true

	case registry.draining:
		closeImmediately = true

	default:
		record.closeFn = closeFn
	}

	registry.mu.Unlock()

	if closeImmediately {
		closeFn()
	}
}

// Release 释放连接名额。
func (lease *SpeechConnectionLease) Release() {
	if lease == nil || lease.registry == nil {
		return
	}

	lease.releaseOnce.Do(func() {
		registry := lease.registry

		registry.mu.Lock()
		defer registry.mu.Unlock()

		record, exists := registry.active[lease.connectionID]
		if !exists {
			return
		}

		delete(registry.active, lease.connectionID)

		userID := record.userID
		if userID == "" {
			userID = lease.userID
		}

		if registry.perUser[userID] <= 1 {
			delete(registry.perUser, userID)
		} else {
			registry.perUser[userID]--
		}
	})
}

// BeginDraining 拒绝新连接并主动关闭全部存量连接。
//
// 返回排空开始时的活动连接数。
func (registry *SpeechConnectionRegistry) BeginDraining() int {
	if registry == nil {
		return 0
	}

	registry.mu.Lock()

	registry.draining = true
	activeCount := len(registry.active)

	closers := make([]func(), 0, activeCount)
	for _, record := range registry.active {
		if record.closeFn != nil {
			closers = append(closers, record.closeFn)
		}
	}

	registry.mu.Unlock()

	for _, closeFn := range closers {
		closeFn()
	}

	return activeCount
}

// ActiveCount 返回当前活动连接数，供健康日志和测试使用。
func (registry *SpeechConnectionRegistry) ActiveCount() int {
	if registry == nil {
		return 0
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	return len(registry.active)
}

// IsDraining 返回注册表是否已经进入排空状态。
func (registry *SpeechConnectionRegistry) IsDraining() bool {
	if registry == nil {
		return true
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	return registry.draining
}
