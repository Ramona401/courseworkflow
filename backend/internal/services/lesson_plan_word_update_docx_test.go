package services

import (
	"bytes"
	"testing"
)

const lessonPlanWordPatchTestXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document
	xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
	<w:body>
		<w:p>
			<w:r><w:t>旧文本</w:t></w:r>
			<w:r><w:drawing><a:blip r:embed="rIdImage"/></w:drawing></w:r>
		</w:p>
		<w:sectPr/>
	</w:body>
</w:document>`

func TestPatchLessonPlanWordDocumentXMLPreservesImageRun(
	t *testing.T,
) {
	patched, err := patchLessonPlanWordDocumentXML(
		[]byte(lessonPlanWordPatchTestXML),
		map[int]lessonPlanWordParagraphPatch{
			1: {
				TextRuns: []string{"新文本"},
			},
		},
	)
	if err != nil {
		t.Fatalf("修改Word文字失败: %v", err)
	}

	for _, required := range [][]byte{
		[]byte("新文本"),
		[]byte(`r:embed="rIdImage"`),
		[]byte("<w:drawing>"),
	} {
		if !bytes.Contains(patched, required) {
			t.Fatalf("修改后Word正文缺少预期内容: %s", required)
		}
	}
	if bytes.Contains(patched, []byte("旧文本")) {
		t.Fatal("旧文字仍然存在")
	}
}

func TestPatchLessonPlanWordDocumentXMLDeletesOnlyRequestedImage(
	t *testing.T,
) {
	source := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<w:document
	xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
	<w:body>
		<w:p>
			<w:r><w:t>前</w:t></w:r>
			<w:r><w:drawing><a:blip r:embed="rId1"/></w:drawing></w:r>
			<w:r><w:t>中</w:t></w:r>
			<w:r><w:drawing><a:blip r:embed="rId2"/></w:drawing></w:r>
			<w:r><w:t>后</w:t></w:r>
		</w:p>
	</w:body>
</w:document>`)

	patched, err := patchLessonPlanWordDocumentXML(
		source,
		map[int]lessonPlanWordParagraphPatch{
			1: {
				DeleteRelationshipIDs: []string{"rId1"},
			},
		},
	)
	if err != nil {
		t.Fatalf("删除指定Word图片失败: %v", err)
	}

	if bytes.Contains(patched, []byte(`r:embed="rId1"`)) {
		t.Fatal("指定删除的图片关系仍然存在")
	}
	if !bytes.Contains(patched, []byte(`r:embed="rId2"`)) {
		t.Fatal("未删除的图片关系被误删")
	}
	for _, text := range [][]byte{
		[]byte("前"),
		[]byte("中"),
		[]byte("后"),
	} {
		if !bytes.Contains(patched, text) {
			t.Fatalf("删除图片时误删文字: %s", text)
		}
	}
}

func TestPatchLessonPlanWordDocumentXMLEscapesReplacementText(
	t *testing.T,
) {
	patched, err := patchLessonPlanWordDocumentXML(
		[]byte(lessonPlanWordPatchTestXML),
		map[int]lessonPlanWordParagraphPatch{
			1: {
				TextRuns: []string{"A < B & C > D"},
			},
		},
	)
	if err != nil {
		t.Fatalf("写入含XML特殊字符的文字失败: %v", err)
	}

	if !bytes.Contains(
		patched,
		[]byte("A &lt; B &amp; C &gt; D"),
	) {
		t.Fatalf("XML特殊字符没有安全转义:\n%s", patched)
	}
}
