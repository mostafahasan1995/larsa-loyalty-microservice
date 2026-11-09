package middleware

import (
	"encoding/json"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/models"
	"loyalty/pkg/util"
	"net/http"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"github.com/go-chi/chi"
)

func Audit(ActionName constants.ActionName, ref string, fieldNames ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := util.NewResponseWriterWrapper(w)
			next.ServeHTTP(response, r)

			go func() {
				if *response.StatusCode == 200 {
					headerParams := common.ExtractHeaderParams(r)
					fieldName := "id"
					if len(fieldNames) > 0 {
						fieldName = fieldNames[0]
					}
					id := chi.URLParam(r, fieldName)

					err := util.ActivityStream(r.Context(), models.Activity{
						Headers:    headerParams,
						ActionName: ActionName.String(),
						ActionData: map[string]any{
							"id":  id,
							"ref": ref,
						},
					})
					if err != nil {
						fmt.Println("🚀 ~ returnhttp.HandlerFunc ~ err:", err.Error())
					}
				}
			}()

		})
	}
}

func AuditForCreate(ActionName constants.ActionName, ref string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := util.NewResponseWriterWrapper(w)
			next.ServeHTTP(response, r)

			var payload map[string]any

			if errDec := json.NewDecoder(response.Body).Decode(&payload); errDec != nil {
				http.Error(w, errDec.Error(), http.StatusBadRequest)
				return
			}

			go func() {
				if *response.StatusCode == 200 {
					headerParams := common.ExtractHeaderParams(r)

					id := payload["_id"]

					err := util.ActivityStream(r.Context(), models.Activity{
						Headers:    headerParams,
						ActionName: ActionName.String(),
						ActionData: map[string]any{
							"id":  id,
							"ref": ref,
						},
					})
					if err != nil {
						fmt.Println("🚀 ~ returnhttp.HandlerFunc ~ err:", err.Error())
					}
				}
			}()

		})
	}
}
