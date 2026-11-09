package middleware

import (
	"context"
	"loyalty/pkg/constants"
	"net/http"
)

func AddLangToContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := r.Header.Get("accept-language")
		ctx := context.WithValue(r.Context(), constants.CtxLangKey, lang)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
