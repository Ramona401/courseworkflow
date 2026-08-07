package utils

// aoci_image.go — 图片IAOCI基础编码、解析和格式化
//
// 协议结构：
//   - 第一行：机器编码，以竖线分隔；
//   - 后续行：[F][L][A][C][S][E][R][N]语义标签；
//   - JSON只能作为接口传输外壳，不是索引本体。
//
// 锚点清理和提纯位于aoci_image_anchor.go。

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"tedna/internal/models"
)

const imageAOCIAnchorKey = "@ANCHOR"

// BuildImageAOCIKey 根据稳定page_id和placeholder_id生成图片键。
func BuildImageAOCIKey(
	pageID string,
	placeholderID string,
) (string, error) {
	pageID = strings.TrimSpace(pageID)
	placeholderID = strings.TrimSpace(placeholderID)

	if pageID == "" {
		return "", fmt.Errorf(
			"生成图片键失败：page_id不能为空",
		)
	}
	if placeholderID == "" {
		return "", fmt.Errorf(
			"生成图片键失败：placeholder_id不能为空",
		)
	}

	digest := sha256.Sum256(
		[]byte(pageID + "\x00" + placeholderID),
	)

	shortHex := strings.ToUpper(
		hex.EncodeToString(digest[:]),
	)[:12]

	return "@I-" + shortHex, nil
}

// IsValidImageAOCIKey 校验稳定图片键。
func IsValidImageAOCIKey(value string) bool {
	value = strings.TrimSpace(value)

	if value == imageAOCIAnchorKey {
		return true
	}
	if len(value) != 15 ||
		!strings.HasPrefix(value, "@I-") {
		return false
	}

	for _, code := range value[3:] {
		if !strings.ContainsRune(
			"0123456789ABCDEF",
			code,
		) {
			return false
		}
	}

	return true
}

// ParseImageAOCIField 从第一行读取机器字段。
func ParseImageAOCIField(
	indexText string,
	fieldCode string,
) string {
	lines := nonEmptyImageAOCILines(indexText)
	if len(lines) == 0 {
		return ""
	}

	prefix := strings.TrimSpace(fieldCode) + ":"

	for _, segment := range strings.Split(
		lines[0],
		"|",
	) {
		segment = strings.TrimSpace(segment)

		if strings.HasPrefix(segment, prefix) {
			return strings.TrimSpace(
				segment[len(prefix):],
			)
		}
	}

	return ""
}

