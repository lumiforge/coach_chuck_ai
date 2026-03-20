package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log"
	"strings"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	cataloga2ui "github.com/lumiforge/coach_chuck_ai/internal/a2ui"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type executor struct {
	runner         *runner.Runner
	sessionService session.Service
}

const maxA2UIRepairAttempts = 3

func (e *executor) Execute(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	log.Printf("EXECUTE START: contextID=%q message=%+v", execCtx.ContextID, execCtx.Message)

	return func(yield func(a2a.Event, error) bool) {
		if execCtx.StoredTask == nil {
			if !yield(a2a.NewSubmittedTask(execCtx, execCtx.Message), nil) {
				return
			}
		}

		if !yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, nil), nil) {
			return
		}

		if clientErr, ok := extractClientA2UIError(execCtx.Message); ok {
			log.Printf(
				"CLIENT A2UI ERROR: contextID=%q taskID=%q error=%q",
				execCtx.ContextID,
				execCtx.TaskID,
				clientErr,
			)
			_ = yield(
				a2a.NewStatusUpdateEvent(
					execCtx,
					a2a.TaskStateFailed,
					a2aMessage("client A2UI error: "+clientErr),
				),
				nil,
			)
			return
		}

		originalInput := buildAgentInput(execCtx.Message)
		if strings.TrimSpace(originalInput) == "" {
			_ = yield(
				a2a.NewStatusUpdateEvent(
					execCtx,
					a2a.TaskStateFailed,
					a2aMessage("empty input"),
				),
				nil,
			)
			return
		}

		sessionID, err := ensureSession(ctx, e.sessionService, execCtx.ContextID)
		if err != nil {
			_ = yield(
				a2a.NewStatusUpdateEvent(
					execCtx,
					a2a.TaskStateFailed,
					a2aMessage(fmt.Sprintf("session error: %v", err)),
				),
				nil,
			)
			return
		}

		currentPrompt := originalInput
		var normalizedOutput string
		var lastValidationIssues []cataloga2ui.ValidationIssue

		for attempt := 1; attempt <= maxA2UIRepairAttempts; attempt++ {
			rawOutput, runErr := e.runModelTurn(ctx, sessionID, currentPrompt)
			if runErr != nil {
				_ = yield(
					a2a.NewStatusUpdateEvent(
						execCtx,
						a2a.TaskStateFailed,
						a2aMessage(fmt.Sprintf("agent run failed: %v", runErr)),
					),
					nil,
				)
				return
			}

			log.Printf(
				"RAW MODEL OUTPUT BEFORE A2UI VALIDATION (attempt=%d):\n%s",
				attempt,
				rawOutput,
			)

			responseParts := cataloga2ui.ParseResponse(rawOutput)

			if len(responseParts) == 0 {
				_ = yield(
					a2a.NewStatusUpdateEvent(
						execCtx,
						a2a.TaskStateFailed,
						a2aMessage("a2ui validation pipeline failed: no A2UI parts found"),
					),
					nil,
				)
				return
			}

			var normalizedParts []string
			var issues []cataloga2ui.ValidationIssue

			for _, part := range responseParts {
				normalized, partIssues, validationErr := cataloga2ui.NormalizeAndValidateJSONL(part)
				if validationErr != nil {
					_ = yield(
						a2a.NewStatusUpdateEvent(
							execCtx,
							a2a.TaskStateFailed,
							a2aMessage(fmt.Sprintf("a2ui validation pipeline failed: %v", validationErr)),
						),
						nil,
					)
					return
				}

				if len(partIssues) > 0 {
					issues = append(issues, partIssues...)
					continue
				}

				normalizedParts = append(normalizedParts, normalized)
			}

			if len(issues) == 0 {
				normalizedOutput = strings.Join(normalizedParts, "\n")
				break
			}

			lastValidationIssues = issues
			log.Printf(
				"INVALID A2UI OUTPUT (attempt=%d): %s",
				attempt,
				cataloga2ui.FormatValidationIssues(issues),
			)

			if attempt == maxA2UIRepairAttempts {
				break
			}

			currentPrompt = cataloga2ui.BuildRepairPrompt(originalInput, rawOutput, issues)
		}

		if strings.TrimSpace(normalizedOutput) == "" {
			_ = yield(
				a2a.NewStatusUpdateEvent(
					execCtx,
					a2a.TaskStateFailed,
					a2aMessage(
						fmt.Sprintf(
							"invalid A2UI v0.9 output after %d attempts: %s",
							maxA2UIRepairAttempts,
							cataloga2ui.FormatValidationIssues(lastValidationIssues),
						),
					),
				),
				nil,
			)
			return
		}

		messages, err := parseA2UIJSONL(normalizedOutput)
		if err != nil {
			_ = yield(
				a2a.NewStatusUpdateEvent(
					execCtx,
					a2a.TaskStateFailed,
					a2aMessage(fmt.Sprintf("invalid A2UI JSONL: %v", err)),
				),
				nil,
			)
			return
		}

		parts := make([]*a2a.Part, 0, len(messages))
		for i, m := range messages {
			part := a2a.NewDataPart(m)
			if part.Metadata == nil {
				part.Metadata = map[string]any{}
			}
			part.Metadata["mimeType"] = a2uiMimeType
			part.Metadata["lastChunk"] = i == len(messages)-1
			parts = append(parts, part)
		}

		msg := &a2a.Message{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			Role:      "agent",
			TaskID:    execCtx.TaskID,
			ContextID: execCtx.ContextID,
			Parts:     parts,
		}

		_ = yield(
			a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, msg),
			nil,
		)
	}
}

func (e *executor) Cancel(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		_ = yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, nil), nil)
	}
}

func (e *executor) runModelTurn(ctx context.Context, sessionID, input string) (string, error) {
	userMsg := &genai.Content{
		Role: string(genai.RoleUser),
		Parts: []*genai.Part{
			genai.NewPartFromText(input),
		},
	}

	var finalText string

	for event, runErr := range e.runner.Run(
		ctx,
		a2aUserID,
		sessionID,
		userMsg,
		agent.RunConfig{
			StreamingMode: agent.StreamingModeNone,
		},
	) {
		if runErr != nil {
			return "", runErr
		}

		if event == nil || event.Content == nil || event.Partial {
			continue
		}

		text := flattenText(event.Content.Parts)
		if strings.TrimSpace(text) != "" {
			finalText = text
		}
	}

	if strings.TrimSpace(finalText) == "" {
		return "", fmt.Errorf("empty final model response")
	}

	return finalText, nil
}

func parseA2UIJSONL(raw string) ([]map[string]any, error) {
	dec := json.NewDecoder(strings.NewReader(raw))
	out := make([]map[string]any, 0, 4)

	for {
		var msg map[string]any
		err := dec.Decode(&msg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no JSON objects found")
	}

	return out, nil
}

func a2aMessage(text string) *a2a.Message {
	return &a2a.Message{
		Role: "agent",
		Parts: []*a2a.Part{
			a2a.NewTextPart(text),
		},
	}
}

func flattenText(parts []*genai.Part) string {
	var b strings.Builder

	for _, p := range parts {
		if p == nil {
			continue
		}
		if txt := p.Text; strings.TrimSpace(txt) != "" {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(txt)
		}
	}

	return b.String()
}
