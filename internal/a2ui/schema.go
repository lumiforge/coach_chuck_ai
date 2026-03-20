package a2ui

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
)

var (
	workoutSchemaOnce sync.Once
	workoutSchema     *jsonschema.Resolved
	workoutSchemaErr  error
)

func WorkoutComponentSchema() (*jsonschema.Resolved, error) {
	workoutSchemaOnce.Do(func() {
		var catalogDoc map[string]any
		if err := json.Unmarshal(WorkoutCatalogJSON, &catalogDoc); err != nil {
			workoutSchemaErr = fmt.Errorf("unmarshal workout catalog: %w", err)
			return
		}

		rawComponents, ok := catalogDoc["components"].(map[string]any)
		if !ok {
			workoutSchemaErr = fmt.Errorf("workout catalog: components object is required")
			return
		}

		rawWorkoutSchema, ok := rawComponents["Workout"].(map[string]any)
		if !ok {
			workoutSchemaErr = fmt.Errorf("workout catalog: Workout component schema is required")
			return
		}

		root := map[string]any{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"$id":     WorkoutCatalogID,
			"$defs":   catalogDoc["$defs"],
			"allOf":   []any{rawWorkoutSchema},
		}

		schemaBytes, err := json.Marshal(root)
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

func ValidateWorkoutComponent(component map[string]any) error {
	rs, err := WorkoutComponentSchema()
	if err != nil {
		return err
	}
	return rs.Validate(component)
}
