package coach

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
	cataloga2ui "github.com/lumiforge/coach_chuck_ai/internal/a2ui"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

var (
	workoutSchemaOnce sync.Once
	workoutSchema     *jsonschema.Resolved
	workoutSchemaErr  error
)

func flattenText(parts []*genai.Part) string {
	var sb strings.Builder
	for _, part := range parts {
		if part == nil {
			continue
		}
		if part.Text != "" {
			sb.WriteString(part.Text)
			sb.WriteString("\n")
		}
	}
	return strings.TrimSpace(sb.String())
}

func ValidateV09Output() llmagent.AfterModelCallback {
	return func(ctx agent.CallbackContext, llmResponse *model.LLMResponse, llmErr error) (*model.LLMResponse, error) {
		if llmErr != nil {
			return nil, llmErr
		}
		if llmResponse == nil || llmResponse.Content == nil || llmResponse.Partial {
			return nil, nil
		}

		hasFunctionCall := false
		hasFunctionResponse := false
		for _, p := range llmResponse.Content.Parts {
			if p == nil {
				continue
			}
			if p.FunctionCall != nil {
				hasFunctionCall = true
			}
			if p.FunctionResponse != nil {
				hasFunctionResponse = true
			}
		}

		if hasFunctionCall || hasFunctionResponse {
			return nil, nil
		}

		raw := flattenText(llmResponse.Content.Parts)
		if strings.TrimSpace(raw) == "" {
			return nil, fmt.Errorf("empty model response")
		}

		if err := validateA2UIV09JSONL(raw); err != nil {
			return nil, fmt.Errorf("invalid A2UI v0.9 output: %w", err)
		}

		return nil, nil
	}
}

func validateA2UIV09JSONL(raw string) error {
	dec := json.NewDecoder(strings.NewReader(raw))

	messageCount := 0
	seenCreateSurface := false
	seenUpdateComponents := false
	rootSeen := false

	for {
		var msg map[string]any
		err := dec.Decode(&msg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("invalid JSONL stream: %w", err)
		}

		messageCount++

		version, _ := msg["version"].(string)
		if version != "v0.9" {
			return fmt.Errorf("message %d: version must be v0.9", messageCount)
		}

		msgTypes := 0

		if cs, ok := msg["createSurface"]; ok {
			msgTypes++
			seenCreateSurface = true

			obj, ok := cs.(map[string]any)
			if !ok {
				return fmt.Errorf("message %d: createSurface must be an object", messageCount)
			}

			surfaceID, _ := obj["surfaceId"].(string)
			if strings.TrimSpace(surfaceID) == "" {
				return fmt.Errorf("message %d: createSurface.surfaceId is required", messageCount)
			}
			if surfaceID != "main" {
				return fmt.Errorf("message %d: createSurface.surfaceId must be main", messageCount)
			}

			catalogID, _ := obj["catalogId"].(string)
			if catalogID != "https://lumiforge.dev/a2ui/catalogs/workout/v1" {
				return fmt.Errorf(
					"message %d: createSurface.catalogId must be %s",
					messageCount,
					"https://lumiforge.dev/a2ui/catalogs/workout/v1",
				)
			}
		}

		if uc, ok := msg["updateComponents"]; ok {
			msgTypes++
			seenUpdateComponents = true

			obj, ok := uc.(map[string]any)
			if !ok {
				return fmt.Errorf("message %d: updateComponents must be an object", messageCount)
			}

			surfaceID, _ := obj["surfaceId"].(string)
			if strings.TrimSpace(surfaceID) == "" {
				return fmt.Errorf("message %d: updateComponents.surfaceId is required", messageCount)
			}
			if surfaceID != "main" {
				return fmt.Errorf("message %d: updateComponents.surfaceId must be main", messageCount)
			}

			components, ok := obj["components"].([]any)
			if !ok || len(components) == 0 {
				return fmt.Errorf("message %d: updateComponents.components must be a non-empty array", messageCount)
			}

			for _, c := range components {
				component, ok := c.(map[string]any)
				if !ok {
					return fmt.Errorf("message %d: each component must be an object", messageCount)
				}

				id, _ := component["id"].(string)
				if strings.TrimSpace(id) == "" {
					return fmt.Errorf("message %d: component.id is required", messageCount)
				}

				discriminator, _ := component["component"].(string)
				if strings.TrimSpace(discriminator) == "" {
					return fmt.Errorf("message %d: component.component is required", messageCount)
				}
				if discriminator != "Workout" {
					return fmt.Errorf("message %d: component.component must be Workout", messageCount)
				}

				if id == "root" {
					rootSeen = true

					if err := validateWorkoutRootComponent(component); err != nil {
						return fmt.Errorf("message %d: invalid root Workout component: %w", messageCount, err)
					}
				}
			}
		}

		if udm, ok := msg["updateDataModel"]; ok {
			msgTypes++

			obj, ok := udm.(map[string]any)
			if !ok {
				return fmt.Errorf("message %d: updateDataModel must be an object", messageCount)
			}

			surfaceID, _ := obj["surfaceId"].(string)
			if strings.TrimSpace(surfaceID) == "" {
				return fmt.Errorf("message %d: updateDataModel.surfaceId is required", messageCount)
			}
			if surfaceID != "main" {
				return fmt.Errorf("message %d: updateDataModel.surfaceId must be main", messageCount)
			}
		}

		if ds, ok := msg["deleteSurface"]; ok {
			msgTypes++
			if _, ok := ds.(map[string]any); !ok {
				return fmt.Errorf("message %d: deleteSurface must be an object", messageCount)
			}
		}

		if msgTypes != 1 {
			return fmt.Errorf("message %d: each JSON object must contain exactly one v0.9 message type", messageCount)
		}
	}

	if messageCount == 0 {
		return fmt.Errorf("no JSON messages found")
	}
	if !seenCreateSurface {
		return fmt.Errorf("missing createSurface message")
	}
	if !seenUpdateComponents {
		return fmt.Errorf("missing updateComponents message")
	}
	if !rootSeen {
		return fmt.Errorf("missing root component with id=root")
	}

	return nil
}

