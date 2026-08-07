package services

// lesson_plan_word_parser.go — DOCX原格式教案安全解析器
//
// 第一阶段解析目标：
//   - 只读取DOCX压缩包内受控OOXML部件，不执行宏、不访问外链；
//   - 保留表格、单元格、合并关系、段落、运行级粗体/上下标、图片引用和公式占位；
//   - 为每个可编辑段落生成稳定block_id；
//   - 生成一份供AI评审、索引和课件生成继续使用的语义Markdown；
//   - 原始DOCX始终作为版式母版保存，解析器不尝试在浏览器中复刻完整Word排版。

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"

	"tedna/internal/models"
)

const (
	lessonPlanWordParserVersion        = "ooxml-v1"
	maxLessonPlanWordZipEntries        = 2000
	maxLessonPlanWordUncompressedBytes = int64(160 * 1024 * 1024)
	maxLessonPlanWordEntryBytes        = int64(64 * 1024 * 1024)
	maxLessonPlanWordXMLBytes          = int64(48 * 1024 * 1024)
	maxLessonPlanWordCompressionRatio  = uint64(500)
)

var errLessonPlanWordDOCXInvalid = errors.New("DOCX结构无效")

// LessonPlanWordPreviewDocument 是可安全返回浏览器的结构化Word预览。
func parseLessonPlanWordDOCX(filePath string) (*lessonPlanWordParseResult, error) {
	archiveReader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w：文件不是有效的DOCX压缩包", errLessonPlanWordDOCXInvalid)
	}
	defer archiveReader.Close()

	entries, err := validateLessonPlanWordArchive(archiveReader.File)
	if err != nil {
		return nil, err
	}

	contentTypesXML, err := readLessonPlanWordZipEntry(
		entries,
		"[Content_Types].xml",
		maxLessonPlanWordXMLBytes,
	)
	if err != nil {
		return nil, err
	}

	if bytes.Contains(bytes.ToLower(contentTypesXML), []byte("macroenabled")) {
		return nil, fmt.Errorf("%w：不支持带宏的Word文档", errLessonPlanWordDOCXInvalid)
	}

	documentXML, err := readLessonPlanWordZipEntry(
		entries,
		"word/document.xml",
		maxLessonPlanWordXMLBytes,
	)
	if err != nil {
		return nil, err
	}

	relationships := make(map[string]lessonPlanWordRelationship)
	if _, ok := entries["word/_rels/document.xml.rels"]; ok {
		relationshipXML, readErr := readLessonPlanWordZipEntry(
			entries,
			"word/_rels/document.xml.rels",
			maxLessonPlanWordXMLBytes,
		)
		if readErr != nil {
			return nil, readErr
		}
		relationships, err = parseLessonPlanWordRelationships(relationshipXML)
		if err != nil {
			return nil, err
		}
	}

	warnings := newLessonPlanWordWarningCollector()
	document, metrics, referencedMedia, err := parseLessonPlanWordDocumentXML(
		documentXML,
		relationships,
		warnings,
	)
	if err != nil {
		return nil, err
	}

	for _, relationship := range relationships {
		if !strings.HasSuffix(strings.ToLower(relationship.Type), "/image") {
			if strings.EqualFold(relationship.TargetMode, "External") {
				metrics.ExternalRelationshipCount++
				warnings.add(
					"external_relationship",
					"文档包含外部链接或外部对象；平台不会访问这些外部地址",
					1,
				)
			}
			continue
		}

		normalizedTarget := normalizeLessonPlanWordRelationshipTarget(
			relationship.Target,
		)
		_, exists := entries[normalizedTarget]
		if strings.EqualFold(relationship.TargetMode, "External") {
			exists = false
			metrics.ExternalRelationshipCount++
			warnings.add(
				"external_image",
				"文档包含外部图片；为保护隐私和稳定性，平台不会联网拉取",
				1,
			)
		}

		contentType := inferLessonPlanWordMediaContentType(normalizedTarget)
		if contentType == "image/x-wmf" || contentType == "image/x-emf" {
			warnings.add(
				"legacy_vector_image",
				"文档包含WMF/EMF老式矢量图；原文件会保留，网页预览可能降级",
				1,
			)
		}

		document.Media = append(document.Media, LessonPlanWordPreviewMedia{
			RelationshipID: relationship.ID,
			Target:         normalizedTarget,
			ContentType:    contentType,
			TargetMode:     relationship.TargetMode,
			Referenced:     referencedMedia[relationship.ID],
			Missing:        !exists,
		})

		if referencedMedia[relationship.ID] && !exists {
			warnings.add(
				"missing_media",
				"文档引用了缺失或不可访问的图片部件",
				1,
			)
		}
	}

	sort.Slice(document.Media, func(i, j int) bool {
		return document.Media[i].RelationshipID < document.Media[j].RelationshipID
	})

	uniqueImages := 0
	for _, media := range document.Media {
		if media.Referenced {
			uniqueImages++
		}
	}
	metrics.UniqueImageCount = uniqueImages
	metrics.BlockCount = len(document.Blocks)
	metrics.TableCount = len(document.Tables)
	metrics.FormulaCount = len(document.Formulas)

	semanticMarkdown := buildLessonPlanWordSemanticMarkdown(document)
	semanticMarkdown = strings.TrimSpace(semanticMarkdown)
	if semanticMarkdown == "" {
		return nil, fmt.Errorf("%w：未提取到可用教学内容", errLessonPlanWordDOCXInvalid)
	}
	metrics.CharacterCount = len([]rune(strings.ReplaceAll(semanticMarkdown, " ", "")))

	structureJSONBytes, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("序列化Word结构失败: %w", err)
	}
	metricsJSONBytes, err := json.Marshal(metrics)
	if err != nil {
		return nil, fmt.Errorf("序列化Word指标失败: %w", err)
	}
	warningList := warnings.list()
	warningsJSONBytes, err := json.Marshal(warningList)
	if err != nil {
		return nil, fmt.Errorf("序列化Word告警失败: %w", err)
	}

	semanticHash := sha256.Sum256([]byte(semanticMarkdown))

	return &lessonPlanWordParseResult{
		Payload: models.LessonPlanWordParsedPayload{
			ParserVersion:          lessonPlanWordParserVersion,
			StructureSchemaVersion: models.LessonPlanWordStructureSchemaVersion,
			StructureJSON:          string(structureJSONBytes),
			SemanticMarkdown:       semanticMarkdown,
			SemanticMarkdownHash:   hex.EncodeToString(semanticHash[:]),
			MetricsJSON:            string(metricsJSONBytes),
			WarningsJSON:           string(warningsJSONBytes),
		},
		Document: document,
		Metrics:  metrics,
		Warnings: warningList,
	}, nil
}

