package handlers

// courseware_style_studio_path.go — AI美术风格工作室路径解析
//
// 支持路径：
//   GET    /api/v1/coursewares/{id}/style-studio
//   POST   /api/v1/coursewares/{id}/style-studio/sessions
//   GET    /api/v1/coursewares/{id}/style-studio/sessions/{session_id}
//   DELETE /api/v1/coursewares/{id}/style-studio/sessions/{session_id}
//   POST   /api/v1/coursewares/{id}/style-studio/sessions/{session_id}/messages
//   POST   /api/v1/coursewares/{id}/style-studio/sessions/{session_id}/previews
//   POST   /api/v1/coursewares/{id}/style-studio/sessions/{session_id}/confirm
//   POST   /api/v1/coursewares/{id}/style-studio/upload-reference
//
// 本模块只解析路径结构。
// 登录、作者权限和教育域校验由Handler与Service共同执行。

import (
	"fmt"
	"strings"
)

const coursewareStyleStudioPathPrefix = "/api/v1/coursewares/"

const (
	coursewareStyleStudioActionActive = "active"

	coursewareStyleStudioActionSessions = "sessions"

	coursewareStyleStudioActionSession = "session"

	coursewareStyleStudioActionMessages = "messages"

	coursewareStyleStudioActionPreviews = "previews"

	coursewareStyleStudioActionConfirm = "confirm"

	coursewareStyleStudioActionUploadReference = "upload_reference"
)

// coursewareStyleStudioPath 风格工作室解析后的路径参数。
type coursewareStyleStudioPath struct {
	CoursewareID string
	SessionID    string
	Action       string
}

// parseCoursewareStyleStudioPath 解析风格工作室请求路径。
func parseCoursewareStyleStudioPath(
	path string,
) (*coursewareStyleStudioPath, error) {
	path = strings.TrimSpace(path)
	path = strings.TrimRight(path, "/")

	if !strings.HasPrefix(
		path,
		coursewareStyleStudioPathPrefix,
	) {
		return nil, fmt.Errorf(
			"不是课件风格工作室路径",
		)
	}

	remaining := strings.TrimPrefix(
		path,
		coursewareStyleStudioPathPrefix,
	)

	parts := strings.Split(
		remaining,
		"/",
	)

	if len(parts) < 2 {
		return nil, fmt.Errorf(
			"风格工作室路径缺少必要层级",
		)
	}

	coursewareID := strings.TrimSpace(
		parts[0],
	)

	if coursewareID == "" {
		return nil, fmt.Errorf(
			"风格工作室路径缺少课件ID",
		)
	}

	if strings.TrimSpace(parts[1]) !=
		"style-studio" {
		return nil, fmt.Errorf(
			"不是风格工作室路径",
		)
	}

	result := &coursewareStyleStudioPath{
		CoursewareID: coursewareID,
	}

	switch {
	// /api/v1/coursewares/{id}/style-studio
	case len(parts) == 2:
		result.Action =
			coursewareStyleStudioActionActive

		return result, nil

	// /api/v1/coursewares/{id}/style-studio/sessions
	case len(parts) == 3 &&
		parts[2] == "sessions":
		result.Action =
			coursewareStyleStudioActionSessions

		return result, nil

	// /api/v1/coursewares/{id}/style-studio/upload-reference
	case len(parts) == 3 &&
		parts[2] == "upload-reference":
		result.Action =
			coursewareStyleStudioActionUploadReference

		return result, nil

	// /api/v1/coursewares/{id}/style-studio/sessions/{session_id}
	case len(parts) == 4 &&
		parts[2] == "sessions":
		sessionID := strings.TrimSpace(
			parts[3],
		)

		if sessionID == "" {
			return nil, fmt.Errorf(
				"风格工作室路径缺少会话ID",
			)
		}

		result.SessionID = sessionID
		result.Action =
			coursewareStyleStudioActionSession

		return result, nil

	// /api/v1/coursewares/{id}/style-studio/sessions/{session_id}/{action}
	case len(parts) == 5 &&
		parts[2] == "sessions":
		sessionID := strings.TrimSpace(
			parts[3],
		)

		if sessionID == "" {
			return nil, fmt.Errorf(
				"风格工作室路径缺少会话ID",
			)
		}

		result.SessionID = sessionID

		switch parts[4] {
		case "messages":
			result.Action =
				coursewareStyleStudioActionMessages

		case "previews":
			result.Action =
				coursewareStyleStudioActionPreviews

		case "confirm":
			result.Action =
				coursewareStyleStudioActionConfirm

		default:
			return nil, fmt.Errorf(
				"未知的风格工作室会话操作: %s",
				parts[4],
			)
		}

		return result, nil

	default:
		return nil, fmt.Errorf(
			"风格工作室路径结构无效",
		)
	}
}
