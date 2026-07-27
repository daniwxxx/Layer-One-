package utils

import (
	"strings"
	"unicode"
)

func Truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func Sanitize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.Join(strings.Fields(s), " ")
}

func NormalizeText(s string) string {
	s = strings.ToLower(s)
	repl := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"à", "a", "è", "e", "ì", "i", "ò", "o", "ù", "u",
		"ä", "a", "ë", "e", "ï", "i", "ö", "o", "ü", "u",
		"â", "a", "ê", "e", "î", "i", "ô", "o", "û", "u",
		"ñ", "n", "ç", "c",
	)
	return repl.Replace(s)
}

func Tokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var b strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '#' || r == '@' {
			b.WriteRune(r)
		} else if b.Len() > 0 {
			tokens = append(tokens, b.String())
			b.Reset()
		}
	}
	if b.Len() > 0 {
		tokens = append(tokens, b.String())
	}
	return tokens
}
