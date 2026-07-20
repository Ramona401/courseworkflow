package services

// video_draft_validation.go — 视频编辑器草稿输入治理
//
// 本文件只包含无数据库依赖的纯输入校验：
//   - 请求体最大2MB；
//   - 草稿名称最长100个字符；
//   - 每个草稿最多100个片段；
//   - clip_count必须与clips_data数组长度一致；
//   - 校验片段ID、URL、时长、裁剪区间、转场时长与独立音量；
//   - 规范化JSON后再交给Repository写入。
//
// 纯函数拆分后可以直接通过单元测试覆盖，不需要连接数据库。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

const (
	// VideoDraftMaxBodyBytes 是Handler和Service共同使用的请求上限。
	VideoDraftMaxBodyBytes = 2 << 20

	videoDraftMaxClips     = 100
	videoDraftMaxNameRunes = 100
	videoDraftMaxURLBytes  = 4096
	videoDraftMaxDuration  = 24 * 60 * 60
)

// videoDraftClipInput 只声明安全校验所需的核心字段。
//
// 其它可选编辑字段仍保留在原始JSON中，不会因为本结构体未声明而丢失。
type videoDraftClipInput struct {
	ID         string   `json:"id"`
	URL        string   `json:"url"`
	Duration   float64  `json:"duration"`
	TrimStart  float64  `json:"trimStart"`
	TrimEnd    float64  `json:"trimEnd"`
	Transition string   `json:"transition"`
	TransDur   float64  `json:"transDur"`
	AudioURL   string   `json:"audioUrl"`
	AudioVol   *float64 `json:"audioVolume"`
}

// normalizeVideoDraftInput 校验并规范化保存请求。
func normalizeVideoDraftInput(
	input *VideoDraftSaveInput,
) (
	string,
	string,
	int,
	error,
) {
	if input == nil {
		return "", "", 0, ErrVideoDraftInputInvalid
	}

	name := strings.TrimSpace(input.Name)
	if utf8.RuneCountInString(name) >
		videoDraftMaxNameRunes {
		return "", "", 0, fmt.Errorf(
			"%w: 草稿名称不能超过%d个字符",
			ErrVideoDraftInputInvalid,
			videoDraftMaxNameRunes,
		)
	}

	if len(input.ClipsData) == 0 {
		return "", "", 0, fmt.Errorf(
			"%w: clips_data不能为空",
			ErrVideoDraftInputInvalid,
		)
	}

	if len(input.ClipsData) >
		VideoDraftMaxBodyBytes {
		return "", "", 0, fmt.Errorf(
			"%w: 草稿数据不能超过2MB",
			ErrVideoDraftInputInvalid,
		)
	}

	var clips []videoDraftClipInput
	if err := json.Unmarshal(
		input.ClipsData,
		&clips,
	); err != nil {
		return "", "", 0, fmt.Errorf(
			"%w: clips_data必须是合法JSON数组",
			ErrVideoDraftInputInvalid,
		)
	}

	if len(clips) == 0 {
		return "", "", 0, fmt.Errorf(
			"%w: 草稿至少需要一个片段",
			ErrVideoDraftInputInvalid,
		)
	}

	if len(clips) >
		videoDraftMaxClips {
		return "", "", 0, fmt.Errorf(
			"%w: 单个草稿最多%d个片段",
			ErrVideoDraftInputInvalid,
			videoDraftMaxClips,
		)
	}

	if input.ClipCount != len(clips) {
		return "", "", 0, fmt.Errorf(
			"%w: clip_count与实际片段数量不一致",
			ErrVideoDraftInputInvalid,
		)
	}

	seenIDs := make(
		map[string]struct{},
		len(clips),
	)

	for index, clip := range clips {
		if err := validateVideoDraftClip(
			index,
			clip,
		); err != nil {
			return "", "", 0, err
		}

		normalizedID := strings.TrimSpace(clip.ID)
		if _, exists := seenIDs[normalizedID]; exists {
			return "", "", 0, fmt.Errorf(
				"%w: 第%d个片段ID重复",
				ErrVideoDraftInputInvalid,
				index+1,
			)
		}

		seenIDs[normalizedID] = struct{}{}
	}

	var compact bytes.Buffer
	if err := json.Compact(
		&compact,
		input.ClipsData,
	); err != nil {
		return "", "", 0, fmt.Errorf(
			"%w: clips_data格式错误",
			ErrVideoDraftInputInvalid,
		)
	}

	return name, compact.String(), len(clips), nil
}

// validateVideoDraftClip 校验单个片段的基本时间轴契约。
func validateVideoDraftClip(
	index int,
	clip videoDraftClipInput,
) error {
	position := index + 1

	if strings.TrimSpace(clip.ID) == "" {
		return fmt.Errorf(
			"%w: 第%d个片段缺少ID",
			ErrVideoDraftInputInvalid,
			position,
		)
	}

	url := strings.TrimSpace(clip.URL)
	if url == "" {
		return fmt.Errorf(
			"%w: 第%d个片段缺少URL",
			ErrVideoDraftInputInvalid,
			position,
		)
	}

	if len(url) > videoDraftMaxURLBytes ||
		strings.ContainsAny(
			url,
			"\r\n\x00",
		) {
		return fmt.Errorf(
			"%w: 第%d个片段URL无效",
			ErrVideoDraftInputInvalid,
			position,
		)
	}

	if !isFinitePositive(clip.Duration) ||
		clip.Duration > videoDraftMaxDuration {
		return fmt.Errorf(
			"%w: 第%d个片段时长无效",
			ErrVideoDraftInputInvalid,
			position,
		)
	}

	if !isFiniteNonNegative(clip.TrimStart) ||
		!isFinitePositive(clip.TrimEnd) ||
		clip.TrimEnd <= clip.TrimStart ||
		clip.TrimEnd > clip.Duration+0.05 {
		return fmt.Errorf(
			"%w: 第%d个片段裁剪区间无效",
			ErrVideoDraftInputInvalid,
			position,
		)
	}

	if !isFiniteNonNegative(clip.TransDur) ||
		clip.TransDur > 10 {
		return fmt.Errorf(
			"%w: 第%d个片段转场时长无效",
			ErrVideoDraftInputInvalid,
			position,
		)
	}

	if utf8.RuneCountInString(
		clip.Transition,
	) > 64 {
		return fmt.Errorf(
			"%w: 第%d个片段转场类型过长",
			ErrVideoDraftInputInvalid,
			position,
		)
	}

	if clip.AudioVol != nil {
		if math.IsNaN(*clip.AudioVol) ||
			math.IsInf(*clip.AudioVol, 0) ||
			*clip.AudioVol < 0 ||
			*clip.AudioVol > 1 {
			return fmt.Errorf(
				"%w: 第%d个片段音量无效",
				ErrVideoDraftInputInvalid,
				position,
			)
		}
	}

	if len(clip.AudioURL) >
		videoDraftMaxURLBytes ||
		strings.ContainsAny(
			clip.AudioURL,
			"\r\n\x00",
		) {
		return fmt.Errorf(
			"%w: 第%d个片段音频URL无效",
			ErrVideoDraftInputInvalid,
			position,
		)
	}

	return nil
}

// isFinitePositive 判断数值是否为有限正数。
func isFinitePositive(
	value float64,
) bool {
	return value > 0 &&
		!math.IsNaN(value) &&
		!math.IsInf(value, 0)
}

// isFiniteNonNegative 判断数值是否为有限非负数。
func isFiniteNonNegative(
	value float64,
) bool {
	return value >= 0 &&
		!math.IsNaN(value) &&
		!math.IsInf(value, 0)
}
