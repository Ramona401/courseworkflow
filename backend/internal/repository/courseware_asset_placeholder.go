package repository

// courseware_asset_placeholder.go — 课件资产占位键稳定化
//
// courseware_assets.placeholder_id数据库字段最长50字符。
// 漫画项目使用UUID组成的语义键可能超过该限制，因此在仓储边界统一转换：
//   - 50字符以内保持原值；
//   - 超长值保留可识别业务前缀；
//   - 追加原始完整值的SHA-256摘要前96位；
//   - 相同原值始终获得相同结果；
//   - 查询入口执行相同转换，调用方仍可使用原始长键。

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	cwAssetPlaceholderMaxRunes  = 50
	cwAssetPlaceholderHashBytes = 12
)

func normalizeCWAssetPlaceholderID(
	value string,
) string {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return ""
	}

	runes :=
		[]rune(value)

	if len(runes) <=
		cwAssetPlaceholderMaxRunes {
		return value
	}

	digest :=
		sha256.Sum256(
			[]byte(value),
		)

	suffix :=
		hex.EncodeToString(
			digest[:cwAssetPlaceholderHashBytes],
		)

	prefix :=
		cwAssetPlaceholderPrefix(
			value,
		)

	maxPrefixRunes :=
		cwAssetPlaceholderMaxRunes -
			1 -
			len([]rune(suffix))

	prefixRunes :=
		[]rune(prefix)

	if len(prefixRunes) >
		maxPrefixRunes {
		prefix =
			string(
				prefixRunes[:maxPrefixRunes],
			)
	}

	prefix =
		strings.Trim(
			prefix,
			" :-_",
		)

	if prefix == "" {
		prefix = "asset"
	}

	return prefix +
		":" +
		suffix
}

func cwAssetPlaceholderPrefix(
	value string,
) string {
	switch {
	case strings.HasPrefix(
		value,
		"comic-character-sheet:",
	):
		return "comic-character-sheet"

	case strings.HasPrefix(
		value,
		"comic-panel:",
	):
		return "comic-panel"
	}

	runes :=
		[]rune(value)

	if len(runes) > 20 {
		runes =
			runes[:20]
	}

	prefix :=
		strings.Trim(
			string(runes),
			" :-_",
		)

	if prefix == "" {
		return "asset"
	}

	return prefix
}
