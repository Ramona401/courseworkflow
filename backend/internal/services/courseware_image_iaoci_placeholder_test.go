package services

import (
	"strings"
	"testing"
)

// TestCWImagePlaceholderStableIDAndFill 验证稳定ID补齐和精确填图。
func TestCWImagePlaceholderStableIDAndFill(
	t *testing.T,
) {
	sourceHTML := `
<div class="page">
	<div class="img-placeholder" data-placeholder-id="HERO" data-desc="封面主视觉">
		🖼️ 封面图片
	</div>
	<div class="card">
		<div class="img-placeholder" data-desc="实验装置">
			🖼️ 实验图片
		</div>
	</div>
</div>`

	normalizedHTML, slots, changed :=
		cwEnsureImagePlaceholderIDs(sourceHTML)

	if !changed {
		t.Fatal("第二个图片槽位没有ID，应当发生修改")
	}

	if len(slots) != 2 {
		t.Fatalf(
			"图片槽位数量错误：got=%d want=2",
			len(slots),
		)
	}

	if slots[0].PlaceholderID != "HERO" {
		t.Fatalf(
			"已有稳定ID被错误修改：%s",
			slots[0].PlaceholderID,
		)
	}

	if slots[1].PlaceholderID !=
		"IMG_SLOT_02" {
		t.Fatalf(
			"自动生成的稳定ID错误：%s",
			slots[1].PlaceholderID,
		)
	}

	if slots[0].Description !=
		"封面主视觉" {
		t.Fatalf(
			"第一个槽位描述错误：%s",
			slots[0].Description,
		)
	}

	if slots[1].Description !=
		"实验装置" {
		t.Fatalf(
			"第二个槽位描述错误：%s",
			slots[1].Description,
		)
	}

	if !strings.Contains(
		normalizedHTML,
		`data-placeholder-id="IMG_SLOT_02"`,
	) {
		t.Fatal("规范化HTML没有写入第二个稳定ID")
	}

	filledHTML, filled :=
		cwFillImagePlaceholder(
			normalizedHTML,
			"IMG_SLOT_02",
			"https://example.com/device.png",
			"实验装置",
		)

	if !filled {
		t.Fatal("未能按placeholder_id精确填图")
	}

	if !strings.Contains(
		filledHTML,
		`data-iaoci-slot="IMG_SLOT_02"`,
	) {
		t.Fatal("填入图片缺少IAOCI槽位绑定")
	}

	if !strings.Contains(
		filledHTML,
		`src="https://example.com/device.png"`,
	) {
		t.Fatal("填入图片URL错误")
	}

	if strings.Contains(
		filledHTML,
		"🖼️ 实验图片",
	) {
		t.Fatal("原占位内容没有被替换")
	}

	if !strings.Contains(
		filledHTML,
		"🖼️ 封面图片",
	) {
		t.Fatal("填第二个槽位时错误修改了第一个槽位")
	}
}

// TestCWImagePlaceholderHide 验证隐藏单个槽位不会影响其它槽位。
func TestCWImagePlaceholderHide(
	t *testing.T,
) {
	sourceHTML := `
<div>
	<div class="img-placeholder" data-placeholder-id="IMG_SLOT_01">第一张</div>
	<div class="img-placeholder" data-placeholder-id="IMG_SLOT_02">第二张</div>
</div>`

	hiddenHTML, hidden :=
		cwHideImagePlaceholder(
			sourceHTML,
			"IMG_SLOT_02",
			"generation_failed",
		)

	if !hidden {
		t.Fatal("未找到需要隐藏的图片槽位")
	}

	if !strings.Contains(
		hiddenHTML,
		`data-image-state="generation_failed"`,
	) {
		t.Fatal("隐藏状态没有写入HTML")
	}

	if !strings.Contains(
		hiddenHTML,
		`data-placeholder-id="IMG_SLOT_01"`,
	) {
		t.Fatal("隐藏第二个槽位时错误删除了第一个槽位")
	}
}

// TestCWImagePlaceholderDuplicateID 验证重复ID会被重新分配。
func TestCWImagePlaceholderDuplicateID(
	t *testing.T,
) {
	sourceHTML := `
<div>
	<div class="img-placeholder" data-placeholder-id="SAME">第一张</div>
	<div class="img-placeholder" data-placeholder-id="SAME">第二张</div>
</div>`

	normalizedHTML, slots, changed :=
		cwEnsureImagePlaceholderIDs(sourceHTML)

	if !changed {
		t.Fatal("重复ID应当触发规范化")
	}

	if len(slots) != 2 {
		t.Fatalf(
			"图片槽位数量错误：got=%d want=2",
			len(slots),
		)
	}

	if slots[0].PlaceholderID ==
		slots[1].PlaceholderID {
		t.Fatal("重复placeholder_id没有被消除")
	}

	if strings.Count(
		normalizedHTML,
		`data-placeholder-id="SAME"`,
	) != 1 {
		t.Fatal("规范化后仍存在重复ID")
	}
}
