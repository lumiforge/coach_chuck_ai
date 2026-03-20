package a2ui

import (
	"fmt"
	"strings"
)

func BuildRepairPrompt(originalUserInput, previousOutput string, issues []ValidationIssue) string {
	var b strings.Builder

	errorMessage := FormatValidationIssues(issues)
	if strings.TrimSpace(errorMessage) == "" {
		errorMessage = "Validation failed."
	}
	b.WriteString("Your previous response was invalid. ")
	b.WriteString(errorMessage)
	b.WriteString(" ")
	b.WriteString("You MUST generate a valid response that strictly follows the A2UI JSON SCHEMA. ")
	b.WriteString("The response MUST be a JSON list of A2UI messages. ")
	b.WriteString("Ensure each JSON part is wrapped in '")
	b.WriteString(A2UIOpenTag)
	b.WriteString("' and '")
	b.WriteString(A2UICloseTag)
	b.WriteString("' tags. ")
	b.WriteString("Ensure each JSON part is wrapped in '<A2UI>' and '</A2UI>' tags. ")

	if strings.TrimSpace(originalUserInput) != "" {
		b.WriteString("Please retry the original request: '")
		b.WriteString(strings.ReplaceAll(originalUserInput, "'", "\\'"))
		b.WriteString("'")
	}

	if strings.TrimSpace(previousOutput) != "" {
		b.WriteString("Previous invalid output:\n")
		b.WriteString(previousOutput)
		b.WriteString("\n")
	}

	return b.String()
}

const (
	A2UIOpenTag  = "<A2UI>"
	A2UICloseTag = "</A2UI>"
)

func FormatValidationIssues(issues []ValidationIssue) string {
	if len(issues) == 0 {
		return ""
	}

	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Path, issue.Message))
	}

	return strings.Join(parts, "; ")
}

func ValidationIssuesAsErrorEnvelopes(issues []ValidationIssue) []map[string]any {
	out := make([]map[string]any, 0, len(issues))
	for _, issue := range issues {
		out = append(out, issue.ErrorEnvelope())
	}
	return out
}
