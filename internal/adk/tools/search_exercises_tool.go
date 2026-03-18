package tools

import (
	"context"

	"github.com/lumiforge/coach_chuck_ai/internal/domain/entities"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

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

type SearchExercisesService interface {
	SearchExercises(ctx context.Context, input entities.SearchExercisesInput) (entities.SearchExercisesOutput, error)
}

type searchExercisesTool struct {
	service SearchExercisesService
}

func NewSearchExercisesTool(service SearchExercisesService) (tool.Tool, error) {
	handler := &searchExercisesTool{service: service}

	return functiontool.New(
		functiontool.Config{
			Name:        "search_exercises",
			Description: "Finds exercises in the exercise catalog using optional filters for body parts, equipment, difficulty, and excluded exercise IDs. Use this tool to retrieve candidate exercises for a workout plan. Do not use it to build the full workout program directly.",
		},
		handler.searchExercises,
	)
}

func (t *searchExercisesTool) searchExercises(ctx tool.Context, args searchExercisesArgs) (searchExercisesResult, error) {
	result, err := t.service.SearchExercises(ctx, entities.SearchExercisesInput{
		BodyPartsAny:       args.BodyPartsAny,
		EquipmentAny:       args.EquipmentAny,
		DifficultyAny:      args.DifficultyAny,
		ExcludeExerciseIDs: args.ExcludeExerciseIDs,
		Limit:              args.Limit,
		Offset:             args.Offset,
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
		HasMore: result.HasMore,
	}, nil
}
