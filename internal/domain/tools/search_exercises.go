package tools

import (
	"context"
	"fmt"

	"github.com/lumiforge/coach_chuck_ai/internal/domain/entities"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

var allowedBodyParts = map[string]struct{}{
	"abs":                     {},
	"arms":                    {},
	"back":                    {},
	"butt/hips":               {},
	"chest":                   {},
	"full body/integrated":    {},
	"legs - calves and shins": {},
	"legs - thighs":           {},
	"neck":                    {},
	"shoulders":               {},
}

var allowedEquipment = map[string]struct{}{
	"barbell":                        {},
	"bench":                          {},
	"bosu trainer":                   {},
	"cones":                          {},
	"dumbbells":                      {},
	"heavy ropes":                    {},
	"hurdles":                        {},
	"kettlebells":                    {},
	"ladder":                         {},
	"medicine ball":                  {},
	"no equipment":                   {},
	"pull up bar":                    {},
	"raised platform/box":            {},
	"resistance bands/cables":        {},
	"stability ball":                 {},
	"trx":                            {},
	"weight machines / selectorized": {},
}

var allowedDifficulty = map[string]struct{}{
	"beginner":     {},
	"intermediate": {},
	"advanced":     {},
}

type searchExercisesArgs struct {
	BodyPartsAny       []string `json:"body_parts_any,omitempty" jsonschema:"Optional. Match exercises that target at least one of these body parts. Allowed values: abs, arms, back, butt/hips, chest, full body/integrated, legs - calves and shins, legs - thighs, neck, shoulders."`
	EquipmentAny       []string `json:"equipment_any,omitempty" jsonschema:"Optional. Match exercises that require at least one of these equipment items. Allowed values: barbell, bench, bosu trainer, cones, dumbbells, heavy ropes, hurdles, kettlebells, ladder, medicine ball, no equipment, pull up bar, raised platform/box, resistance bands/cables, stability ball, trx, weight machines / selectorized."`
	DifficultyAny      []string `json:"difficulty_any,omitempty" jsonschema:"Optional. Allowed difficulty levels: beginner, intermediate, advanced."`
	ExcludeExerciseIDs []int64  `json:"exclude_exercise_ids,omitempty" jsonschema:"Optional. Exercise IDs to exclude from the results."`
	Limit              int      `json:"limit,omitempty" jsonschema:"Optional. Maximum number of exercises to return. Default is 20. Maximum is 50."`
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
type searchExerciseRepository interface {
	SearchExercises(ctx context.Context, filter entities.ExerciseSearchFilter) (entities.ExerciseSearchResult, error)
}

type searchExercisesTool struct {
	repo searchExerciseRepository
}

func NewSearchExercisesTool(repo searchExerciseRepository) (tool.Tool, error) {
	handler := &searchExercisesTool{repo: repo}

	return functiontool.New(
		functiontool.Config{
			Name:        "search_exercises",
			Description: "Finds exercises in the exercise catalog using optional filters for body parts, equipment, difficulty, and excluded exercise IDs. Use this tool to retrieve candidate exercises for a workout plan. Do not use it to build the full workout program directly.",
		},
		handler.searchExercises,
	)
}

func (t *searchExercisesTool) searchExercises(ctx tool.Context, args searchExercisesArgs) (searchExercisesResult, error) {
	if err := validateAllowedStrings(args.BodyPartsAny, allowedBodyParts, "body_parts_any"); err != nil {
		return searchExercisesResult{
			Status: "error",
			Error:  err.Error(),
		}, nil
	}

	if err := validateAllowedStrings(args.EquipmentAny, allowedEquipment, "equipment_any"); err != nil {
		return searchExercisesResult{
			Status: "error",
			Error:  err.Error(),
		}, nil
	}

	if err := validateAllowedStrings(args.DifficultyAny, allowedDifficulty, "difficulty_any"); err != nil {
		return searchExercisesResult{
			Status: "error",
			Error:  err.Error(),
		}, nil
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	offset := args.Offset
	if offset < 0 {
		offset = 0
	}

	result, err := t.repo.SearchExercises(ctx, entities.ExerciseSearchFilter{
		BodyPartsAny:       args.BodyPartsAny,
		EquipmentAny:       args.EquipmentAny,
		DifficultyAny:      args.DifficultyAny,
		ExcludeExerciseIDs: args.ExcludeExerciseIDs,
		Limit:              limit,
		Offset:             offset,
	})
	if err != nil {
		return searchExercisesResult{
			Status: "error",
			Error:  err.Error(),
		}, nil
	}

	items := make([]exerciseSummary, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, exerciseSummary{
			ID:         item.ID,
			Name:       item.Name,
			Difficulty: item.Difficulty,
			BodyParts:  item.BodyParts,
			Equipment:  item.Equipment,
		})
	}

	return searchExercisesResult{
		Status:  "success",
		Items:   items,
		Total:   result.Total,
		HasMore: offset+len(items) < result.Total,
	}, nil
}

func validateAllowedStrings(values []string, allowed map[string]struct{}, fieldName string) error {
	for _, value := range values {
		if _, ok := allowed[value]; !ok {
			return fmt.Errorf("invalid %s value: %q", fieldName, value)
		}
	}
	return nil
}
