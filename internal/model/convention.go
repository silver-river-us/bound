package model

import (
	"strings"
	"unicode"
)

func ConventionalFolder(name string) string {
	runes := []rune(name)
	var result []rune
	for index, character := range runes {
		if character == '_' || character == '-' {
			if len(result) > 0 && result[len(result)-1] != '_' {
				result = append(result, '_')
			}
			continue
		}
		if unicode.IsUpper(character) && len(result) > 0 && result[len(result)-1] != '_' {
			previousIsLower := unicode.IsLower(runes[index-1]) || unicode.IsDigit(runes[index-1])
			nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
			if previousIsLower || nextIsLower {
				result = append(result, '_')
			}
		}
		result = append(result, unicode.ToLower(character))
	}
	return strings.Trim(string(result), "_")
}

func ConventionalEntrypoint(name string) string {
	return strings.ReplaceAll(ConventionalFolder(name), "_", "-")
}
