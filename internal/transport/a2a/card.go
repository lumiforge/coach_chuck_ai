package a2a

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func serveAgentCard(port string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}

		host := r.Host
		if host == "" {
			host = "localhost:" + port
		}

		card := map[string]any{
			"name":               "coach_agent",
			"description":        "Coach agent with A2UI v0.9 output over A2A DataPart",
			"url":                fmt.Sprintf("%s://%s/a2a/invoke", scheme, host),
			"version":            "0.1.0",
			"protocolVersion":    "0.3.0",
			"defaultInputModes":  []string{"text/plain"},
			"defaultOutputModes": []string{a2uiMimeType},
			"capabilities": map[string]any{
				"streaming": true,
				"extensions": []map[string]any{
					{
						"uri":         a2uiExtensionURI,
						"description": "Ability to render A2UI",
						"required":    false,
						"params": map[string]any{
							"supportedCatalogIds":   []string{basicCatalogV09ID},
							"acceptsInlineCatalogs": false,
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(card)
	}
}