// ParseImageAOCI 严格解析完整IAOCI。
func ParseImageAOCI(
	indexText string,
) (*models.ImageAOCI, error) {
	lines := nonEmptyImageAOCILines(indexText)
	if len(lines) == 0 {
		return nil, fmt.Errorf("IAOCI为空")
	}

	header, err := parseImageAOCIHeader(lines[0])
	if err != nil {
		return nil, err
	}

	indexVersion, err := strconv.Atoi(header["IV"])
	if err != nil || indexVersion < 1 {
		return nil, fmt.Errorf(
			"IAOCI字段IV必须为大于等于1的整数",
		)
	}

	continuityLevel, err :=
		strconv.Atoi(header["CT"])
	if err != nil ||
		continuityLevel < 0 ||
		continuityLevel > 3 {
		return nil, fmt.Errorf(
			"IAOCI字段CT必须为0至3",
		)
	}

	aoci := &models.ImageAOCI{
		ImageKey:        header["IK"],
		IndexVersion:    indexVersion,
		IndexType:       header["IT"],
		UsageRole:       header["UR"],
		ContinuityLevel: continuityLevel,
		SubjectType:     header["SB"],
		AspectRatio:     header["AR"],
		RelationCount:   header["RC"],
		RawText: strings.TrimSpace(
			indexText,
		),
	}

	if !IsValidImageAOCIKey(aoci.ImageKey) {
		return nil, fmt.Errorf(
			"IAOCI字段IK不是合法图片键",
		)
	}
	if !models.IsValidCWImageIndexType(
		aoci.IndexType,
	) {
		return nil, fmt.Errorf("IAOCI字段IT不合法")
	}
	if !models.IsValidCWImageUsageRole(
		aoci.UsageRole,
	) {
		return nil, fmt.Errorf("IAOCI字段UR不合法")
	}
	if !models.IsValidCWImageSubjectType(
		aoci.SubjectType,
	) {
		return nil, fmt.Errorf("IAOCI字段SB不合法")
	}
	if !models.IsValidCWImageAspectRatio(
		aoci.AspectRatio,
	) {
		return nil, fmt.Errorf("IAOCI字段AR不合法")
	}
	if aoci.RelationCount != "0" &&
		aoci.RelationCount != "1" &&
		aoci.RelationCount != "M" {
		return nil, fmt.Errorf(
			"IAOCI字段RC只能为0、1或M",
		)
	}

	if aoci.IndexType ==
		models.CWImageIndexTypeAnchor {
		if aoci.ImageKey != imageAOCIAnchorKey {
			return nil, fmt.Errorf(
				"课程锚点IK必须为@ANCHOR",
			)
		}
	} else if aoci.ImageKey ==
		imageAOCIAnchorKey {
		return nil, fmt.Errorf(
			"页面图片不能使用@ANCHOR",
		)
	}

	tagValues := make(map[string]string)
	hasZeroRelation := false

	for _, line := range lines[1:] {
		if !isImageAOCISemanticLine(line) {
			return nil, fmt.Errorf(
				"IAOCI语义行格式错误：%s",
				line,
			)
		}

		tag := line[1:2]
		content := strings.TrimSpace(line[3:])

		if content == "" {
			return nil, fmt.Errorf(
				"IAOCI标签[%s]内容不能为空",
				tag,
			)
		}

		switch tag {
		case "R":
			if content == "0" {
				if len(aoci.Relations) > 0 {
					return nil, fmt.Errorf(
						"[R]0不能与其它关系并存",
					)
				}
				hasZeroRelation = true
				continue
			}

			if hasZeroRelation {
				return nil, fmt.Errorf(
					"[R]0不能与其它关系并存",
				)
			}

			relation, relationErr :=
				parseImageAOCIRelation(content)
			if relationErr != nil {
				return nil, relationErr
			}

			aoci.Relations = append(
				aoci.Relations,
				relation,
			)

		case "F", "L", "A", "C", "S", "E", "N":
			if _, duplicated := tagValues[tag]; duplicated {
				return nil, fmt.Errorf(
					"IAOCI标签[%s]重复",
					tag,
				)
			}
			tagValues[tag] = content

		default:
			return nil, fmt.Errorf(
				"IAOCI包含未知标签[%s]",
				tag,
			)
		}
	}

	for _, requiredTag := range []string{
		"F", "L", "A", "C", "S", "E", "N",
	} {
		if strings.TrimSpace(
			tagValues[requiredTag],
		) == "" {
			return nil, fmt.Errorf(
				"IAOCI缺少标签[%s]",
				requiredTag,
			)
		}
	}

	if !hasZeroRelation &&
		len(aoci.Relations) == 0 {
		return nil, fmt.Errorf(
			"IAOCI缺少[R]声明",
		)
	}

	actualCount :=
		models.CWImageRelationCountCode(
			len(aoci.Relations),
		)

	if actualCount != aoci.RelationCount {
		return nil, fmt.Errorf(
			"IAOCI关系数量不一致：RC=%s，实际=%d",
			aoci.RelationCount,
			len(aoci.Relations),
		)
	}

	aoci.FocusText = tagValues["F"]
	aoci.LayoutText = tagValues["L"]
	aoci.ArtText = tagValues["A"]
	aoci.CharacterText = tagValues["C"]
	aoci.SceneText = tagValues["S"]
	aoci.ExportText = tagValues["E"]
	aoci.NegativeText = tagValues["N"]

	return aoci, nil
}

// ValidateImageAOCI 验证IAOCI。
func ValidateImageAOCI(indexText string) error {
	_, err := ParseImageAOCI(indexText)
	return err
}

