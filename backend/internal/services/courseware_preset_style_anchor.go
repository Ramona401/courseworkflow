package services

// courseware_preset_style_anchor.go — 课件快捷预设画风的确定性锚点服务
//
// 快捷预设现在不再为每个课件重复调用图片模型：
//
//  1. 系统预先生成十张真实高清画风图；
//  2. 前端展示对应的轻量卡片图；
//  3. 老师只需选择一个预设键；
//  4. 后端读取服务器白名单和系统高清图；
//  5. 创建属于当前课件的锚点资产记录；
//  6. 确定性构造IAOCI并写入课程锚点。
//
// 系统图片复用安全：
//   - 课程资产的OssURL保存为完整HTTPS公网地址；
//   - 普通资产删除逻辑只删除以CWAssetURLPrefix开头的本地文件；
//   - 因此删除某个课件的锚点资产不会删除系统共享缩略图；
//   - 不复制高清文件，不额外占用磁盘，也不产生图片模型费用。
//
// 兼容说明：
//   - 方法签名继续保留assetID参数，以兼容旧调用方；
//   - 快捷预设模式不再使用该参数；
//   - 无preset_style_key的手动图片锚点仍走原有多模态链路。

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// coursewarePresetStyleDefinition 是服务器端可信的快捷画风定义。
type coursewarePresetStyleDefinition struct {
	Label   string
	ArtText string
}

// coursewarePresetStyleOrder 是前端展示和系统缩略图生成的稳定顺序。
var coursewarePresetStyleOrder = []string{
	"pixar",
	"flat",
	"ghibli",
	"realistic",
	"chinese",
	"ink_wash",
	"guochao",
	"storybook",
	"science",
	"tech",
}

// coursewarePresetStyleDefinitions 是快捷预设画风的后端唯一事实源。
var coursewarePresetStyleDefinitions = map[string]coursewarePresetStyleDefinition{
	"pixar": {
		Label: "皮克斯3D",
		ArtText: "高品质三维动画渲染语言；" +
			"圆润饱满的造型设计；柔和全局光照与体积光；" +
			"明快温暖且层次清晰的配色；细腻材质与电影级CG质感",
	},
	"flat": {
		Label: "扁平插画",
		ArtText: "现代扁平矢量插画语言；简洁几何色块；" +
			"干净利落的描边；明快和谐的教育风配色；" +
			"克制使用渐变与阴影；信息层次清晰",
	},
	"ghibli": {
		Label: "治愈手绘",
		ArtText: "温暖细腻的日系手绘动画插画语言；" +
			"柔和水彩质感与自然手绘颗粒；" +
			"清新治愈的色调；通透柔和的自然光影；" +
			"富有故事感但不过度装饰",
	},
	"realistic": {
		Label: "写实摄影",
		ArtText: "高清写实摄影语言；照片级真实感；" +
			"自然可信的光影和景深；细腻真实的材质纹理；" +
			"专业摄影构图质量；禁止卡通化和插画化渲染",
	},
	"chinese": {
		Label: "东方雅韵",
		ArtText: "宽泛的新中式东方美学语言；" +
			"雅致克制的东方配色；讲究留白、节奏和含蓄层次；" +
			"可按具体教学内容灵活融合淡彩、水墨晕染、工笔线条或宋式清雅质感；" +
			"不锁定具体朝代、服饰、器物和单一绘画技法",
	},
	"ink_wash": {
		Label: "水墨写意",
		ArtText: "中国水墨写意语言；墨色浓淡干湿变化；" +
			"宣纸肌理与含蓄淡彩点染；大面积留白；" +
			"笔触简练有力；整体意境清雅通透",
	},
	"guochao": {
		Label: "现代国潮",
		ArtText: "现代国潮插画语言；传统东方纹样与现代平面构成融合；" +
			"朱红、黛青、鎏金等高识别度配色；" +
			"装饰性云纹与几何秩序并存；精致醒目但不过度繁复",
	},
	"storybook": {
		Label: "儿童绘本",
		ArtText: "温柔儿童绘本插画语言；粉彩与蜡笔般的柔和笔触；" +
			"轻微纸张颗粒；明亮但不刺眼的配色；" +
			"简洁亲切的造型；富有童真与叙事感",
	},
	"science": {
		Label: "科普线稿",
		ArtText: "现代科普线稿与信息图解语言；" +
			"清晰准确的结构线条；克制的功能性色彩；" +
			"干净背景与明确层级；兼顾科学严谨性和教学可读性；" +
			"避免无关装饰",
	},
	"tech": {
		Label: "未来科技",
		ArtText: "未来数字科技视觉语言；深色基底与克制霓虹光效；" +
			"流动数据线条、粒子光点和几何网格；" +
			"层次清晰的冷色光影；具有高级数字质感",
	},
}

