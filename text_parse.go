package main

import (
	//"fmt"
	"regexp"
	"unicode"
)



var (
	// 预编译中文正则
	reChinese = regexp.MustCompile(`[\x{4e00}-\x{9fff}\x{3400}-\x{4dbf}\x{20000}-\x{2a6df}]`)
	
	// 标点符号快速查找表
	punctMap = make(map[rune]struct{})
)

const (
	TypeChinese     = "chinese"
	TypeEnglish     = "english" // 包含空格
	TypeNumber      = "number"
	TypePunctuation = "punctuation"
	TypeOther       = "other"
)

func init() {
	// 初始化标点符号 Map，提升查找性能
	allPunctuation := `!"#$%&'()*+,-./:;<=>?@[\]^_\` + "`{|}~。，、；：？！…—·ˉ¨\"'\"\"々～‖∶＂＇｀｜〃〔〕〈〉《》「」『』．" + `“”±×÷≤≥≠≈∈∑∏∫√∞∆∇‰€£¥§¶†‡•◦‣※←→↑↓⇌⇒⇔∀∃∧∨¬⊕⊗－＊／＝＼＾＿｀｛｜｝～￥（）【】《》〈〉「」『』〚〛〘〙‹›«»※‽⁄⁰¹²³⁴⁵⁶⁷⁸⁹₀₁₂₃₄₅₆₇₈₉`
	for _, r := range allPunctuation {
		punctMap[r] = struct{}{}
	}
}

type TextSegment struct {
	Type    string
	Content string
}

// getCharType 判定字符类型
func getCharType(r rune) string {
	// 1. 中文判定 (正则)
	if reChinese.MatchString(string(r)) {
		return TypeChinese
	}
	
	// 2. 英文与空格判定 (采纳建议：空格归类为英文，方便处理词组)
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || unicode.IsSpace(r) {
		return TypeEnglish
	}
	
	// 3. 数字判定
	if r >= '0' && r <= '9' {
		return TypeNumber
	}
	
	// 4. 标点符号判定 (采纳建议：O(1) Map 查找 + 系统标点库)
	if _, ok := punctMap[r]; ok || unicode.IsPunct(r) || unicode.IsSymbol(r) {
		return TypePunctuation
	}

	return TypeOther
}

func SplitText(input string) []TextSegment {
	runes := []rune(input)
	if len(runes) == 0 {
		return nil
	}

	var segments []TextSegment
	currentStart := 0
	currentType := getCharType(runes[0])

	for i := 1; i < len(runes); i++ {
		t := getCharType(runes[i])
		if t != currentType {
			// 记录当前片段
			segments = append(segments, TextSegment{
				Type:    currentType,
				Content: string(runes[currentStart:i]),
			})
			currentType = t
			currentStart = i
		}
	}

	// 闭合最后一段
	segments = append(segments, TextSegment{
		Type:    currentType,
		Content: string(runes[currentStart:]),
	})

	return segments
}