// FormatImageAOCI 确定性格式化IAOCI。
func FormatImageAOCI(
	aoci *models.ImageAOCI,
) (string, error) {
	if aoci == nil {
		return "", fmt.Errorf("IAOCI对象为空")
	}

	relationCount :=
		models.CWImageRelationCountCode(
			len(aoci.Relations),
		)

	var builder strings.Builder

	builder.WriteString(fmt.Sprintf(
		"IK:%s|IV:%d|IT:%s|UR:%s|CT:%d|SB:%s|AR:%s|RC:%s\n",
		strings.TrimSpace(aoci.ImageKey),
		aoci.IndexVersion,
		strings.TrimSpace(aoci.IndexType),
		strings.TrimSpace(aoci.UsageRole),
		aoci.ContinuityLevel,
		strings.TrimSpace(aoci.SubjectType),
		strings.TrimSpace(aoci.AspectRatio),
		relationCount,
	))

	builder.WriteString(
		"[F]" +
			normalizedImageAOCISemantic(aoci.FocusText) +
			"\n",
	)
	builder.WriteString(
		"[L]" +
			normalizedImageAOCISemantic(aoci.LayoutText) +
			"\n",
	)
	builder.WriteString(
		"[A]" +
			normalizedImageAOCISemantic(aoci.ArtText) +
			"\n",
	)
	builder.WriteString(
		"[C]" +
			normalizedImageAOCISemantic(aoci.CharacterText) +
			"\n",
	)
	builder.WriteString(
		"[S]" +
			normalizedImageAOCISemantic(aoci.SceneText) +
			"\n",
	)
	builder.WriteString(
		"[E]" +
			normalizedImageAOCISemantic(aoci.ExportText) +
			"\n",
	)

	if len(aoci.Relations) == 0 {
		builder.WriteString("[R]0\n")
	} else {
		seen := make(
			map[string]bool,
			len(aoci.Relations),
		)

		for _, relation := range aoci.Relations {
			line, err :=
				formatImageAOCIRelation(relation)
			if err != nil {
				return "", err
			}
			if seen[line] {
				return "", fmt.Errorf(
					"IAOCI包含重复关系：%s",
					line,
				)
			}

			seen[line] = true
			builder.WriteString("[R]" + line + "\n")
		}
	}

	builder.WriteString(
		"[N]" +
			normalizedImageAOCISemantic(
				aoci.NegativeText,
			),
	)

	formatted := builder.String()

	if _, err := ParseImageAOCI(formatted); err != nil {
		return "", fmt.Errorf(
			"格式化结果未通过自校验: %w",
			err,
		)
	}

	return formatted, nil
}

// BuildCoursewareImageIndexFromAOCI 转换为数据库模型。
func BuildCoursewareImageIndexFromAOCI(
	coursewareID string,
	pageID *string,
	placeholderID string,
	slotOrder int,
	aoci *models.ImageAOCI,
) (*models.CoursewareImageIndex, error) {
	if aoci == nil {
		return nil, fmt.Errorf("IAOCI对象为空")
	}

	formatted, err := FormatImageAOCI(aoci)
	if err != nil {
		return nil, err
	}

	return &models.CoursewareImageIndex{
		CoursewareID: strings.TrimSpace(
			coursewareID,
		),
		PageID: pageID,
		PlaceholderID: strings.TrimSpace(
			placeholderID,
		),
		ImageKey: strings.TrimSpace(
			aoci.ImageKey,
		),
		SlotOrder:       slotOrder,
		IndexVersion:    aoci.IndexVersion,
		IndexType:       aoci.IndexType,
		UsageRole:       aoci.UsageRole,
		ContinuityLevel: aoci.ContinuityLevel,
		SubjectType:     aoci.SubjectType,
		AspectRatio:     aoci.AspectRatio,
		RelationCount: models.CWImageRelationCountCode(
			len(aoci.Relations),
		),
		FocusText:     aoci.FocusText,
		LayoutText:    aoci.LayoutText,
		ArtText:       aoci.ArtText,
		CharacterText: aoci.CharacterText,
		SceneText:     aoci.SceneText,
		ExportText:    aoci.ExportText,
		NegativeText:  aoci.NegativeText,
		AOCIText:      formatted,
		Status:        models.CWImageIndexStatusPlanned,
		Version:       1,
	}, nil
}

