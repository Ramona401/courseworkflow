package models

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestImportExistingPlanRequestWordImportSessionIDJSON 验证浏览器提交的
// word_import_session_id会准确进入已有教案导入请求，而不是误落到其它模型。
func TestImportExistingPlanRequestWordImportSessionIDJSON(
	t *testing.T,
) {
	payload := []byte(`{
		"subject": "化学",
		"grade": "九年级",
		"topic": "酸碱盐",
		"source_type": "docx_fidelity",
		"word_import_session_id": "session-contract-001"
	}`)

	var request ImportExistingPlanRequest

	if err := json.Unmarshal(
		payload,
		&request,
	); err != nil {
		t.Fatalf(
			"解析Word确认导入请求失败: %v",
			err,
		)
	}

	if request.WordImportSessionID !=
		"session-contract-001" {
		t.Fatalf(
			"word_import_session_id解析位置错误，实际值=%q",
			request.WordImportSessionID,
		)
	}

	encoded, err := json.Marshal(
		request,
	)
	if err != nil {
		t.Fatalf(
			"序列化Word确认导入请求失败: %v",
			err,
		)
	}

	if !bytes.Contains(
		encoded,
		[]byte(`"word_import_session_id":"session-contract-001"`),
	) {
		t.Fatalf(
			"序列化结果缺少word_import_session_id: %s",
			string(encoded),
		)
	}
}

// TestExtractionListItemDoesNotExposeWordImportSessionID 防止内部Word导入
// 会话字段再次误加到萃取列表响应，造成无关协议污染。
func TestExtractionListItemDoesNotExposeWordImportSessionID(
	t *testing.T,
) {
	encoded, err := json.Marshal(
		ExtractionListItem{
			ID: "extraction-contract-001",
		},
	)
	if err != nil {
		t.Fatalf(
			"序列化萃取列表条目失败: %v",
			err,
		)
	}

	if bytes.Contains(
		encoded,
		[]byte("word_import_session_id"),
	) {
		t.Fatalf(
			"萃取列表不应暴露Word导入会话字段: %s",
			string(encoded),
		)
	}
}
