package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestIDKey struct{}

const (
	RequestIDHeader   = "X-Request-ID"
	CorrelationIDAnno = "cloudivision.io/correlation-id"
)

func WithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}

func RequestIDFromRequest(r *http.Request) string {
	if requestID := r.Header.Get(RequestIDHeader); requestID != "" {
		return requestID
	}
	return NewRequestID()
}

func NewRequestID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "req-unknown"
	}
	return hex.EncodeToString(buf[:])
}
