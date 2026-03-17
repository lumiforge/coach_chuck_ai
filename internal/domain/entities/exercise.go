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

type ExerciseDetails struct {
	ID          int64
	Name        string
	Difficulty  string
	Description string
	BodyParts   []string
	Equipment   []string
}

type ExerciseDetailsResult struct {
	Items []ExerciseDetails
}

type TaxonomyEntity string

const (
	TaxonomyEntityBodyParts TaxonomyEntity = "body_parts"
	TaxonomyEntityEquipment TaxonomyEntity = "equipment"
)

type TaxonomyListResult struct {
	Items []string
}
