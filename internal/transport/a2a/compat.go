package a2a

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

type bufferedResponseWriter struct {
	header     http.Header
	body       bytes.Buffer
	statusCode int
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{
		header: make(http.Header),
	}
}

func (w *bufferedResponseWriter) Header() http.Header {
	return w.header
}

func (w *bufferedResponseWriter) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.body.Write(data)
}

func (w *bufferedResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *bufferedResponseWriter) Flush() {}

func (w *bufferedResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, fmt.Errorf("hijack not supported")
}

func (w *bufferedResponseWriter) Push(target string, opts *http.PushOptions) error {
	return http.ErrNotSupported
}

func withOutgoingA2ACompat(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := newBufferedResponseWriter()
		next.ServeHTTP(rec, r)

		for key, values := range rec.Header() {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		body := rec.body.String()
		contentType := rec.Header().Get("Content-Type")

		if rec.statusCode != 0 {
			w.WriteHeader(rec.statusCode)
		}

		if !strings.Contains(strings.ToLower(contentType), "text/event-stream") {
			_, _ = w.Write([]byte(body))
			return
		}

		rewritten := rewriteSSEPayloads(body)
		_, _ = w.Write([]byte(rewritten))

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	})
}

func rewriteSSEPayloads(raw string) string {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	var out strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			out.WriteString("data: ")
			out.WriteString(rewriteJSONRPCPayload(payload))
			out.WriteString("\n")
			continue
		}

		out.WriteString(line)
		out.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		log.Printf("rewriteSSEPayloads scanner error: %v", err)
		return raw
	}

	return out.String()
}

func rewriteJSONRPCPayload(payload string) string {
	if strings.TrimSpace(payload) == "" {
		return payload
	}

	var root map[string]any
	if err := json.Unmarshal([]byte(payload), &root); err != nil {
		return payload
	}

	result, ok := root["result"].(map[string]any)
	if !ok {
		return payload
	}

	switch {
	case result["message"] != nil:
		message, ok := result["message"].(map[string]any)
		if !ok {
			return payload
		}
		normalizeMessageShape(message)
		root["result"] = message

	case result["task"] != nil:
		task, ok := result["task"].(map[string]any)
		if !ok {
			return payload
		}
		task["kind"] = "task"
		normalizeTaskShape(task)
		root["result"] = task

	case result["statusUpdate"] != nil:
		statusUpdate, ok := result["statusUpdate"].(map[string]any)
		if !ok {
			return payload
		}

		out := map[string]any{
			"kind":      "status-update",
			"taskId":    statusUpdate["taskId"],
			"contextId": statusUpdate["contextId"],
			"status":    statusUpdate["status"],
			"final":     false,
		}

		if status, ok := out["status"].(map[string]any); ok {
			if state, _ := status["state"].(string); state != "" {
				status["state"] = normalizeState(state)
				out["final"] = isFinalState(state)
			}
			normalizeMessageShape(status["message"])
		}

		root["result"] = out

	case result["artifactUpdate"] != nil:
		artifactUpdate, ok := result["artifactUpdate"].(map[string]any)
		if !ok {
			return payload
		}

		appendValue, _ := artifactUpdate["append"].(bool)
		lastChunkValue := extractLastChunk(artifactUpdate)
		artifact := artifactUpdate["artifact"]
		normalizeArtifactShape(artifact)

		out := map[string]any{
			"kind":      "artifact-update",
			"taskId":    artifactUpdate["taskId"],
			"contextId": artifactUpdate["contextId"],
			"append":    appendValue,
			"lastChunk": lastChunkValue,
			"artifact":  artifact,
		}

		root["result"] = out
	}

	updated, err := json.Marshal(root)
	if err != nil {
		log.Printf("rewriteJSONRPCPayload marshal error: %v", err)
		return payload
	}

	return string(updated)
}

func extractLastChunk(artifactUpdate map[string]any) bool {
	if v, ok := artifactUpdate["lastChunk"].(bool); ok {
		return v
	}

	artifact, ok := artifactUpdate["artifact"].(map[string]any)
	if !ok {
		return false
	}

	parts, ok := artifact["parts"].([]any)
	if !ok || len(parts) == 0 {
		return false
	}

	part, ok := parts[0].(map[string]any)
	if !ok {
		return false
	}

	metadata, ok := part["metadata"].(map[string]any)
	if !ok {
		return false
	}

	v, _ := metadata["lastChunk"].(bool)
	return v
}

