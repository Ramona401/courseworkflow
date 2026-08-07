package services

// courseware_component_reference_guard.go — 页面组件引用写回守卫。
//
// 本文件收口所有“已有matched_component_ids继续写回页面”的路径：
//   - 单页AI微调；
//   - 页面历史版本恢复；
//   - 就地文字编辑；
//   - 后续任何保留旧组件ID的覆盖式页面写入。
//
// 新生成页面的组件来自同域匹配入口，可以直接写入；
// 历史引用则必须在每次继续保存前重新读取组件最小快照并执行：
//   - 组件存在；
//   - is_active=true；
//   - review_status=approved；
//   - 组件教育域等于课件域或为common。
//
// 历史引用采用兼容过滤：失效、异域或不存在ID被移除；
// 课件缺失、课件域非法、JSON非法或数据库查询失败则整体fail-closed。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
)

var (
	ErrCWComponentReferenceJSONInvalid = errors.New(
		"页面课件组件引用格式无效",
	)

	ErrCWComponentReferenceCoursewareRequired = errors.New(
		"页面课件组件引用缺少有效课件快照",
	)
)

// parseHistoricalCWComponentIDsJSON 解析历史matched_component_ids。
//
// 数据库列是jsonb，但服务层仍显式校验，避免测试对象、旧导入数据或
// 未来调用方把对象、数字等非字符串数组格式带入写回路径。
func parseHistoricalCWComponentIDsJSON(
	rawJSON string,
) ([]string, error) {
	rawJSON = strings.TrimSpace(rawJSON)

	if rawJSON == "" ||
		rawJSON == "null" {
		return []string{}, nil
	}

	var componentIDs []string

	if err := json.Unmarshal(
		[]byte(rawJSON),
		&componentIDs,
	); err != nil {
		return nil, fmt.Errorf(
			"%w: %v",
			ErrCWComponentReferenceJSONInvalid,
			err,
		)
	}

	return normalizeUniqueCWComponentIDs(
		componentIDs,
	), nil
}

// encodeHistoricalCWComponentIDsJSON 编码安全过滤后的组件ID。
//
// 空数组继续使用空字符串，使既有Repository的nullIfEmpty逻辑写入NULL，
// 不制造无意义的JSON空数组差异。
func encodeHistoricalCWComponentIDsJSON(
	componentIDs []string,
) string {
	componentIDs = normalizeUniqueCWComponentIDs(
		componentIDs,
	)

	if len(componentIDs) == 0 {
		return ""
	}

	data, err := json.Marshal(componentIDs)
	if err != nil {
		// []string理论上不会序列化失败；异常时仍fail-closed为空。
		return ""
	}

	return string(data)
}

// sanitizeHistoricalCWComponentIDsJSON 按课件快照域复核历史组件ID。
func sanitizeHistoricalCWComponentIDsJSON(
	ctx context.Context,
	rawJSON string,
	coursewareDomain string,
) (string, error) {
	componentIDs, err :=
		parseHistoricalCWComponentIDsJSON(
			rawJSON,
		)
	if err != nil {
		return "", err
	}

	filteredIDs, err :=
		FilterHistoricalCWComponentIDsForEducationDomain(
			ctx,
			componentIDs,
			coursewareDomain,
		)
	if err != nil {
		return "", err
	}

	return encodeHistoricalCWComponentIDsJSON(
		filteredIDs,
	), nil
}

// sanitizeCoursewarePageMatchedComponentIDsForWrite 使用正式课件快照
// 复核即将继续写回页面的历史组件ID。
//
// 调用方不能只传字符串教育域，必须传入刚刚通过访问服务重新读取的完整课件，
// 避免使用请求参数、Actor当前组织域或异步任务启动前的旧内存值。
func sanitizeCoursewarePageMatchedComponentIDsForWrite(
	ctx context.Context,
	courseware *models.Courseware,
	rawJSON string,
) (string, error) {
	if courseware == nil ||
		strings.TrimSpace(
			courseware.ID,
		) == "" {
		return "",
			ErrCWComponentReferenceCoursewareRequired
	}

	return sanitizeHistoricalCWComponentIDsJSON(
		ctx,
		rawJSON,
		courseware.EducationDomain,
	)
}
