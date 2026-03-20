package a2ui

import "strings"

func ParseResponse(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	if !strings.Contains(raw, A2UIOpenTag) || !strings.Contains(raw, A2UICloseTag) {
		return []string{raw}
	}

	var parts []string
	start := 0

	for {
		openIdx := strings.Index(raw[start:], A2UIOpenTag)
		if openIdx < 0 {
			break
		}
		openIdx += start + len(A2UIOpenTag)

		closeIdx := strings.Index(raw[openIdx:], A2UICloseTag)
		if closeIdx < 0 {
			break
		}
		closeIdx += openIdx

		part := strings.TrimSpace(raw[openIdx:closeIdx])
		if part != "" {
			parts = append(parts, part)
		}
		start = closeIdx + len(A2UICloseTag)
	}

	return parts
}
