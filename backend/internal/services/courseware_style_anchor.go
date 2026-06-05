package services

// courseware_style_anchor.go — 课件配图「风格锚点」服务（VAOCI 课程级风格一致性）
//
// 轮1职责（仅底层提取能力，不接路由不接业务流程）：
//   - ExtractVAOCIFromImageURL: 给定一张图的【公网可访问URL】，多模态读图，
//     按 prompt_courseware_vaoci_extract 规范提取该图的 VAOCI 风格索引文本。
//
// 设计要点：
//   - 复用已验证可跑的多模态范式 ai.CallAIMultimodal（refinePage 同款）。
//     CallAIMultimodal 的 imageDataURI 参数既支持 data:image/...;base64,xxx，
//     也支持 https:// 公网URL（见 MultimodalImageURL.URL 注释）；图生图/读图都要求公网可达，
//     故本函数直接收公网URL传入，不在轮1触碰上云逻辑。
//   - 场景码复用 sceneCWMediaPrompt("courseware_media_prompt")——该场景已用于
//     AI写提示词，确认支持多模态回退；未在 ai_scene_configs 显式配置时自动回退全局默认模型。
//   - 容错：VAOCI 提取本质依赖"读图"，降级纯文本无意义，故多模态失败直接返回错误；
//     但 AI 返回的内容【原样存储】（不强校验是否严格符合 VAOCI 单图格式）——
//     即便格式略有偏差，索引文字约束仍可用，图生图(ref_image)不受影响（见PRD 6.1容错条款）。
//
// 后续轮次：
//   - 轮2：设/查/清锚点接口 + 生成链路自动套用（调用本文件的提取能力 + repo 的
//     UpdateCoursewareStyleAnchor/ClearCoursewareStyleAnchor）。
//   - 本文件方法挂在 CoursewareAssetService 上，复用其 s.cfg（与 courseware_asset_prompt.go 一致）。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/repository"
)

// vaociExtractPromptKey 图→VAOCI 提取的系统提示词键（内嵌完整VAOCI单图索引规范）
const vaociExtractPromptKey = "prompt_courseware_vaoci_extract"

// vaociExtractUserText 多模态调用时随图附带的用户文本指令
// 系统提示词已内嵌完整规范，这里只做简短引导，主依据是图本身
const vaociExtractUserText = "请按系统提示词的 VAOCI 单图索引规范，分析这张课件配图：既提取视觉风格（A 属性：风格关键词+光影+色彩+质感），也提取画面中人物/主体角色的固定外貌特征（C 角色，若有），输出一行 VAOCI 索引。"

// ExtractVAOCIFromImageURL 多模态读图，提取一张课件配图的 VAOCI 风格索引文本。
//
// 入参：
//   - imageURL: 图片的【公网可访问URL】（http/https）。调用方负责保证可达
//     （优先用资产的 public_oss_url；未上云的本地图需先补全公网前缀或先上云，由轮2业务流程兜底）。
//   - userID: 当前操作用户ID，用于AI调用追踪埋点（trace）。
//
// 出参：
//   - string: AI 提取的 VAOCI 索引文本（原样返回，已去首尾空白）。
//   - error: 提取失败（配置缺失/提示词缺失/多模态调用失败/AI返回空）。
func (s *CoursewareAssetService) ExtractVAOCIFromImageURL(ctx context.Context, imageURL string, userID string) (string, error) {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return "", fmt.Errorf("锚点图URL为空，无法提取VAOCI")
	}
	// 必须是公网可达URL（多模态读图要求服务端能下载到图）
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		return "", fmt.Errorf("锚点图URL必须是公网可访问的http(s)地址，当前为: %s", imageURL)
	}

	// 1. 加载 VAOCI 提取系统提示词
	sysPrompt, err := repository.GetCurrentPromptByKey(vaociExtractPromptKey)
	if err != nil {
		return "", fmt.Errorf("加载VAOCI提取提示词失败(%s): %w", vaociExtractPromptKey, err)
	}

	// 2. 获取AI配置（复用 courseware_media_prompt 场景，支持多模态，未配置则回退全局默认）
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), sceneCWMediaPrompt,
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		return "", fmt.Errorf("获取AI配置失败: %w", err)
	}

	// 3. 多模态调用：把公网图URL作为 imageDataURI 传入（CallAIMultimodal 支持URL）
	//    注意：VAOCI 提取依赖读图，多模态失败不降级纯文本（无意义），直接返回错误
	traceCtx := &ai.TraceContext{SceneCode: sceneCWMediaPrompt, UserID: &userID}
	result, aiErr := ai.CallAIMultimodal(aiCfg, sysPrompt.Content, vaociExtractUserText, imageURL, traceCtx)
	if aiErr != nil {
		return "", fmt.Errorf("多模态读图提取VAOCI失败: %w", aiErr)
	}

	// 4. 原样返回AI输出（去首尾空白；不强校验VAOCI格式——容错条款：格式偏差也存，索引文字约束仍可用）
	vaoci := strings.TrimSpace(result.Content)
	if vaoci == "" {
		return "", fmt.Errorf("AI未返回有效的VAOCI索引文本")
	}

	cwAssetLog.Info("VAOCI风格索引提取成功",
		"image_url", imageURL,
		"vaoci_len", len([]rune(vaoci)),
		"model", result.ModelUsed,
		"tokens", result.TokensUsed,
	)
	return vaoci, nil
}