// CoursewarePresetStyleDescriptor 是系统缩略图命令使用的只读描述。
type CoursewarePresetStyleDescriptor struct {
	Key     string
	Label   string
	ArtText string
}

// ListCoursewarePresetStyleDescriptors 按稳定顺序返回全部预设画风。
func ListCoursewarePresetStyleDescriptors() []CoursewarePresetStyleDescriptor {
	result := make(
		[]CoursewarePresetStyleDescriptor,
		0,
		len(coursewarePresetStyleOrder),
	)

	for _, key := range coursewarePresetStyleOrder {
		definition, exists := coursewarePresetStyleDefinitions[key]

		if !exists {
			continue
		}

		result = append(
			result,
			CoursewarePresetStyleDescriptor{
				Key:     key,
				Label:   definition.Label,
				ArtText: definition.ArtText,
			},
		)
	}

	return result
}

// IsCoursewarePresetStyleKey 判断请求键是否属于服务器白名单。
func IsCoursewarePresetStyleKey(
	value string,
) bool {
	key := strings.ToLower(
		strings.TrimSpace(value),
	)

	_, exists := coursewarePresetStyleDefinitions[key]

	return exists
}

// SetPresetStyleAnchor 直接使用系统预设高清图设置课程风格锚点。
//
// assetID参数仅为兼容旧调用方保留，快捷预设模式不再读取或信任它。
func (s *CoursewareAssetService) SetPresetStyleAnchor(
	ctx context.Context,
	coursewareID string,
	_ string,
	presetStyleKey string,
	actor *CoursewareActorContext,
) (*SetStyleAnchorResult, error) {
	// 所有资产创建和锚点写入前重新执行作者运行域授权。
	if _, _, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		); err != nil {
		return nil, err
	}

	presetStyleKey = strings.ToLower(
		strings.TrimSpace(presetStyleKey),
	)

	if !IsCoursewarePresetStyleKey(
		presetStyleKey,
	) {
		return nil, fmt.Errorf(
			"不支持的快捷预设画风: %s",
			presetStyleKey,
		)
	}

	thumbnail, err :=
		resolveSystemPresetStyleThumbnail(
			presetStyleKey,
		)
	if err != nil {
		return nil, err
	}

	anchorURL, err :=
		resolveSystemPresetStyleThumbnailPublicURL(
			thumbnail.URL,
		)
	if err != nil {
		return nil, err
	}

	vaoci, err :=
		buildCoursewarePresetAnchorAOCI(
			presetStyleKey,
		)
	if err != nil {
		return nil, err
	}

	metadataJSON, _ :=
		json.Marshal(
			map[string]interface{}{
				"asset_role":
					"system_preset_style_anchor",
				"preset_style_key":
					presetStyleKey,
				"system_thumbnail_url":
					thumbnail.URL,
				"shared_system_file":
					true,
			},
		)

	// 使用完整HTTPS地址而不是本地/uploads路径。
	// 这样删除课程资产时不会删除系统共享物理文件。
	asset := &models.CoursewareAsset{
		CoursewareID: coursewareID,
		PageID:       nil,
		PlaceholderID: "preset-style:" +
			presetStyleKey,
		AssetType:        models.CWAssetTypeImage,
		GenerationPrompt: "",
		OssURL:           anchorURL,
		FileSize:         thumbnail.FileSize,
		MimeType:         thumbnail.MimeType,
		Metadata:         string(metadataJSON),
		Status:           models.CWAssetStatusUploaded,
	}

	if err :=
		repository.CreateCWAsset(
			ctx,
			asset,
		); err != nil {
		return nil, fmt.Errorf(
			"创建快捷预设锚点资产失败: %w",
			err,
		)
	}

	if err :=
		repository.UpdateCoursewareStyleAnchor(
			ctx,
			coursewareID,
			asset.ID,
			vaoci,
		); err != nil {
		// 锚点写入失败时删除刚创建的孤立资产记录。
		// OssURL是共享完整公网地址，不会触碰系统图片文件。
		if rollbackErr :=
			repository.DeleteCWAsset(
				ctx,
				asset.ID,
			); rollbackErr != nil {
			cwAssetLog.Warn(
				"快捷预设锚点写入失败后清理孤立资产记录失败",
				"courseware_id", coursewareID,
				"asset_id", asset.ID,
				"error", rollbackErr,
			)
		}

		return nil, fmt.Errorf(
			"保存快捷预设风格锚点失败: %w",
			err,
		)
	}

	definition := coursewarePresetStyleDefinitions[presetStyleKey]

	cwAssetLog.Info(
		"系统预设缩略图已直接设置为课程风格锚点",
		"courseware_id", coursewareID,
		"anchor_asset_id", asset.ID,
		"preset_style_key", presetStyleKey,
		"preset_style_label", definition.Label,
		"anchor_url", anchorURL,
		"vaoci_len", len([]rune(vaoci)),
	)

	return &SetStyleAnchorResult{
		AssetID:   asset.ID,
		AnchorURL: anchorURL,
		VAOCI:     vaoci,
	}, nil
}