func validateWorkoutRootComponent(component map[string]any) error {
	rs, err := getWorkoutComponentSchema()
	if err != nil {
		return err
	}

	if err := rs.Validate(component); err != nil {
		return err
	}

	return nil
}

func getWorkoutComponentSchema() (*jsonschema.Resolved, error) {
	workoutSchemaOnce.Do(func() {
		var catalogDoc map[string]any
		if err := json.Unmarshal(cataloga2ui.WorkoutCatalogJSON, &catalogDoc); err != nil {
			workoutSchemaErr = fmt.Errorf("unmarshal workout catalog: %w", err)
			return
		}

		rawComponents, ok := catalogDoc["components"].(map[string]any)
		if !ok {
			workoutSchemaErr = fmt.Errorf("workout catalog: components object is required")
			return
		}

		rawWorkoutSchema, ok := rawComponents["Workout"]
		if !ok {
			workoutSchemaErr = fmt.Errorf("workout catalog: Workout component schema is required")
			return
		}

		schemaBytes, err := json.Marshal(rawWorkoutSchema)
		if err != nil {
			workoutSchemaErr = fmt.Errorf("marshal Workout component schema: %w", err)
			return
		}

		var schema jsonschema.Schema
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			workoutSchemaErr = fmt.Errorf("unmarshal Workout component schema: %w", err)
			return
		}

		workoutSchema, workoutSchemaErr = schema.Resolve(nil)
		if workoutSchemaErr != nil {
			workoutSchemaErr = fmt.Errorf("resolve Workout component schema: %w", workoutSchemaErr)
			return
		}
	})

	if workoutSchemaErr != nil {
		return nil, workoutSchemaErr
	}

	return workoutSchema, nil
}
