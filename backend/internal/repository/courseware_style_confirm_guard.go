package repository

// courseware_style_confirm_guard.go — 风格工作室确认来源纯规则
//
// 本文件把确认来源裁决从数据库查询中抽离为纯函数，便于定向测试。
//
// 规则顺序：
//   1. 请求reference_mode必须等于会话已经持久化的reference_mode；
//   2. 当前会话状态为generated的预览图可以确认；
//   3. 只有style_character模式可以直接确认当前会话参考图；
//   4. 其它同课件图片一律拒绝。
//
// 第1条必须先执行。这样老师或恶意客户端不能临时把请求模式改成
// style_character，再绕过会话模式持久化直接确认原始参考图。

import (
	"strings"

	"tedna/internal/models"
)

func validateCoursewareStyleConfirmSelection(
	referenceMode string,
	storedReferenceMode string,
	storedReferenceAssetID *string,
	confirmedAssetID string,
	generatedPreview bool,
) error {
	referenceMode =
		strings.TrimSpace(referenceMode)

	storedReferenceMode =
		strings.TrimSpace(storedReferenceMode)

	confirmedAssetID =
		strings.TrimSpace(confirmedAssetID)

	if referenceMode !=
		storedReferenceMode {
		return ErrCoursewareStylePreviewModeStale
	}

	if generatedPreview {
		return nil
	}

	if referenceMode ==
		models.CWStyleReferenceModeCharacter &&
		storedReferenceAssetID != nil &&
		strings.TrimSpace(
			*storedReferenceAssetID,
		) == confirmedAssetID {
		return nil
	}

	return ErrCoursewareStyleConfirmAssetInvalid
}
