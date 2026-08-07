package repository

import (
	"fmt"
	"strings"
	"testing"
)

func TestNormalizeCWAssetPlaceholderID(
	t *testing.T,
) {
	shortValue :=
		"image:P01"

	if result :=
		normalizeCWAssetPlaceholderID(
			shortValue,
		); result != shortValue {
		t.Fatalf(
			"短占位键不应变化，实际为%q",
			result,
		)
	}

	projectID :=
		"123e4567-e89b-12d3-a456-426614174000"

	characterSheet :=
		"comic-character-sheet:" +
			projectID

	panel :=
		fmt.Sprintf(
			"comic-panel:%s:%02d",
			projectID,
			1,
		)

	values :=
		[]string{
			characterSheet,
			panel,
			strings.Repeat(
				"中文占位键",
				20,
			),
		}

	for _, value := range values {
		first :=
			normalizeCWAssetPlaceholderID(
				value,
			)

		second :=
			normalizeCWAssetPlaceholderID(
				value,
			)

		if first != second {
			t.Fatalf(
				"相同长键必须稳定映射：%q与%q",
				first,
				second,
			)
		}

		if len([]rune(first)) >
			cwAssetPlaceholderMaxRunes {
			t.Fatalf(
				"映射结果仍超过50字符：%q",
				first,
			)
		}

		if first == value {
			t.Fatalf(
				"超长键没有被转换：%q",
				value,
			)
		}
	}

	anotherPanel :=
		normalizeCWAssetPlaceholderID(
			fmt.Sprintf(
				"comic-panel:%s:%02d",
				projectID,
				2,
			),
		)

	firstPanel :=
		normalizeCWAssetPlaceholderID(
			panel,
		)

	if anotherPanel ==
		firstPanel {
		t.Fatal(
			"不同漫画格不能映射成相同占位键",
		)
	}

	if !strings.HasPrefix(
		normalizeCWAssetPlaceholderID(
			characterSheet,
		),
		"comic-character-sheet:",
	) {
		t.Fatal(
			"人物设定图短键应保留业务前缀",
		)
	}

	if !strings.HasPrefix(
		firstPanel,
		"comic-panel:",
	) {
		t.Fatal(
			"漫画格短键应保留业务前缀",
		)
	}
}
