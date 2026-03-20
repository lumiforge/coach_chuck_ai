package a2a

import (
	"encoding/gob"
)

const (
	appName          = "coach_agent"
	a2uiExtensionURI = "https://a2ui.org/a2a-extension/a2ui/v0.9"
	a2uiMimeType     = "application/json+a2ui"

	a2aUserID = "a2a-user"
)

func init() {
	gob.Register(map[string]any{})
	gob.Register([]map[string]any{})
	gob.Register([]any{})
}
