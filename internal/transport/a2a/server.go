package a2a

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
)

type Server struct {
	port           string
	httpServer     *http.Server
	sessionService session.Service
	runner         *runner.Runner
	agent          agent.Agent
}

func NewServer(_ context.Context, uiAgent agent.Agent, port string) (*Server, error) {
	sessionService := session.InMemoryService()

	r, err := runner.New(runner.Config{
		AppName:        appName,
		Agent:          uiAgent,
		SessionService: sessionService,
	})
	if err != nil {
		return nil, fmt.Errorf("create runner: %w", err)
	}

	executor := &executor{
		runner:         r,
		sessionService: sessionService,
	}

	requestHandler := a2asrv.NewHandler(executor)

	jsonrpcHandler := a2asrv.NewJSONRPCHandler(
		requestHandler,
		a2asrv.WithTransportKeepAlive(15*time.Second),
	)

	restHandler := a2asrv.NewRESTHandler(requestHandler)

	mux := http.NewServeMux()
	agentCardHandler := serveAgentCard(port)

	mux.HandleFunc(a2asrv.WellKnownAgentCardPath, agentCardHandler)
	mux.HandleFunc("/.well-known/agent.json", agentCardHandler)

	mux.Handle("/message:send", withEndpointLog("REST message:send", restHandler))
	mux.Handle("/message:stream", withEndpointLog("REST message:stream", restHandler))
	mux.Handle("/tasks", withEndpointLog("REST tasks", restHandler))
	mux.Handle("/tasks/", withEndpointLog("REST tasks/*", restHandler))

	mux.Handle(
		"/",
		withEndpointLog(
			"JSON-RPC /",
			withJSONRPCMethodCompat(
				withOutgoingA2ACompat(jsonrpcHandler),
			),
		),
	)

	mux.Handle(
		"/a2a/invoke",
		withEndpointLog(
			"JSON-RPC /a2a/invoke",
			withJSONRPCMethodCompat(
				withOutgoingA2ACompat(jsonrpcHandler),
			),
		),
	)

	addr := ":" + port

	return &Server{
		port:           port,
		sessionService: sessionService,
		runner:         r,
		agent:          uiAgent,
		httpServer: &http.Server{
			Addr:    addr,
			Handler: withRequestDump(withCORS(mux)),
		},
	}, nil
}

func (s *Server) Serve() error {
	log.Printf("A2A server starts on http://localhost:%s/a2a/invoke", s.port)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
