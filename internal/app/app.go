package app

import (
	"context"
	"fmt"
	"time"

	adkopenai "github.com/byebyebruce/adk-go-openai"
	"github.com/lumiforge/coach_chuck_ai/internal/adk/agents/coach"
	adktools "github.com/lumiforge/coach_chuck_ai/internal/adk/tools"
	"github.com/lumiforge/coach_chuck_ai/internal/config"
	"github.com/lumiforge/coach_chuck_ai/internal/domain/services/exercise_service"
	"github.com/lumiforge/coach_chuck_ai/internal/repositories/exercise_repository"
	"github.com/lumiforge/coach_chuck_ai/internal/transport/a2a"
	"github.com/lumiforge/coach_chuck_ai/pkg/client/postgresql"
	goopenai "github.com/sashabaranov/go-openai"
	"google.golang.org/adk/tool"
)

type App struct {
	cfg       *config.Config
	a2aServer *a2a.Server
}

func NewApp(ctx context.Context, cfg *config.Config) (*App, error) {
	pgConfig := postgresql.NewPgConfig(
		cfg.PostgreSQL.Username,
		cfg.PostgreSQL.Password,
		cfg.PostgreSQL.Host,
		cfg.PostgreSQL.Port,
		cfg.PostgreSQL.Database,
	)

	pgClient, err := postgresql.NewClient(ctx, 5, 5*time.Second, pgConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres client: %w", err)
	}

	exerciseRepo := exercise_repository.NewExerciseRepository(pgClient)
	exerciseService := exercise_service.NewExercisesService(exerciseRepo)

	searchExercisesTool, err := adktools.NewSearchExercisesTool(exerciseService)
	if err != nil {
		return nil, fmt.Errorf("create search_exercises tool: %w", err)
	}

	getExerciseDetailsTool, err := adktools.NewGetExerciseDetailsTool(exerciseService)
	if err != nil {
		return nil, fmt.Errorf("create get_exercise_details tool: %w", err)
	}

	openaiCfg := goopenai.DefaultConfig(cfg.CoachAgent.OpenAI.APIKey)
	if cfg.CoachAgent.OpenAI.BaseURL != "" {
		openaiCfg.BaseURL = cfg.CoachAgent.OpenAI.BaseURL
	}

	model := adkopenai.NewOpenAIModel(cfg.CoachAgent.ModelName, openaiCfg)

	rootAgent, err := coach.NewRootAgent(model, []tool.Tool{
		searchExercisesTool,
		getExerciseDetailsTool,
	})
	if err != nil {
		return nil, fmt.Errorf("create root agent: %w", err)
	}

	a2aServer, err := a2a.NewServer(ctx, rootAgent, cfg.A2A.Port)
	if err != nil {
		return nil, fmt.Errorf("create a2a server: %w", err)
	}

	return &App{
		cfg:       cfg,
		a2aServer: a2aServer,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	return a.a2aServer.Serve()
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.a2aServer.Shutdown(ctx)
}