func validateLessonPlanWordArchive(
	files []*zip.File,
) (map[string]*zip.File, error) {
	if len(files) == 0 || len(files) > maxLessonPlanWordZipEntries {
		return nil, fmt.Errorf(
			"%w：压缩包文件数量异常",
			errLessonPlanWordDOCXInvalid,
		)
	}

	entries := make(map[string]*zip.File, len(files))
	var totalUncompressed uint64

	for _, file := range files {
		name := strings.ReplaceAll(file.Name, "\\", "/")
		cleanInput := strings.TrimSuffix(name, "/")
		cleanName := path.Clean(cleanInput)

		if name == "" ||
			cleanInput == "" ||
			strings.ContainsRune(name, '\x00') ||
			strings.HasPrefix(name, "/") ||
			cleanName == "." ||
			cleanName == ".." ||
			strings.HasPrefix(cleanName, "../") ||
			cleanName != cleanInput {
			return nil, fmt.Errorf(
				"%w：压缩包包含不安全路径",
				errLessonPlanWordDOCXInvalid,
			)
		}

		if file.Mode()&0o170000 == 0o120000 {
			return nil, fmt.Errorf(
				"%w：压缩包包含符号链接",
				errLessonPlanWordDOCXInvalid,
			)
		}

		if file.Flags&0x1 != 0 {
			return nil, fmt.Errorf(
				"%w：不支持加密的Word文档",
				errLessonPlanWordDOCXInvalid,
			)
		}

		if file.UncompressedSize64 > uint64(maxLessonPlanWordEntryBytes) {
			return nil, fmt.Errorf(
				"%w：压缩包内单个部件过大",
				errLessonPlanWordDOCXInvalid,
			)
		}

		if file.CompressedSize64 > 0 &&
			file.UncompressedSize64 > 1024*1024 &&
			file.UncompressedSize64/file.CompressedSize64 >
				maxLessonPlanWordCompressionRatio {
			return nil, fmt.Errorf(
				"%w：压缩比异常，疑似压缩炸弹",
				errLessonPlanWordDOCXInvalid,
			)
		}

		totalUncompressed += file.UncompressedSize64
		if totalUncompressed > uint64(maxLessonPlanWordUncompressedBytes) {
			return nil, fmt.Errorf(
				"%w：解压后总体积过大",
				errLessonPlanWordDOCXInvalid,
			)
		}

		if _, exists := entries[name]; exists {
			return nil, fmt.Errorf(
				"%w：压缩包包含重复部件",
				errLessonPlanWordDOCXInvalid,
			)
		}
		entries[name] = file
	}

	for _, requiredName := range []string{
		"[Content_Types].xml",
		"word/document.xml",
	} {
		if _, ok := entries[requiredName]; !ok {
			return nil, fmt.Errorf(
				"%w：缺少%s",
				errLessonPlanWordDOCXInvalid,
				requiredName,
			)
		}
	}

	return entries, nil
}

