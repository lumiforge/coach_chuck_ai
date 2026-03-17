package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	adkopenai "github.com/byebyebruce/adk-go-openai"
	"github.com/lumiforge/coach_chuck_ai/internal/domain/tools"
	"github.com/lumiforge/coach_chuck_ai/internal/repository"
	pgclient "github.com/lumiforge/coach_chuck_ai/pkg/client/postgresql"
	goopenai "github.com/sashabaranov/go-openai"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/tool"
)

func main() {
	ctx := context.Background()

	modelName := os.Getenv("OPENAI_MODEL")
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}

	openaiCfg := goopenai.DefaultConfig(os.Getenv("HYDRA_AI_API_KEY"))
	if baseURL := os.Getenv("HYDRA_AI_BASE_URL"); baseURL != "" {
		openaiCfg.BaseURL = baseURL
	}

	fmt.Printf("URL: %s\nOrg ID: %s\nModel: %s\n", openaiCfg.BaseURL, openaiCfg.OrgID, modelName)

	model := adkopenai.NewOpenAIModel(modelName, openaiCfg)

	pgCfg := pgclient.NewPgConfig(
		os.Getenv("CHUCK_AI_POSTGRES_USER"),
		os.Getenv("CHUCK_AI_POSTGRES_PASSWORD"),
		os.Getenv("CHUCK_AI_POSTGRES_HOST"),
		os.Getenv("CHUCK_AI_POSTGRES_PORT"),
		os.Getenv("CHUCK_AI_POSTGRES_DB"),
	)

	pg, err := pgclient.NewClient(ctx, 3, 2*time.Second, pgCfg)
	if err != nil {
		log.Fatalf("failed to create postgres client: %v", err)
	}

	repo := repository.NewExerciseRepository(pg)

	searchExercisesTool, err := tools.NewSearchExercisesTool(repo)
	if err != nil {
		log.Fatalf("failed to create search_exercises tool: %v", err)
	}

	getExerciseDetailsTool, err := tools.NewGetExerciseDetailsTool(repo)
	if err != nil {
		log.Fatalf("failed to create get_exercise_details tool: %v", err)
	}

	rootAgent, err := llmagent.New(llmagent.Config{
		Name:        "coach_chuck",
		Model:       model,
		Description: "Agent that helps select exercises from the exercise catalog and explain them.",
		Instruction: "You are a fitness assistant with access to an exercise catalog. Use search_exercises to find candidate exercises by body parts, equipment, and difficulty. Use get_exercise_details only after you have specific exercise IDs and need full descriptions. Never invent exercise IDs, body parts, equipment names, or exercise details. If the user asks for workout ideas, first retrieve relevant exercises from the catalog, then explain the options clearly.",
		Tools: []tool.Tool{
			searchExercisesTool,
			getExerciseDetailsTool,
		},
	})
	if err != nil {
		log.Fatalf("failed to create root agent: %v", err)
	}

	cfg := &launcher.Config{
		AgentLoader: agent.NewSingleLoader(rootAgent),
	}

	l := full.NewLauncher()
	if err := l.Execute(ctx, cfg, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