func normalizeTaskShape(task map[string]any) {
	if status, ok := task["status"].(map[string]any); ok {
		if state, _ := status["state"].(string); state != "" {
			status["state"] = normalizeState(state)
		}
	}

	if history, ok := task["history"].([]any); ok {
		for _, item := range history {
			normalizeMessageShape(item)
		}
	}
}

func normalizeMessageShape(raw any) {
	msg, ok := raw.(map[string]any)
	if !ok {
		return
	}

	msg["kind"] = "message"

	if id, ok := msg["id"]; ok {
		if _, hasMessageID := msg["messageId"]; !hasMessageID {
			msg["messageId"] = id
		}
	}

	if parts, ok := msg["parts"].([]any); ok {
		for i, rawPart := range parts {
			part, ok := rawPart.(map[string]any)
			if !ok {
				continue
			}
			normalizePartShape(part)
			parts[i] = part
		}
		msg["parts"] = parts
	}
}

func normalizeArtifactShape(raw any) {
	artifact, ok := raw.(map[string]any)
	if !ok {
		return
	}

	if id, ok := artifact["id"]; ok {
		if _, hasArtifactID := artifact["artifactId"]; !hasArtifactID {
			artifact["artifactId"] = id
		}
	}

	if artifactID, ok := artifact["artifactId"]; ok {
		if _, hasID := artifact["id"]; !hasID {
			artifact["id"] = artifactID
		}
	}

	if parts, ok := artifact["parts"].([]any); ok {
		for i, rawPart := range parts {
			part, ok := rawPart.(map[string]any)
			if !ok {
				continue
			}
			normalizePartShape(part)
			parts[i] = part
		}
		artifact["parts"] = parts
	}
}

func normalizePartShape(part map[string]any) {
	if _, hasKind := part["kind"]; !hasKind {
		switch {
		case part["data"] != nil:
			part["kind"] = "data"
		case part["text"] != nil:
			part["kind"] = "text"
		}
	}

	if part["data"] != nil {
		metadata, _ := part["metadata"].(map[string]any)
		if metadata == nil {
			metadata = map[string]any{}
			part["metadata"] = metadata
		}
		if _, ok := metadata["mimeType"]; !ok {
			metadata["mimeType"] = a2uiMimeType
		}
	}
}

func normalizeState(state string) string {
	switch strings.ToUpper(state) {
	case "TASK_STATE_SUBMITTED":
		return "submitted"
	case "TASK_STATE_WORKING":
		return "working"
	case "TASK_STATE_INPUT_REQUIRED":
		return "input-required"
	case "TASK_STATE_COMPLETED":
		return "completed"
	case "TASK_STATE_CANCELED":
		return "canceled"
	case "TASK_STATE_FAILED":
		return "failed"
	default:
		return strings.ToLower(state)
	}
}

func isFinalState(state string) bool {
	switch strings.ToUpper(state) {
	case "TASK_STATE_COMPLETED", "TASK_STATE_CANCELED", "TASK_STATE_FAILED":
		return true
	default:
		return false
	}
}

func withJSONRPCMethodCompat(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Body == nil {
			next.ServeHTTP(w, r)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("read request body: %v", err), http.StatusBadRequest)
			return
		}
		_ = r.Body.Close()

		if len(bytes.TrimSpace(bodyBytes)) == 0 {
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			next.ServeHTTP(w, r)
			return
		}

		var payload map[string]any
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			next.ServeHTTP(w, r)
			return
		}

		method, _ := payload["method"].(string)
		rewritten := false

		switch method {
		case "message/stream":
			payload["method"] = "SendStreamingMessage"
			rewritten = true
		case "message/send":
			payload["method"] = "SendMessage"
			rewritten = true
		}

		if rewritten {
			updatedBody, err := json.Marshal(payload)
			if err != nil {
				http.Error(w, fmt.Sprintf("marshal rewritten body: %v", err), http.StatusInternalServerError)
				return
			}

			log.Printf("JSON-RPC compatibility rewrite: method=%q -> %q", method, payload["method"])

			r.Body = io.NopCloser(bytes.NewReader(updatedBody))
			r.ContentLength = int64(len(updatedBody))
			r.Header.Set("Content-Length", fmt.Sprintf("%d", len(updatedBody)))
			next.ServeHTTP(w, r)
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		r.ContentLength = int64(len(bodyBytes))
		r.Header.Set("Content-Length", fmt.Sprintf("%d", len(bodyBytes)))
		next.ServeHTTP(w, r)
	})
}
