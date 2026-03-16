package tools

import "google.golang.org/adk/tool"

type searchExercisesArgs struct {
	BodyPartsAny       []string `json:"body_parts_any,omitempty" jsonschema:"Optional. Match exercises that target at least one of these body parts."`
	EquipmentAny       []string `json:"equipment_any,omitempty" jsonschema:"Optional. Match exercises that require at least one of these equipment items."`
	DifficultyAny      []string `json:"difficulty_any,omitempty" jsonschema:"Optional. Allowed difficulty levels: beginner, intermediate, advanced."`
	ExcludeExerciseIDs []int64  `json:"exclude_exercise_ids,omitempty" jsonschema:"Optional. Exercise IDs to exclude from the results."`
	Limit              int      `json:"limit,omitempty" jsonschema:"Optional. Maximum number of exercises to return. Default is 20."`
	Offset             int      `json:"offset,omitempty" jsonschema:"Optional. Number of matching exercises to skip for pagination. Default is 0."`
}

type exerciseSummary struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	Difficulty string   `json:"difficulty"`
	BodyParts  []string `json:"body_parts"`
	Equipment  []string `json:"equipment"`
}

type searchExercisesResult struct {
	Status  string            `json:"status"`
	Items   []exerciseSummary `json:"items,omitempty"`
	Total   int               `json:"total,omitempty"`
	HasMore bool              `json:"has_more,omitempty"`
	Error   string            `json:"error,omitempty"`
}

func searchExercises(ctx tool.Context, args searchExercisesArgs) (searchExercisesResult, error) {

}
