// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package tsweb

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"tailscale.com/util/ctxkey"
)

// RequestID is an opaque identifier for a HTTP request, used to correlate
// user-visible errors with backend server logs. The RequestID is typically
// threaded through an HTTP Middleware (WithRequestID) and then can be extracted
// by HTTP Handlers to include in their logs.
//
// RequestID is an opaque identifier for a HTTP request, used to correlate
// user-visible errors with backend server logs. If present in the context, the
// RequestID will be printed alongside the message text and logged in the
// AccessLogRecord.
//
// A RequestID has the format "REQ-1{ID}", and the ID should be treated as an
// opaque string. The current implementation uses a UUID.
type RequestID string

// RequestIDKey stores and loads [RequestID] values within a [context.Context].
var RequestIDKey ctxkey.Key[RequestID]

// RequestIDHeader is a custom HTTP header that the WithRequestID middleware
// uses to determine whether to re-use a given request ID from the client
// or generate a new one.
const RequestIDHeader = "X-Tailscale-Request-Id"

// SetRequestID is an HTTP middleware that injects a RequestID in the
// *http.Request Context. The value of that request id is either retrieved from
// the RequestIDHeader or a randomly generated one if not exists. Inner
// handlers can retrieve this ID from the RequestIDFromContext function.
func SetRequestID(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			// REQ-1 indicates the version of the RequestID pattern. It is
			// currently arbitrary but allows for forward compatible
			// transitions if needed.
			id = "REQ-1" + uuid.NewString()
		}
		ctx := RequestIDKey.WithValue(r.Context(), RequestID(id))
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}

// RequestIDFromContext retrieves the RequestID from context that can be set by
// the SetRequestID function.
//
// Deprecated: Use [RequestIDKey.Value] instead.
func RequestIDFromContext(ctx context.Context) RequestID {
	return RequestIDKey.Value(ctx)
}
