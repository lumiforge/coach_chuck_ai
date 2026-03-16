package repository

import (
	"context"
	"strings"

	"github.com/lumiforge/coach_chuck_ai/internal/domain/entities"
	client "github.com/lumiforge/coach_chuck_ai/pkg/client/postgresql"
)

type exerciseRepository struct {
	client client.PostgreSQLClient
}

func (r *exerciseRepository) SearchExercises(ctx context.Context, filter entities.ExerciseSearchFilter) (entities.ExerciseSearchResult, error) {
	const defaultLimit = 20
	const maxLimit = 50

	limit := filter.Limit
	switch {
	case limit <= 0:
		limit = defaultLimit
	case limit > maxLimit:
		limit = maxLimit
	}

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	bodyPartsAny := normalizeStringSlice(filter.BodyPartsAny)
	equipmentAny := normalizeStringSlice(filter.EquipmentAny)
	difficultyAny := normalizeStringSlice(filter.DifficultyAny)
	excludeExerciseIDs := normalizeInt64Slice(filter.ExcludeExerciseIDs)

	const countSQL = `
SELECT COUNT(*)
FROM exercises e
WHERE
	($1::text[] IS NULL OR EXISTS (
		SELECT 1
		FROM exercise_body_parts ebp
		INNER JOIN body_parts bp ON bp.id = ebp.body_part_id
		WHERE ebp.exercise_id = e.id
		  AND bp.name = ANY($1::text[])
	))
	AND ($2::text[] IS NULL OR EXISTS (
		SELECT 1
		FROM exercise_equipment ee
		INNER JOIN equipment eq ON eq.id = ee.equipment_id
		WHERE ee.exercise_id = e.id
		  AND eq.name = ANY($2::text[])
	))
	AND ($3::text[] IS NULL OR e.difficulty = ANY($3::text[]))
	AND ($4::bigint[] IS NULL OR NOT (e.id = ANY($4::bigint[])));
`

	var total int
	if err := r.client.QueryRow(
		ctx,
		countSQL,
		bodyPartsAny,
		equipmentAny,
		difficultyAny,
		excludeExerciseIDs,
	).Scan(&total); err != nil {
		return entities.ExerciseSearchResult{}, err
	}

	if total == 0 {
		return entities.ExerciseSearchResult{
			Items: []entities.ExerciseSummary{},
			Total: 0,
		}, nil
	}

	const dataSQL = `
WITH filtered AS (
	SELECT e.id, e.name, e.difficulty
	FROM exercises e
	WHERE
		($1::text[] IS NULL OR EXISTS (
			SELECT 1
			FROM exercise_body_parts ebp
			INNER JOIN body_parts bp ON bp.id = ebp.body_part_id
			WHERE ebp.exercise_id = e.id
			  AND bp.name = ANY($1::text[])
		))
		AND ($2::text[] IS NULL OR EXISTS (
			SELECT 1
			FROM exercise_equipment ee
			INNER JOIN equipment eq ON eq.id = ee.equipment_id
			WHERE ee.exercise_id = e.id
			  AND eq.name = ANY($2::text[])
		))
		AND ($3::text[] IS NULL OR e.difficulty = ANY($3::text[]))
		AND ($4::bigint[] IS NULL OR NOT (e.id = ANY($4::bigint[])))
	ORDER BY e.name ASC, e.id ASC
	LIMIT $5 OFFSET $6
)
SELECT
	f.id,
	f.name,
	f.difficulty,
	COALESCE(
		ARRAY_AGG(DISTINCT bp.name ORDER BY bp.name)
			FILTER (WHERE bp.name IS NOT NULL),
		'{}'::text[]
	) AS body_parts,
	COALESCE(
		ARRAY_AGG(DISTINCT eq.name ORDER BY eq.name)
			FILTER (WHERE eq.name IS NOT NULL),
		'{}'::text[]
	) AS equipment
FROM filtered f
LEFT JOIN exercise_body_parts ebp ON ebp.exercise_id = f.id
LEFT JOIN body_parts bp ON bp.id = ebp.body_part_id
LEFT JOIN exercise_equipment ee ON ee.exercise_id = f.id
LEFT JOIN equipment eq ON eq.id = ee.equipment_id
GROUP BY f.id, f.name, f.difficulty
ORDER BY f.name ASC, f.id ASC;
`

	rows, err := r.client.Query(
		ctx,
		dataSQL,
		bodyPartsAny,
		equipmentAny,
		difficultyAny,
		excludeExerciseIDs,
		limit,
		offset,
	)
	if err != nil {
		return entities.ExerciseSearchResult{}, err
	}
	defer rows.Close()

	items := make([]entities.ExerciseSummary, 0, limit)

	for rows.Next() {
		var item entities.ExerciseSummary

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Difficulty,
			&item.BodyParts,
			&item.Equipment,
		); err != nil {
			return entities.ExerciseSearchResult{}, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return entities.ExerciseSearchResult{}, err
	}

	return entities.ExerciseSearchResult{
		Items: items,
		Total: total,
	}, nil
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		out = append(out, value)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func normalizeInt64Slice(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}

	out := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))

	for _, value := range values {
		if value <= 0 {
			continue
		}

		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		out = append(out, value)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}
