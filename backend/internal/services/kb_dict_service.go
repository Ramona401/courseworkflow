package services

import (
	"strings"

	"tedna/internal/models"
)

// ============================================================
// 知识库压缩入库系统 · 字典解码服务（课标）
//
// 职责（PRD §3.4 专利/商业秘密保护核心）：
//   把 KP 一行制索引原文解码为「人话」供审核员查看，
//   索引编码体系(KP ... | SJ:M|DP:2 [K]...)绝不展示给审核员，
//   只展示符号翻译后的中文卡片(SJ:M→数学、DP:2→理解应用层、[E]→学业要求)。
//
// 解码规则 1:1 照搬已入库的 dict_curriculum 字典 v1，不凭记忆造。
//
// fail-safe 原则：任何无法识别的符号/标签一律原样降级显示并标记
//   DecodeFailed=true，绝不 panic、绝不报错中断审核流程。
//
// 行结构（dict_curriculum 定义，逐字对齐）：
//   KP <kp_code> | SJ:<码>|SG:<码>|GR:<码>|DP:<码> [K]文本 [B]文本 [E]文本 ...
//   三段：
//     标识段：行首 "KP " + kp_code，到首个 " | " 前
//     符号段：首个 " | " 之后，到首个 " [" 之前，内部以 "|" 切分、每段 KEY:码
//     语义段：首个 "[" 起，按 "[" 切分、每段 [字母]后接文本直到下一个 "["
// ============================================================

// ---------- 符号码表（封闭，照搬 dict_curriculum 第三节） ----------

// kbDictSubject SJ 学科码→中文
var kbDictSubject = map[string]string{
	"M": "数学",
	"E": "英语",
	"C": "语文",
	"I": "信息科技",
}

// kbDictStage SG 学段码→中文
var kbDictStage = map[string]string{
	"1": "小学低段(1-2年级)",
	"2": "小学中段(3-4年级)",
	"3": "小学高段(5-6年级)",
	"4": "初中",
	"5": "高中",
}

// kbDictDepth DP 深度档→中文
var kbDictDepth = map[string]string{
	"1": "体验感知层",
	"2": "理解应用层",
	"3": "分析迁移层",
}

// ---------- 语义标签表（照搬 dict_curriculum 第四节） ----------

// kbDictLabel 语义标签字母→中文标签名
// 顺序无关，解析时按出现顺序保留。
var kbDictLabel = map[string]string{
	"K":  "知识点名称",
	"E":  "学业要求",
	"S":  "能力指向",
	"B":  "内容边界",
	"H":  "教学提示",
	"DM": "所属领域",
	"CC": "核心素养",
}

// ---------- 解码主函数 ----------

