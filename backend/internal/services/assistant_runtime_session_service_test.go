package services

// assistant_runtime_session_service_test.go
//
// 本文件只验证教学智能体会话类型功能总闸门。
// 测试不连接数据库、不调用AI、不执行积分结算。

import (
	"context"
	"errors"
	"testing"
	"time"

	"tedna/internal/models"
)

const (
	assistantRuntimeTestSigningSecret = "assistant-runtime-test-signing-secret-20260803-abcdefghijklmnopqrstuvwxyz"

	assistantRuntimeTestPrivacySalt = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

func newAssistantRuntimeGateTestService(
	t *testing.T,
) *AssistantRuntimeSessionService {
	t.Helper()

	service :=
		NewAssistantRuntimeSessionService(
			assistantRuntimeTestSigningSecret,
			assistantRuntimeTestPrivacySalt,
			10*time.Minute,
		)

	if service == nil ||
		!service.configured() {
		t.Fatal("运行会话测试服务配置失败")
	}

	return service
}

func TestAssistantRuntimeSessionKindGateFailsClosed(
	t *testing.T,
) {
	service :=
		newAssistantRuntimeGateTestService(t)

	// 构造后未显式注入公开开关，external必须默认拒绝。
	err :=
		service.validateSessionKindEnabled(
			models.AssistantRuntimeSessionKindExternal,
		)
	if !errors.Is(
		err,
		ErrAssistantRuntimeDeploymentUnavailable,
	) {
		t.Fatalf(
			"external默认未被拒绝: error=%v",
			err,
		)
	}

	// teacher_preview必须独立于公开运行开关。
	err =
		service.validateSessionKindEnabled(
			models.AssistantRuntimeSessionKindTeacherPreview,
		)
	if err != nil {
		t.Fatalf(
			"teacher_preview被公开开关错误拒绝: error=%v",
			err,
		)
	}

	// 未知类型必须按无效令牌处理。
	err =
		service.validateSessionKindEnabled(
			"unknown-session-kind",
		)
	if !errors.Is(
		err,
		ErrAssistantRuntimeTokenInvalid,
	) {
		t.Fatalf(
			"未知会话类型没有fail-closed: error=%v",
			err,
		)
	}
}

func TestAssistantRuntimeSessionKindGateAllowsExternalWhenEnabled(
	t *testing.T,
) {
	service :=
		newAssistantRuntimeGateTestService(t)

	service.SetPublicRuntimeEnabled(true)

	err :=
		service.validateSessionKindEnabled(
			models.AssistantRuntimeSessionKindExternal,
		)
	if err != nil {
		t.Fatalf(
			"公开开关开启后external仍被拒绝: error=%v",
			err,
		)
	}

	// 开启external后不能改变teacher_preview行为。
	err =
		service.validateSessionKindEnabled(
			models.AssistantRuntimeSessionKindTeacherPreview,
		)
	if err != nil {
		t.Fatalf(
			"公开开关开启后teacher_preview异常: error=%v",
			err,
		)
	}
}

func TestStartExternalSessionBlockedBeforeDatabaseWhenDisabled(
	t *testing.T,
) {
	service :=
		newAssistantRuntimeGateTestService(t)

	// 这里没有初始化数据库。
	//
	// 如果StartExternalSession没有在任何Repository调用前执行总闸门，
	// 测试会访问未初始化数据库并失败或发生异常。
	response, err :=
		service.StartExternalSession(
			context.Background(),
			"public-id",
			"https://course.example.com",
			"11111111-1111-4111-8111-111111111111",
			"127.0.0.1",
		)

	if response != nil {
		t.Fatalf(
			"公开开关关闭时不应创建external会话: response=%+v",
			response,
		)
	}

	if !errors.Is(
		err,
		ErrAssistantRuntimeDeploymentUnavailable,
	) {
		t.Fatalf(
			"公开开关关闭时错误类型不正确: error=%v",
			err,
		)
	}
}

func TestNilAssistantRuntimeSessionServiceFailsClosed(
	t *testing.T,
) {
	var service *AssistantRuntimeSessionService

	err :=
		service.validateSessionKindEnabled(
			models.AssistantRuntimeSessionKindExternal,
		)

	if !errors.Is(
		err,
		ErrAssistantRuntimeDeploymentUnavailable,
	) {
		t.Fatalf(
			"nil服务没有fail-closed: error=%v",
			err,
		)
	}
}
