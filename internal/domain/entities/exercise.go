package entities

type ExerciseSearchFilter struct {
	BodyPartsAny       []string
	EquipmentAny       []string
	DifficultyAny      []string
	ExcludeExerciseIDs []int64
	Limit              int
	Offset             int
}

type ExerciseSummary struct {
	ID         int64
	Name       string
	Difficulty string
	BodyParts  []string
	Equipment  []string
}

type ExerciseSearchResult struct {
	Items []ExerciseSummary
	Total int
}