func readLessonPlanWordZipEntry(
	entries map[string]*zip.File,
	name string,
	limit int64,
) ([]byte, error) {
	file, ok := entries[name]
	if !ok {
		return nil, fmt.Errorf("%w：缺少%s", errLessonPlanWordDOCXInvalid, name)
	}
	if file.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("%w：%s过大", errLessonPlanWordDOCXInvalid, name)
	}

	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("读取%s失败: %w", name, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("读取%s失败: %w", name, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w：%s超过解析上限", errLessonPlanWordDOCXInvalid, name)
	}
	return data, nil
}

func parseLessonPlanWordRelationships(
	data []byte,
) (map[string]lessonPlanWordRelationship, error) {
	type relationshipXML struct {
		ID         string `xml:"Id,attr"`
		Type       string `xml:"Type,attr"`
		Target     string `xml:"Target,attr"`
		TargetMode string `xml:"TargetMode,attr"`
	}
	type relationshipsXML struct {
		Items []relationshipXML `xml:"Relationship"`
	}

	var document relationshipsXML
	if err := xml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("解析Word关系文件失败: %w", err)
	}

	result := make(map[string]lessonPlanWordRelationship, len(document.Items))
	for _, item := range document.Items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		result[id] = lessonPlanWordRelationship{
			ID:         id,
			Type:       strings.TrimSpace(item.Type),
			Target:     strings.TrimSpace(item.Target),
			TargetMode: strings.TrimSpace(item.TargetMode),
		}
	}
	return result, nil
}

