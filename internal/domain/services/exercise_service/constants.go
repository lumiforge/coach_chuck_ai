package exercise_service

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
