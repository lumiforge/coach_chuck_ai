package a2a

import (
	"context"

	"google.golang.org/adk/session"
)

func ensureSession(ctx context.Context, sessionService session.Service, sessionID string) (string, error) {
	if sessionID == "" {
		resp, err := sessionService.Create(ctx, &session.CreateRequest{
			AppName: appName,
			UserID:  a2aUserID,
		})
		if err != nil {
			return "", err
		}
		return resp.Session.ID(), nil
	}

	_, err := sessionService.Get(ctx, &session.GetRequest{
		AppName:   appName,
		UserID:    a2aUserID,
		SessionID: sessionID,
	})
	if err == nil {
		return sessionID, nil
	}

	_, err = sessionService.Create(ctx, &session.CreateRequest{
		AppName:   appName,
		UserID:    a2aUserID,
		SessionID: sessionID,
	})
	if err != nil {
		return "", err
	}

	return sessionID, nil
}
