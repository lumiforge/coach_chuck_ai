package utils

import "strings"

func NormalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		out = append(out, value)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func NormalizeInt64Slice(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}

	out := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))

	for _, value := range values {
		if value <= 0 {
			continue
		}

		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		out = append(out, value)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}
