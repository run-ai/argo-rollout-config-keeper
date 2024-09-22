package tools

import (
	"strings"
)

func ContainsString(slice []string, s string) (string, bool) {
	for _, item := range slice {
		if strings.Contains(item, s) {
			return item, true
		}
	}
	return "", false
}

func RemoveString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

func CreateMapFromStringList(list []string) map[string]bool {
	result := make(map[string]bool)
	for _, item := range list {
		result[item] = true
	}
	return result
}
