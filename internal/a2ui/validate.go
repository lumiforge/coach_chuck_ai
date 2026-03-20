package a2ui

import (
	"fmt"
	"strings"
)

type ValidationIssue struct {
	SurfaceID string `json:"surfaceId"`
	Path      string `json:"path"`
	Message   string `json:"message"`
}

func (v ValidationIssue) ErrorEnvelope() map[string]any {
	errObj := map[string]any{
		"error": map[string]any{
			"code": "VALIDATION_FAILED",

			"message": v.Message,
		},
	}
	if strings.TrimSpace(v.SurfaceID) != "" {
		errObj["error"].(map[string]any)["surfaceId"] = v.SurfaceID
	}
	if strings.TrimSpace(v.Path) != "" {
		errObj["error"].(map[string]any)["path"] = v.Path
	}
	return errObj
}

func NormalizeAndValidateJSONL(raw string) (string, []ValidationIssue, error) {
	normalized, err := NormalizeJSONL(raw)
	if err != nil {
		return "", []ValidationIssue{
			{

				Path:    "/",
				Message: fmt.Sprintf("invalid JSONL stream: %v", err),
			},
		}, nil
	}

	issues, err := ValidateJSONL(normalized)
	if err != nil {
		return "", nil, err
	}

	return normalized, issues, nil
}

func ValidateJSONL(raw string) ([]ValidationIssue, error) {
	messages, err := DecodeJSONL(raw)
	if err != nil {
		return []ValidationIssue{
			{

				Path:    "/",
				Message: fmt.Sprintf("invalid JSONL stream: %v", err),
			},
		}, nil
	}

	issues := make([]ValidationIssue, 0, 8)

	messageCount := 0
	seenCreateSurface := false
	seenUpdateComponents := false
	rootSeen := false

	for _, msg := range messages {
		messageCount++

		version, _ := msg["version"].(string)
		if version != "v0.9" {
			issues = append(issues, ValidationIssue{

				Path:    fmt.Sprintf("/messages/%d/version", messageCount-1),
				Message: "version must be v0.9",
			})
		}

		msgTypes := 0

		if cs, ok := msg["createSurface"]; ok {
			msgTypes++
			seenCreateSurface = true

			obj, ok := cs.(map[string]any)
			if !ok {
				issues = append(issues, ValidationIssue{

					Path:    fmt.Sprintf("/messages/%d/createSurface", messageCount-1),
					Message: "createSurface must be an object",
				})
			} else {
				surfaceID, _ := obj["surfaceId"].(string)
				if surfaceID == "" {
					issues = append(issues, ValidationIssue{SurfaceID: surfaceID, Path: fmt.Sprintf("/messages/%d/createSurface/surfaceId", messageCount-1), Message: "createSurface.surfaceId is required"})
				}

				catalogID, _ := obj["catalogId"].(string)
				if catalogID == "" {
					issues = append(issues, ValidationIssue{SurfaceID: surfaceID, Path: fmt.Sprintf("/messages/%d/createSurface/catalogId", messageCount-1), Message: "createSurface.catalogId is required"})
				}
			}
		}

		if uc, ok := msg["updateComponents"]; ok {
			msgTypes++
			seenUpdateComponents = true

			obj, ok := uc.(map[string]any)
			if !ok {
				issues = append(issues, ValidationIssue{Path: fmt.Sprintf("/messages/%d/updateComponents", messageCount-1), Message: "updateComponents must be an object"})
			} else {
				surfaceID, _ := obj["surfaceId"].(string)
				if surfaceID == "" {
					issues = append(issues, ValidationIssue{SurfaceID: surfaceID, Path: fmt.Sprintf("/messages/%d/updateComponents/surfaceId", messageCount-1), Message: "updateComponents.surfaceId is required"})
				}

				components, ok := obj["components"].([]any)
				if !ok || len(components) == 0 {
					issues = append(issues, ValidationIssue{SurfaceID: surfaceID, Path: fmt.Sprintf("/messages/%d/updateComponents/components", messageCount-1), Message: "updateComponents.components must be a non-empty array"})
				} else {
					for i, c := range components {
						component, ok := c.(map[string]any)
						if !ok {
							issues = append(issues, ValidationIssue{SurfaceID: surfaceID, Path: fmt.Sprintf("/messages/%d/updateComponents/components/%d", messageCount-1, i), Message: "each component must be an object"})
							continue
						}

						id, _ := component["id"].(string)
						if strings.TrimSpace(id) == "" {
							issues = append(issues, ValidationIssue{SurfaceID: surfaceID, Path: fmt.Sprintf("/messages/%d/updateComponents/components/%d/id", messageCount-1, i), Message: "component.id is required"})
						}

						discriminator, _ := component["component"].(string)
						if strings.TrimSpace(discriminator) == "" {
							issues = append(issues, ValidationIssue{SurfaceID: surfaceID, Path: fmt.Sprintf("/messages/%d/updateComponents/components/%d/component", messageCount-1, i), Message: "component.component is required"})

						} else if discriminator != "Workout" {
							issues = append(issues, ValidationIssue{SurfaceID: surfaceID, Path: fmt.Sprintf("/messages/%d/updateComponents/components/%d/component", messageCount-1, i), Message: "component.component must be Workout"})
						}

						if id == "root" {
							rootSeen = true
							if err := ValidateWorkoutComponent(component); err != nil {
								issues = append(issues, ValidationIssue{
									SurfaceID: surfaceID,
									Path:      fmt.Sprintf("/messages/%d/updateComponents/components/%d", messageCount-1, i),
									Message:   fmt.Sprintf("invalid root Workout component: %v", err),
								})
							}
						}
					}
				}
			}
		}

		if udm, ok := msg["updateDataModel"]; ok {
			msgTypes++
			obj, ok := udm.(map[string]any)
			if !ok {
				issues = append(issues, ValidationIssue{Path: fmt.Sprintf("/messages/%d/updateDataModel", messageCount-1), Message: "updateDataModel must be an object"})
			} else {
				surfaceID, _ := obj["surfaceId"].(string)
				if surfaceID == "" {
					issues = append(issues, ValidationIssue{SurfaceID: surfaceID, Path: fmt.Sprintf("/messages/%d/updateDataModel/surfaceId", messageCount-1), Message: "updateDataModel.surfaceId is required"})
				}
			}
		}

		if ds, ok := msg["deleteSurface"]; ok {
			msgTypes++
			if _, ok := ds.(map[string]any); !ok {
				issues = append(issues, ValidationIssue{Path: fmt.Sprintf("/messages/%d/deleteSurface", messageCount-1), Message: "deleteSurface must be an object"})
			}
		}

		if msgTypes != 1 {
			issues = append(issues, ValidationIssue{Path: fmt.Sprintf("/messages/%d", messageCount-1), Message: "each JSON object must contain exactly one v0.9 message type"})
		}
	}

	if messageCount == 0 {
		issues = append(issues, ValidationIssue{Path: "/", Message: "no JSON messages found"})
	}
	if !seenCreateSurface {
		issues = append(issues, ValidationIssue{Message: "missing createSurface message"})
	}
	if !seenUpdateComponents {
		issues = append(issues, ValidationIssue{Message: "missing updateComponents message"})
	}
	if !rootSeen {
		issues = append(issues, ValidationIssue{Message: "missing root component with id=root"})
	}

	return issues, nil
}