// DecodeCurriculumIndex 将一条 KP 课标索引行解码为人话卡片。
// 入参 line 为索引原文(可能含首尾空白)；返回解码后的结构，绝不返回 error。
// 任何无法识别的部分原样保留并置 DecodeFailed=true。
func DecodeCurriculumIndex(line string) *models.KBDecodedIndex {
	result := &models.KBDecodedIndex{
		Fields: []models.KBDecodedField{},
	}
	s := strings.TrimSpace(line)
	if s == "" {
		result.DecodeFailed = true
		return result
	}

	// ---- 1. 切出语义段起点（首个 " [" 或行首 "["） ----
	// 语义段以 "[" 开始；符号段+标识段在它之前。
	semStart := strings.Index(s, "[")
	var headPart, semPart string
	if semStart >= 0 {
		headPart = strings.TrimSpace(s[:semStart])
		semPart = s[semStart:]
	} else {
		// 没有语义段（异常索引），整行当头部，标记降级
		headPart = s
		semPart = ""
		result.DecodeFailed = true
	}

	// ---- 2. 头部拆「标识段」与「符号段」（以首个 "|" 为界） ----
	// 标识段：KP <kp_code>；符号段：SJ:..|SG:..|GR:..|DP:..
	barIdx := strings.Index(headPart, "|")
	var idPart, symPart string
	if barIdx >= 0 {
		idPart = strings.TrimSpace(headPart[:barIdx])
		symPart = strings.TrimSpace(headPart[barIdx+1:])
	} else {
		idPart = headPart
		symPart = ""
		result.DecodeFailed = true
	}

	// ---- 2a. 解析标识段取 kp_code ----
	// 形如 "KP MATH-G3-NA-001"，取 KP 之后的部分作编码
	idFields := strings.Fields(idPart)
	if len(idFields) >= 2 && idFields[0] == "KP" {
		result.KPCode = idFields[1]
	} else if len(idFields) == 1 {
		// 容错：只有编码没有 KP 前缀
		result.KPCode = idFields[0]
		result.DecodeFailed = true
	} else {
		result.DecodeFailed = true
	}

	// ---- 2b. 解析符号段 SJ/SG/GR/DP ----
	// 内部以 "|" 切分，每段 KEY:码
	if symPart != "" {
		for _, seg := range strings.Split(symPart, "|") {
			seg = strings.TrimSpace(seg)
			if seg == "" {
				continue
			}
			kv := strings.SplitN(seg, ":", 2)
			if len(kv) != 2 {
				result.DecodeFailed = true
				continue
			}
			key := strings.TrimSpace(kv[0])
			code := strings.TrimSpace(kv[1])
			switch key {
			case "SJ":
				if name, ok := kbDictSubject[code]; ok {
					result.SubjectName = name
				} else {
					result.SubjectName = code // 降级原样显示
					result.DecodeFailed = true
				}
			case "SG":
				if name, ok := kbDictStage[code]; ok {
					result.StageName = name
				} else {
					result.StageName = code
					result.DecodeFailed = true
				}
			case "GR":
				result.GradeName = decodeGrade(code)
			case "DP":
				if name, ok := kbDictDepth[code]; ok {
					result.DepthName = name
				} else {
					result.DepthName = code
					result.DecodeFailed = true
				}
			default:
				// 未知符号键，忽略但标记降级
				result.DecodeFailed = true
			}
		}
	}

	// ---- 3. 解析语义段 [字母]文本 ----
	// 按 "[" 切分，每段形如 "K]知识点名" → 标签K + 内容
	if semPart != "" {
		// 去掉开头的 "["，再按 "[" 切分，避免首元素为空
		trimmed := strings.TrimPrefix(semPart, "[")
		segs := strings.Split(trimmed, "[")
		for _, seg := range segs {
			seg = strings.TrimRight(seg, " ")
			if seg == "" {
				continue
			}
			// 形如 "K]内容" 或 "DM]内容"，以 "]" 分标签与内容
			rb := strings.Index(seg, "]")
			if rb < 0 {
				result.DecodeFailed = true
				continue
			}
			tag := strings.TrimSpace(seg[:rb])
			content := strings.TrimSpace(seg[rb+1:])
			labelName, ok := kbDictLabel[tag]
			if !ok {
				labelName = "[" + tag + "]" // 未知标签原样降级
				result.DecodeFailed = true
			}
			result.Fields = append(result.Fields, models.KBDecodedField{
				Label:   labelName,
				Tag:     tag,
				Content: content,
			})
		}
	}

	return result
}

// decodeGrade 解码 GR 年级码：1-12=对应年级，0=学段级
func decodeGrade(code string) string {
	switch code {
	case "0":
		return "学段级(不绑定具体年级)"
	case "1":
		return "一年级"
	case "2":
		return "二年级"
	case "3":
		return "三年级"
	case "4":
		return "四年级"
	case "5":
		return "五年级"
	case "6":
		return "六年级"
	case "7":
		return "七年级(初一)"
	case "8":
		return "八年级(初二)"
	case "9":
		return "九年级(初三)"
	case "10":
		return "高一"
	case "11":
		return "高二"
	case "12":
		return "高三"
	default:
		return code // 无法识别原样显示
	}
}