func nonEmptyImageAOCILines(
	indexText string,
) []string {
	normalized := strings.ReplaceAll(
		indexText,
		"\r\n",
		"\n",
	)
	normalized = strings.ReplaceAll(
		normalized,
		"\r",
		"\n",
	)

	rawLines := strings.Split(normalized, "\n")
	lines := make([]string, 0, len(rawLines))

	for _, rawLine := range rawLines {
		line := strings.TrimSpace(rawLine)
		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

func parseImageAOCIHeader(
	line string,
) (map[string]string, error) {
	requiredCodes := []string{
		"IK", "IV", "IT", "UR",
		"CT", "SB", "AR", "RC",
	}

	result := make(
		map[string]string,
		len(requiredCodes),
	)

	for _, segment := range strings.Split(line, "|") {
		separator := strings.Index(segment, ":")
		if separator <= 0 {
			return nil, fmt.Errorf(
				"IAOCI编码段格式错误：%s",
				segment,
			)
		}

		code := strings.TrimSpace(
			segment[:separator],
		)
		value := strings.TrimSpace(
			segment[separator+1:],
		)

		if code == "" || value == "" {
			return nil, fmt.Errorf(
				"IAOCI编码字段不能为空",
			)
		}
		if _, duplicated := result[code]; duplicated {
			return nil, fmt.Errorf(
				"IAOCI编码字段重复：%s",
				code,
			)
		}

		result[code] = value
	}

	for _, code := range requiredCodes {
		if result[code] == "" {
			return nil, fmt.Errorf(
				"IAOCI缺少编码字段%s",
				code,
			)
		}
	}

	if len(result) != len(requiredCodes) {
		return nil, fmt.Errorf(
			"IAOCI包含未定义编码字段",
		)
	}

	return result, nil
}

func parseImageAOCIRelation(
	content string,
) (models.CoursewareImageRelationSpec, error) {
	content = strings.TrimSpace(content)
	relation := models.CoursewareImageRelationSpec{}

	switch {
	case strings.HasPrefix(
		content,
		models.CWImageRelationContrast,
	):
		relation.RelationCode =
			models.CWImageRelationContrast
		content = strings.TrimSpace(
			content[len(models.CWImageRelationContrast):],
		)

	case strings.HasPrefix(
		content,
		models.CWImageRelationContinue,
	):
		relation.RelationCode =
			models.CWImageRelationContinue
		content = strings.TrimSpace(content[1:])

	case strings.HasPrefix(
		content,
		models.CWImageRelationSameView,
	):
		relation.RelationCode =
			models.CWImageRelationSameView
		content = strings.TrimSpace(content[1:])

	case strings.HasPrefix(
		content,
		models.CWImageRelationParallel,
	):
		relation.RelationCode =
			models.CWImageRelationParallel
		content = strings.TrimSpace(content[1:])

	case strings.HasPrefix(
		content,
		models.CWImageRelationDetail,
	):
		relation.RelationCode =
			models.CWImageRelationDetail
		content = strings.TrimSpace(content[1:])

	default:
		return relation, fmt.Errorf(
			"IAOCI关系缺少合法关系符号",
		)
	}

	maskStart := strings.Index(content, "[")
	maskEnd := strings.Index(content, "]")

	if maskStart <= 0 || maskEnd <= maskStart {
		return relation, fmt.Errorf(
			"IAOCI关系缺少继承掩码",
		)
	}

	relation.TargetImageKey =
		strings.TrimSpace(content[:maskStart])

	if relation.TargetImageKey == imageAOCIAnchorKey ||
		!IsValidImageAOCIKey(
			relation.TargetImageKey,
		) {
		return relation, fmt.Errorf(
			"IAOCI关系目标必须是页面图片键",
		)
	}

	mask, ok :=
		models.NormalizeCWImageInheritMask(
			content[maskStart+1 : maskEnd],
		)
	if !ok {
		return relation, fmt.Errorf(
			"IAOCI关系继承掩码不合法",
		)
	}

	relation.InheritMask = mask

	rest := strings.TrimSpace(content[maskEnd+1:])
	if rest != "" {
		if !strings.HasPrefix(rest, "{") ||
			!strings.HasSuffix(rest, "}") {
			return relation, fmt.Errorf(
				"IAOCI关系说明必须使用大括号",
			)
		}

		relation.SemanticNote =
			strings.TrimSpace(
				rest[1 : len(rest)-1],
			)

		if strings.ContainsAny(
			relation.SemanticNote,
			"{}",
		) {
			return relation, fmt.Errorf(
				"关系说明不能嵌套大括号",
			)
		}
	}

	return relation, nil
}

func formatImageAOCIRelation(
	relation models.CoursewareImageRelationSpec,
) (string, error) {
	if !models.IsValidCWImageRelationCode(
		relation.RelationCode,
	) {
		return "", fmt.Errorf(
			"图片关系类型不合法",
		)
	}

	if relation.TargetImageKey == imageAOCIAnchorKey ||
		!IsValidImageAOCIKey(
			relation.TargetImageKey,
		) {
		return "", fmt.Errorf(
			"图片关系目标键不合法",
		)
	}

	mask, ok :=
		models.NormalizeCWImageInheritMask(
			relation.InheritMask,
		)
	if !ok {
		return "", fmt.Errorf(
			"图片关系掩码不合法",
		)
	}

	note := strings.TrimSpace(
		relation.SemanticNote,
	)
	if strings.ContainsAny(note, "{}") {
		return "", fmt.Errorf(
			"图片关系说明不能包含大括号",
		)
	}

	line :=
		relation.RelationCode +
			strings.TrimSpace(
				relation.TargetImageKey,
			) +
			"[" + mask + "]"

	if note != "" {
		line += "{" + note + "}"
	}

	return line, nil
}

func normalizedImageAOCISemantic(
	value string,
) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Ø"
	}

	return value
}

func isImageAOCISemanticLine(
	line string,
) bool {
	if len(line) < 3 ||
		line[0] != '[' ||
		line[2] != ']' {
		return false
	}

	switch line[1:2] {
	case "F", "L", "A", "C", "S", "E", "R", "N":
		return true
	default:
		return false
	}
}
