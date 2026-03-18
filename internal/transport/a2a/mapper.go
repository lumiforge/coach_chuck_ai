package a2a

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

func buildAgentInput(msg *a2a.Message) string {
	if msg == nil {
		return ""
	}

	var textParts []string

	for _, p := range msg.Parts {
		if p == nil {
			continue
		}

		if txt := p.Text(); strings.TrimSpace(txt) != "" {
			textParts = append(textParts, txt)
			continue
		}

		data := p.Data()
		obj, ok := data.(map[string]any)
		if !ok {
			continue
		}

		if version, _ := obj["version"].(string); version == "v0.9" {
			if rawErr, ok := obj["error"]; ok {
				errObj, ok := rawErr.(map[string]any)
				if !ok {
					textParts = append(textParts, fmt.Sprintf("CLIENT_A2UI_ERROR: %v", rawErr))
					continue
				}
				code := asString(errObj["code"])
				message := asString(errObj["message"])
				textParts = append(textParts, fmt.Sprintf("CLIENT_A2UI_ERROR: %s: %s", code, message))
				continue
			}
		}

		rawAction, ok := obj["userAction"]
		if !ok {
			continue
		}

		userAction, ok := rawAction.(map[string]any)
		if !ok {
			continue
		}

		actionName := asString(userAction["actionName"])
		ctxObj, _ := userAction["context"].(map[string]any)

		switch actionName {
		default:
			b, _ := json.Marshal(ctxObj)
			return fmt.Sprintf("User submitted an event: %s with data: %s", actionName, string(b))
		}
	}

	return strings.Join(textParts, "\n")
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}

func extractClientA2UIError(msg *a2a.Message) (string, bool) {
	if msg == nil {
		return "", false
	}

	for _, p := range msg.Parts {
		if p == nil {
			continue
		}

		data := p.Data()
		obj, ok := data.(map[string]any)
		if !ok {
			continue
		}

		version, _ := obj["version"].(string)
		if version != "v0.9" {
			continue
		}

		rawErr, ok := obj["error"]
		if !ok {
			continue
		}

		errObj, ok := rawErr.(map[string]any)
		if !ok {
			return fmt.Sprintf("%v", rawErr), true
		}

		code := strings.TrimSpace(asString(errObj["code"]))
		message := strings.TrimSpace(asString(errObj["message"]))

		switch {
		case code != "" && message != "":
			return fmt.Sprintf("%s: %s", code, message), true
		case message != "":
			return message, true
		default:
			return fmt.Sprintf("%v", errObj), true
		}
	}

	return "", false
}
