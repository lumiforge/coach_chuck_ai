package tools

import (
	"github.com/lumiforge/coach_chuck_ai/internal/domain/entities"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type getExerciseDetailsArgs struct {
	ExerciseIDs []int64 `json:"exercise_ids" jsonschema:"Required. Exercise IDs for which to load detailed exercise information."`
}

type exerciseDetails struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Difficulty  string   `json:"difficulty"`
	Description string   `json:"description"`
	BodyParts   []string `json:"body_parts"`
	Equipment   []string `json:"equipment"`
}

type getExerciseDetailsResult struct {
	Status string            `json:"status"`
	Items  []exerciseDetails `json:"items,omitempty"`
	Error  string            `json:"error,omitempty"`
}

type exerciseDetailsRepository interface {
	GetExerciseDetails(ctx tool.Context, exerciseIDs []int64) (entities.ExerciseDetailsResult, error)
}

type getExerciseDetailsTool struct {
	repo exerciseDetailsRepository
}

func newGetExerciseDetailsTool(repo exerciseDetailsRepository) (tool.Tool, error) {
	handler := &getExerciseDetailsTool{repo: repo}

	return functiontool.New(
		functiontool.Config{
			Name:        "get_exercise_details",
			Description: "Loads detailed exercise information for specific exercise IDs. Use this tool after selecting candidate exercises to retrieve their full descriptions and metadata.",
		},
		handler.getExerciseDetails,
	)
}

func (t *getExerciseDetailsTool) getExerciseDetails(ctx tool.Context, args getExerciseDetailsArgs) (getExerciseDetailsResult, error) {
	result, err := t.repo.GetExerciseDetails(ctx, args.ExerciseIDs)
	if err != nil {
		return getExerciseDetailsResult{
			Status: "error",
			Error:  err.Error(),
		}, nil
	}

	items := make([]exerciseDetails, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, exerciseDetails{
			ID:          item.ID,
			Name:        item.Name,
			Difficulty:  item.Difficulty,
			Description: item.Description,
			BodyParts:   item.BodyParts,
			Equipment:   item.Equipment,
		})
	}

	return getExerciseDetailsResult{
		Status: "success",
		Items:  items,
	}, nil
}
