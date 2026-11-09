package middleware

import (
	"context"
	"loyalty/pkg/constants"
	"net/http"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
)

func Auth(restructions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := make(chan common.Credentials)

			go common.Guard(c, common.ExtractHeaderParams(r), restructions)

			credentials := <-c

			if credentials.Err != nil {
				http.Error(w, "not authenticated", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), constants.CtxAuthKey, credentials)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
