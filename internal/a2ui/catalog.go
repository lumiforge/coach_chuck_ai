package a2ui

import _ "embed"

//go:embed workout_catalog.json
var WorkoutCatalogJSON []byte

const WorkoutCatalogID = "https://github.com/lumiforge/coach_chuck/catalogs/workout/v1"