func parseLessonPlanWordDocumentXML(
	data []byte,
	relationships map[string]lessonPlanWordRelationship,
	warnings *lessonPlanWordWarningCollector,
) (
	LessonPlanWordPreviewDocument,
	LessonPlanWordPreviewMetrics,
	map[string]bool,
	error,
) {
	document := LessonPlanWordPreviewDocument{
		SchemaVersion: models.LessonPlanWordStructureSchemaVersion,
		ParserVersion: lessonPlanWordParserVersion,
		SourceFormat:  "docx",
		Flow:          make([]LessonPlanWordPreviewFlowItem, 0),
		Blocks:        make([]LessonPlanWordPreviewBlock, 0),
		Tables:        make([]LessonPlanWordPreviewTable, 0),
		Media:         make([]LessonPlanWordPreviewMedia, 0),
		Formulas:      make([]LessonPlanWordPreviewFormula, 0),
	}
	metrics := LessonPlanWordPreviewMetrics{}
	referencedMedia := make(map[string]bool)

	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = true

	tableStack := make([]lessonPlanWordTableParseContext, 0)
	var paragraph *lessonPlanWordParagraphBuilder
	var run *lessonPlanWordRunBuilder
	paragraphCounter := 0
	formulaCounter := 0
	formulaDepth := 0
	deletedDepth := 0
	objectDepth := 0
	captureMode := ""
	var formulaText strings.Builder

	flushRunText := func() {
		if paragraph == nil || run == nil || run.text.Len() == 0 {
			return
		}
		text := run.text.String()
		paragraph.runs = append(paragraph.runs, LessonPlanWordPreviewRun{
			Kind:          "text",
			Text:          text,
			Bold:          run.bold,
			Italic:        run.italic,
			Underline:     run.underline,
			VerticalAlign: run.verticalAlign,
		})
		run.text.Reset()
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return document, metrics, referencedMedia, fmt.Errorf(
				"解析Word正文XML失败: %w",
				err,
			)
		}

		switch item := token.(type) {
		case xml.StartElement:
			switch item.Name.Local {
			case "del":
				deletedDepth++

			case "tbl":
				parentTableIndex := -1
				parentRowIndex := -1
				parentCellIndex := -1
				nested := len(tableStack) > 0

				if nested {
					parent := tableStack[len(tableStack)-1]
					parentTableIndex = parent.tableIndex
					parentRowIndex = parent.rowIndex
					parentCellIndex = parent.cellIndex
					metrics.NestedTableCount++
					warnings.add(
						"nested_table",
						"文档包含嵌套表格；原文件会保留，网页编辑按内容块展示",
						1,
					)
				}

				tableIndex := len(document.Tables)
				document.Tables = append(document.Tables, LessonPlanWordPreviewTable{
					Index:            tableIndex,
					Nested:           nested,
					ParentTableIndex: parentTableIndex,
					ParentRowIndex:   parentRowIndex,
					ParentCellIndex:  parentCellIndex,
					GridWidths:       make([]int, 0),
					Rows:             make([]LessonPlanWordPreviewTableRow, 0),
				})

				if nested {
					parentCell := currentLessonPlanWordTableCell(
						&document,
						tableStack,
					)
					if parentCell != nil {
						parentCell.NestedTableIndices = append(
							parentCell.NestedTableIndices,
							tableIndex,
						)
					}
				} else {
					document.Flow = append(document.Flow, LessonPlanWordPreviewFlowItem{
						Kind:       "table",
						TableIndex: tableIndex,
					})
				}

				tableStack = append(tableStack, lessonPlanWordTableParseContext{
					tableIndex: tableIndex,
					rowIndex:   -1,
					cellIndex:  -1,
				})

			case "tr":
				if len(tableStack) == 0 {
					continue
				}
				context := &tableStack[len(tableStack)-1]
				table := &document.Tables[context.tableIndex]
				rowIndex := len(table.Rows)
				table.Rows = append(table.Rows, LessonPlanWordPreviewTableRow{
					Index: rowIndex,
					Cells: make([]LessonPlanWordPreviewTableCell, 0),
				})
				context.rowIndex = rowIndex
				context.cellIndex = -1

			case "tc":
				if len(tableStack) == 0 {
					continue
				}
				context := &tableStack[len(tableStack)-1]
				if context.rowIndex < 0 {
					continue
				}
				row := &document.Tables[context.tableIndex].Rows[context.rowIndex]
				cellIndex := len(row.Cells)
				row.Cells = append(row.Cells, LessonPlanWordPreviewTableCell{
					Index:    cellIndex,
					GridSpan: 1,
					BlockIDs: make([]string, 0),
				})
				context.cellIndex = cellIndex

			case "gridCol":
				if len(tableStack) == 0 {
					continue
				}
				width, _ := strconv.Atoi(xmlAttrValue(item, "w"))
				context := tableStack[len(tableStack)-1]
				document.Tables[context.tableIndex].GridWidths = append(
					document.Tables[context.tableIndex].GridWidths,
					width,
				)

			case "gridSpan":
				cell := currentLessonPlanWordTableCell(&document, tableStack)
				if cell == nil {
					continue
				}
				span, _ := strconv.Atoi(xmlAttrValue(item, "val"))
				if span > 1 {
					cell.GridSpan = span
					metrics.MergedCellCount++
				}

			case "vMerge":
				cell := currentLessonPlanWordTableCell(&document, tableStack)
				if cell == nil {
					continue
				}
				value := strings.TrimSpace(xmlAttrValue(item, "val"))
				if value == "" {
					value = "continue"
				}
				cell.VerticalMerge = value
				metrics.MergedCellCount++

			case "tcW":
				cell := currentLessonPlanWordTableCell(&document, tableStack)
				if cell == nil {
					continue
				}
				width, _ := strconv.Atoi(xmlAttrValue(item, "w"))
				cell.WidthTwips = width

			case "p":
				paragraphCounter++
				paragraph = &lessonPlanWordParagraphBuilder{
					paragraphIndex: paragraphCounter,
					tableIndex:     -1,
					rowIndex:       -1,
					cellIndex:      -1,
					runs:           make([]LessonPlanWordPreviewRun, 0),
				}
				if len(tableStack) > 0 {
					context := tableStack[len(tableStack)-1]
					paragraph.tableIndex = context.tableIndex
					paragraph.rowIndex = context.rowIndex
					paragraph.cellIndex = context.cellIndex
				}

			case "r":
				run = &lessonPlanWordRunBuilder{}

			case "b":
				if run != nil {
					run.bold = xmlOnOffValue(item)
				}

			case "i":
				if run != nil {
					run.italic = xmlOnOffValue(item)
				}

			case "u":
				if run != nil {
					value := strings.ToLower(strings.TrimSpace(xmlAttrValue(item, "val")))
					run.underline = value != "none" && value != "0" && value != "false"
				}

			case "vertAlign":
				if run != nil {
					value := strings.ToLower(strings.TrimSpace(xmlAttrValue(item, "val")))
					switch value {
					case "superscript":
						run.verticalAlign = "superscript"
					case "subscript":
						run.verticalAlign = "subscript"
					}
				}

			case "t":
				if deletedDepth > 0 {
					captureMode = "deleted"
				} else if formulaDepth > 0 {
					captureMode = "formula"
				} else {
					captureMode = "text"
				}

			case "tab":
				if deletedDepth == 0 && run != nil {
					run.text.WriteByte('\t')
				}

			case "br", "cr":
				if deletedDepth == 0 && run != nil {
					run.text.WriteByte('\n')
				}

			case "blip", "imagedata":
				if deletedDepth > 0 || paragraph == nil {
					continue
				}
				flushRunText()
				relationshipID := strings.TrimSpace(xmlAttrValue(item, "embed"))
				if relationshipID == "" {
					relationshipID = strings.TrimSpace(xmlAttrValue(item, "id"))
				}
				if relationshipID == "" {
					relationshipID = strings.TrimSpace(xmlAttrValue(item, "link"))
				}
				if relationshipID == "" {
					continue
				}
				referencedMedia[relationshipID] = true
				mediaTarget := ""
				if relationship, ok := relationships[relationshipID]; ok {
					mediaTarget = normalizeLessonPlanWordRelationshipTarget(relationship.Target)
				}
				paragraph.runs = append(paragraph.runs, LessonPlanWordPreviewRun{
					Kind:           "image",
					RelationshipID: relationshipID,
					MediaTarget:    mediaTarget,
				})
				metrics.ImageCount++

			case "oMath", "oMathPara":
				if formulaDepth == 0 {
					flushRunText()
					formulaText.Reset()
				}
				formulaDepth++

			case "object":
				objectDepth++
				if objectDepth == 1 {
					metrics.UnsupportedObjectCount++
					warnings.add(
						"embedded_object",
						"文档包含OLE或旧版嵌入对象；原文件会保留，第一版不直接编辑该对象",
						1,
					)
				}
			}

		case xml.CharData:
			switch captureMode {
			case "text":
				if run != nil {
					run.text.Write([]byte(item))
				}
			case "formula":
				formulaText.Write([]byte(item))
			}

		case xml.EndElement:
			switch item.Name.Local {
			case "t":
				captureMode = ""

			case "r":
				flushRunText()
				run = nil

			case "oMath", "oMathPara":
				if formulaDepth > 0 {
					formulaDepth--
				}
				if formulaDepth == 0 && paragraph != nil {
					formulaCounter++
					formulaID := fmt.Sprintf("formula-%04d", formulaCounter)
					text := strings.TrimSpace(formulaText.String())
					if text == "" {
						text = "Word公式"
					}
					paragraph.runs = append(paragraph.runs, LessonPlanWordPreviewRun{
						Kind:      "formula",
						Text:      text,
						FormulaID: formulaID,
					})
					formulaText.Reset()
				}

			case "p":
				if paragraph != nil {
					block := finalizeLessonPlanWordParagraph(
						paragraph,
						&document,
						&metrics,
					)
					if block != nil {
						document.Blocks = append(document.Blocks, *block)
						if block.TableIndex < 0 {
							document.Flow = append(document.Flow, LessonPlanWordPreviewFlowItem{
								Kind:    "block",
								BlockID: block.ID,
							})
						}

						for _, previewRun := range block.Runs {
							if previewRun.Kind == "formula" {
								document.Formulas = append(document.Formulas, LessonPlanWordPreviewFormula{
									ID:      previewRun.FormulaID,
									BlockID: block.ID,
									Text:    previewRun.Text,
								})
							}
						}
					}
				}
				paragraph = nil

			case "tc":
				if len(tableStack) > 0 {
					tableStack[len(tableStack)-1].cellIndex = -1
				}

			case "tr":
				if len(tableStack) > 0 {
					tableStack[len(tableStack)-1].rowIndex = -1
					tableStack[len(tableStack)-1].cellIndex = -1
				}

			case "tbl":
				if len(tableStack) > 0 {
					tableStack = tableStack[:len(tableStack)-1]
				}

			case "object":
				if objectDepth > 0 {
					objectDepth--
				}

			case "del":
				if deletedDepth > 0 {
					deletedDepth--
				}
			}
		}
	}

	return document, metrics, referencedMedia, nil
}
