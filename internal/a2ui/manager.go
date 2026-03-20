package a2ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type SchemaManager struct {
	selectedCatalog SelectedCatalog
}

type SelectedCatalog struct {
	CatalogSchema map[string]any
	Validator     *CatalogValidator
}

type CatalogValidator struct{}

func NewSchemaManager() (*SchemaManager, error) {
	var catalog map[string]any
	if err := json.Unmarshal(WorkoutCatalogJSON, &catalog); err != nil {
		return nil, fmt.Errorf("unmarshal workout catalog: %w", err)
	}

	return &SchemaManager{
		selectedCatalog: SelectedCatalog{
			CatalogSchema: catalog,
			Validator:     &CatalogValidator{},
		},
	}, nil
}

func (m *SchemaManager) GetSelectedCatalog() SelectedCatalog {
	return m.selectedCatalog
}

func (m *SchemaManager) GenerateSystemPrompt(
	roleDescription string,
	uiDescription string,
	includeSchema bool,
	includeExamples bool,
	validateExamples bool,
) (string, error) {
	var b strings.Builder

	if strings.TrimSpace(roleDescription) != "" {
		b.WriteString(strings.TrimSpace(roleDescription))
		b.WriteString("\n\n")
	}

	if strings.TrimSpace(uiDescription) != "" {
		b.WriteString(strings.TrimSpace(uiDescription))
		b.WriteString("\n\n")
	}

	b.WriteString("Return valid A2UI responses only.\n")
	b.WriteString("Use A2UI envelope messages, not a top-level \"type\" field.\n")
	b.WriteString("For a new UI response, create the surface first and then update the UI.\n")
	b.WriteString("Do not output plain text outside the A2UI response.\n")
	b.WriteString("Do not use markdown fences.\n")

	if includeSchema {
		schemaBytes, err := json.MarshalIndent(m.selectedCatalog.CatalogSchema, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal catalog schema: %w", err)
		}
		b.WriteString("\nCatalog schema:\n")
		b.Write(schemaBytes)
		b.WriteString("\n")
	}

	if includeExamples {
		example := []map[string]any{
			{
				"version": "v0.9",
				"createSurface": map[string]any{
					"surfaceId":     "workout-surface",
					"catalogId":     WorkoutCatalogID,
					"sendDataModel": false,
				},
			},
			{
				"version": "v0.9",
				"updateComponents": map[string]any{
					"surfaceId": "workout-surface",
					"components": []map[string]any{
						{
							"id":        "root",
							"component": "Workout",
							"title":     "Workout",
							"blocks": []map[string]any{
								{
									"title": "Main block",
									"sets": []map[string]any{
										{
											"items": []map[string]any{
												{
													"type": "exercise",
													"name": "Push-Up",
													"reps": 10,
												},
												{
													"type":        "rest",
													"durationSec": 30,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		if validateExamples {
			exampleBytes, err := marshalJSONLines(example)
			if err != nil {
				return "", fmt.Errorf("marshal example jsonl: %w", err)
			}
			if _, issues, err := NormalizeAndValidateJSONL(exampleBytes); err != nil {
				return "", fmt.Errorf("validate example jsonl: %w", err)
			} else if len(issues) > 0 {
				return "", fmt.Errorf("example jsonl validation failed: %s", FormatValidationIssues(issues))
			}
		}

		exampleBytes, _ := marshalJSONLines(example)
		b.WriteString("\nExamples:\n")
		b.WriteString(exampleBytes)
		b.WriteString("\n")
	}

	return b.String(), nil
}

func (v *CatalogValidator) Validate(raw string) error {
	_, issues, err := NormalizeAndValidateJSONL(raw)
	if err != nil {
		return err
	}
	if len(issues) > 0 {
		return fmt.Errorf(FormatValidationIssues(issues))
	}
	return nil
}

func marshalJSONLines(messages []map[string]any) (string, error) {
	var buf bytes.Buffer
	for i, msg := range messages {
		b, err := json.Marshal(msg)
		if err != nil {
			return "", err
		}
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.Write(b)
	}
	return buf.String(), nil
}
