package tools

import (
	_ "embed"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

//go:embed sensitive_words.txt
var sensitiveWordsText string

var sensitiveWordsPattern = compileSensitiveWords(sensitiveWordsText)

func compileSensitiveWords(text string) *regexp.Regexp {
	words := make([]string, 0)
	for _, line := range strings.Split(text, "\n") {
		word := strings.TrimSpace(line)
		if word != "" && !strings.HasPrefix(word, "#") {
			words = append(words, word)
		}
	}
	if len(words) == 0 {
		return nil
	}
	// 优先匹配长词，避免短词先替换后暴露长词的剩余部分。
	sort.Slice(words, func(i, j int) bool { return len(words[i]) > len(words[j]) })
	for i := range words {
		words[i] = regexp.QuoteMeta(words[i])
	}
	return regexp.MustCompile("(?i)(?:" + strings.Join(words, "|") + ")")
}

// FilterSensitiveWords 将基础词表命中的每个 Unicode 字符替换为一个 *。
func FilterSensitiveWords(content string) string {
	if sensitiveWordsPattern == nil {
		return content
	}
	return sensitiveWordsPattern.ReplaceAllStringFunc(content, func(match string) string {
		return strings.Repeat("*", utf8.RuneCountInString(match))
	})
}
