package api

import (
	"context"
	"net/http"
)

type requestIDContextKey struct{}

func withRequestIDCtx(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, id)
}

// RequestIDFrom 从 ctx 取请求 id（站点中间件生成或透传的那个）。
// 平台统一契约需要它：不提供的话，平台面的 RequestId 会退回读调用方可控的
// X-Management-Request-Id 头，审计里的请求 id 就可能是调用方编的（评审 P2-2）。
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDContextKey{}).(string)
	return id
}

// WithRequestID 暴露请求 id 中间件，供站点链之外的处理链（平台面）复用。
func WithRequestID(next http.Handler) http.Handler { return withRequestID(next) }

func requestIDFrom(r *http.Request) string {
	id, _ := r.Context().Value(requestIDContextKey{}).(string)
	return id
}
