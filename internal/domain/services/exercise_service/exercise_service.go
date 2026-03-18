package exercise_service

import (
	"context"
	"fmt"

	"github.com/lumiforge/coach_chuck_ai/internal/domain/entities"
	"github.com/lumiforge/coach_chuck_ai/pkg/utils"
)

type ExercisesRepository interface {
	GetExerciseDetails(ctx context.Context, exerciseIDs []int64) (entities.ExerciseDetailsResult, error)
	SearchExercises(ctx context.Context, filter entities.ExerciseSearchFilter) (entities.ExerciseSearchResult, error)
}

type ExercisesService struct {
	repo ExercisesRepository
}

func NewExercisesService(repo ExercisesRepository) *ExercisesService {
	return &ExercisesService{repo: repo}
}

func (s *ExercisesService) GetExerciseDetails(ctx context.Context, input entities.GetExerciseDetailsInput) (entities.GetExerciseDetailsOutput, error) {
	if len(input.ExerciseIDs) == 0 {
		return entities.GetExerciseDetailsOutput{}, fmt.Errorf("exercise_ids is required")
	}

	result, err := s.repo.GetExerciseDetails(ctx, input.ExerciseIDs)
	if err != nil {
		return entities.GetExerciseDetailsOutput{}, err
	}

	return entities.GetExerciseDetailsOutput{
		Items: result.Items,
	}, nil
}

func (s *ExercisesService) SearchExercises(ctx context.Context, input entities.SearchExercisesInput) (entities.SearchExercisesOutput, error) {
	if err := utils.ValidateAllowedStrings(input.BodyPartsAny, allowedBodyParts, "body_parts_any"); err != nil {
		return entities.SearchExercisesOutput{}, err
	}

	if err := utils.ValidateAllowedStrings(input.EquipmentAny, allowedEquipment, "equipment_any"); err != nil {
		return entities.SearchExercisesOutput{}, err
	}

	if err := utils.ValidateAllowedStrings(input.DifficultyAny, allowedDifficulty, "difficulty_any"); err != nil {
		return entities.SearchExercisesOutput{}, err
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	result, err := s.repo.SearchExercises(ctx, entities.ExerciseSearchFilter{
		BodyPartsAny:       input.BodyPartsAny,
		EquipmentAny:       input.EquipmentAny,
		DifficultyAny:      input.DifficultyAny,
		ExcludeExerciseIDs: input.ExcludeExerciseIDs,
		Limit:              limit,
		Offset:             offset,
	})
	if err != nil {
		return entities.SearchExercisesOutput{}, err
	}

	return entities.SearchExercisesOutput{
		Items:   result.Items,
		Total:   result.Total,
		HasMore: offset+len(result.Items) < result.Total,
	}, nil
}
