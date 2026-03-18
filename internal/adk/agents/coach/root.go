package coach

import (
	"fmt"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
)

func NewRootAgent(llm model.LLM, tools []tool.Tool) (agent.Agent, error) {
	a, err := llmagent.New(llmagent.Config{
		Name:        "coach_chuck",
		Description: "Fitness assistant that returns A2UI v0.9 UI responses.",
		Model:       llm,
		Instruction: GetUIPrompt(),
		Tools:       tools,
		AfterModelCallbacks: []llmagent.AfterModelCallback{
			ValidateV09Output(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create llm agent: %w", err)
	}

	return a, nil
}
