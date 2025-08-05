package common

import (
	"encoding/json"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// ToInterfaceSlice converts a slice of strings to a slice of empty interfaces.
func ToInterfaceSlice[T any](slice []T) []interface{} {
	interfaceSlice := make([]interface{}, len(slice))
	for i, v := range slice {
		interfaceSlice[i] = v
	}
	return interfaceSlice
}

// []string -> string, " join, space separated
func StringSliceJoin(slice []string) string {
	result := make([]string, len(slice))
	for i, v := range slice {
		result[i] = `"` + v + `"`
	}
	return strings.Join(result, " ")
}

func GetAttrs[A, B any](extract func(A) B, attrs ...A) []B {
	result := make([]B, len(attrs))
	for i, attr := range attrs {
		result[i] = extract(attr)
	}
	return result
}

// Deduplicate removes duplicates from a slice based on a key function
// T: the type of elements in the slice
// K: the type of key used for deduplication
func Deduplicate[T any, K comparable](keyFunc func(T) K, items ...T) []T {
	seen := make(map[K]T)
	for _, item := range items {
		key := keyFunc(item)
		if _, exists := seen[key]; !exists {
			seen[key] = item
		}
	}
	return slices.Collect(maps.Values(seen))
}

// ParseLLMJsonResponse parses a JSON response from LLM, handling cases where JSON is wrapped in code blocks.
// This is useful when LLMs return responses like:
// ```json
// {"key": "value"}
// ```
// or regular JSON responses directly.
func ParseLLMJsonResponse(content string, target interface{}) error {
	// First, try to parse directly as JSON
	err := json.Unmarshal([]byte(content), target)
	if err == nil {
		return nil
	}

	// If direct parsing fails, try to extract JSON from code blocks
	re := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)```")
	matches := re.FindStringSubmatch(content)
	if len(matches) >= 2 {
		// Extract the JSON content within the code block
		jsonContent := strings.TrimSpace(matches[1])
		return json.Unmarshal([]byte(jsonContent), target)
	}

	// If no code block found, return the original error
	return err
}
