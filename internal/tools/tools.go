package tools

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
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

func CopyMap(m map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = v
	}
	return result
}

func Int32Ptr(i int32) *int32 {
	return &i
}

func IntOrStringPtr(s string) *intstr.IntOrString {
	return &intstr.IntOrString{Type: intstr.String, StrVal: s}
}

func BoolPtr(b bool) *bool {
	return &b
}