// resolveSystemPresetStyleThumbnail 查找系统已生成的高清预设图片。
func resolveSystemPresetStyleThumbnail(
	presetStyleKey string,
) (
	CoursewarePresetStyleThumbnailItem,
	error,
) {
	definition, exists :=
		coursewarePresetStyleDefinitions[presetStyleKey]

	if !exists {
		return CoursewarePresetStyleThumbnailItem{},
			fmt.Errorf(
				"快捷预设画风不存在: %s",
				presetStyleKey,
			)
	}

	descriptor := CoursewarePresetStyleDescriptor{
		Key:     presetStyleKey,
		Label:   definition.Label,
		ArtText: definition.ArtText,
	}

	outputDirectory := filepath.Join(
		CWAssetUploadDir,
		cwPresetThumbnailDirName,
	)

	thumbnail, found :=
		findExistingPresetStyleThumbnail(
			outputDirectory,
			descriptor,
		)

	if !found {
		return CoursewarePresetStyleThumbnailItem{},
			fmt.Errorf(
				"系统预设画风图片尚未生成: %s",
				definition.Label,
			)
	}

	return thumbnail, nil
}

// resolveSystemPresetStyleThumbnailPublicURL 将系统图片转为稳定公网地址。
func resolveSystemPresetStyleThumbnailPublicURL(
	value string,
) (string, error) {
	url := strings.TrimSpace(value)

	if url == "" {
		return "", fmt.Errorf(
			"系统预设画风图片地址为空",
		)
	}

	if strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "http://") {
		return url, nil
	}

	if strings.HasPrefix(url, "/uploads/") {
		return cwAssetPublicHost + url, nil
	}

	return "", fmt.Errorf(
		"系统预设画风图片地址不合法",
	)
}

// buildCoursewarePresetAnchorAOCI 根据服务器白名单构造课程锚点IAOCI。
func buildCoursewarePresetAnchorAOCI(
	presetStyleKey string,
) (string, error) {
	definition, exists :=
		coursewarePresetStyleDefinitions[presetStyleKey]

	if !exists {
		return "", fmt.Errorf(
			"快捷预设画风不存在: %s",
			presetStyleKey,
		)
	}

	aoci := &models.ImageAOCI{
		ImageKey:        "@ANCHOR",
		IndexVersion:    1,
		IndexType:       models.CWImageIndexTypeAnchor,
		UsageRole:       models.CWImageUsageBackground,
		ContinuityLevel: 0,
		SubjectType:     models.CWImageSubjectNone,
		AspectRatio:     models.CWImageAspectFlexible,
		RelationCount:   "0",

		FocusText: "定义本课件统一的" +
			definition.Label +
			"艺术语言；具体页面主体由教学内容和图片槽位单独决定",

		LayoutText: "Ø；课程锚点不锁定具体构图、镜头、景别、" +
			"主体位置、画面比例和留白",

		ArtText: definition.ArtText,

		CharacterText: "Ø",

		SceneText: "Ø；样板图中的教室、家具、背景、道具和" +
			"具体空间关系均不作为跨页面约束",

		ExportText: "保持统一渲染质量；具体尺寸、画幅、清晰度和" +
			"留白由每个页面图片槽位决定",

		NegativeText: "禁止可读文字、Logo和水印；" +
			"禁止继承样板图中的环境、家具、构图、镜头、景别和道具位置；" +
			"禁止把人物或固定主体强行加入不需要人物的页面",

		Relations: nil,
	}

	formatted, err :=
		utils.FormatImageAOCI(aoci)
	if err != nil {
		return "", fmt.Errorf(
			"构造快捷预设风格IAOCI失败: %w",
			err,
		)
	}

	return formatted, nil
}
