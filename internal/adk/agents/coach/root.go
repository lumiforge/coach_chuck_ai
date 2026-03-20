package coach

import (
	"fmt"

	cataloga2ui "github.com/lumiforge/coach_chuck_ai/internal/a2ui"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
)

func NewRootAgent(llm model.LLM, tools []tool.Tool) (agent.Agent, error) {

	schemaManager, err := cataloga2ui.NewSchemaManager()
	if err != nil {
		return nil, fmt.Errorf("create a2ui schema manager: %w", err)
	}
	instruction, err := schemaManager.GenerateSystemPrompt(
		RoleDescription,
		A2UIPrompt,
		true,
		true,
		true,
	)
	if err != nil {
		return nil, fmt.Errorf("generate a2ui system prompt: %w", err)
	}
	a, err := llmagent.New(llmagent.Config{
		Name:        "coach_chuck",
		Description: "Fitness assistant that returns A2UI v0.9 UI responses.",
		Model:       llm,
		Instruction: LanguageRule + "\n" + instruction,
		Tools:       tools,
	})
	if err != nil {
		return nil, fmt.Errorf("create llm agent: %w", err)
	}

	return a, nil
}
