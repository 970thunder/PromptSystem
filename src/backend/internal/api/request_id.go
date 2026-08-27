package api

import (
	"context"
	"net/http"
)

type requestIDContextKey struct{}

func withRequestIDCtx(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, id)
}

func requestIDFrom(r *http.Request) string {
	id, _ := r.Context().Value(requestIDContextKey{}).(string)
	return id
}
