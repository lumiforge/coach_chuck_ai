package a2ui

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func NormalizeJSONL(raw string) (string, error) {
	messages, err := DecodeJSONL(raw)
	if err != nil {
		return "", err
	}

	out := make([]string, 0, len(messages))
	for _, msg := range messages {
		normalized := NormalizeMessage(msg)
		b, err := json.Marshal(normalized)
		if err != nil {
			return "", err
		}
		out = append(out, string(b))
	}

	return strings.Join(out, "\n"), nil
}

func DecodeJSONL(raw string) ([]map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("no JSON objects found")
	}

	if strings.HasPrefix(raw, "[") {
		var arr []map[string]any
		if err := json.Unmarshal([]byte(raw), &arr); err != nil {
			return nil, err
		}
		if len(arr) == 0 {
			return nil, fmt.Errorf("no JSON objects found")
		}
		return arr, nil
	}

	dec := json.NewDecoder(strings.NewReader(raw))

	out := make([]map[string]any, 0, 4)

	for {
		var msg map[string]any
		err := dec.Decode(&msg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no JSON objects found")
	}

	return out, nil
}

func NormalizeMessage(msg map[string]any) map[string]any {
	msg = convertTypedEnvelope(msg)

	return msg
}

func convertTypedEnvelope(msg map[string]any) map[string]any {

	if hasEnvelope(msg) {
		return msg
	}
	rawType, ok := msg["type"].(string)
	if !ok || strings.TrimSpace(rawType) == "" {
		return msg
	}

	out := make(map[string]any, 2)
	payload := make(map[string]any, len(msg))
	for k, v := range msg {
		switch k {
		case "type":
			continue
		case "version":
			out[k] = v
		default:
			payload[k] = v
		}
	}

	switch rawType {
	case "createSurface", "updateComponents", "updateDataModel", "deleteSurface":
		out[rawType] = payload
	default:
		for k, v := range payload {
			out[k] = v
		}
	}

	return out
}

func hasEnvelope(msg map[string]any) bool {
	if msg == nil {
		return false
	}
	_, hasCreateSurface := msg["createSurface"]
	_, hasUpdateComponents := msg["updateComponents"]
	_, hasUpdateDataModel := msg["updateDataModel"]
	_, hasDeleteSurface := msg["deleteSurface"]

	return hasCreateSurface || hasUpdateComponents || hasUpdateDataModel || hasDeleteSurface
}
