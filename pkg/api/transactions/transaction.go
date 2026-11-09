package transactions

import (
	"fmt"
	"loyalty/pkg/caching"
	"loyalty/pkg/constants"
	"loyalty/pkg/models"
	"loyalty/pkg/services"
	"loyalty/pkg/util"
	"net/http"

	"github.com/goccy/go-json"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"github.com/go-chi/chi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BulkDeleteRequest represents the request for bulk delete operations
type BulkDeleteRequest struct {
	Ids []string `json:"ids"`
}

// BulkDeleteResponse represents the response for bulk delete operations
type BulkDeleteResponse struct {
	TotalCount   int      `json:"totalCount"`
	SuccessCount int      `json:"successCount"`
	Errors       []string `json:"errors"`
}

func GetTransactions(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	rawQuery := r.URL.RawQuery
	path := r.URL.Path

	cacheKey := util.CacheKeyForMulti(constants.TRANSACTION, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		args, err := util.Paginate(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		transactionService := services.NewTransactionSvcs(r.Context(), db)

		result, err := transactionService.Get(args, nil)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		j, _ := json.MarshalContext(r.Context(), result)
		//add to cache
		caching.CacheByteByKey(cacheKey, j, 0)

		w.Write(j)
	} else {
		fmt.Println("serve from cache")
		w.Write([]byte(cache))
	}
}

func GetAllTransactions(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	//

	rawQuery := r.URL.RawQuery
	path := r.URL.Path
	query := r.URL.Query().Get("query")

	cacheKey := util.CacheKeyForMulti(constants.TRANSACTION, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		transactionService := services.NewTransactionSvcs(r.Context(), db)
		result, err := transactionService.GetAll(query)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		j, _ := json.MarshalContext(r.Context(), result)
		//add to cache
		caching.CacheByteByKey(cacheKey, j, 0)

		w.Write(j)
	} else {
		w.Write([]byte(cache))
	}
}

func GetTransactionById(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	cacheKey := util.CacheKeyForOne(constants.TRANSACTION, db, id, lang)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		transactionService := services.NewTransactionSvcs(r.Context(), db)
		result, err := transactionService.GetOne(id)

		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		j, _ := json.MarshalContext(r.Context(), result)
		//add to cache
		caching.CacheByteByKey(cacheKey, j, 0)

		w.Write(j)
	} else {
		w.Write([]byte(cache))
	}

}

func CreateTransaction(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var data models.Transaction

	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	transactionService := services.NewTransactionSvcs(r.Context(), db)
	res, errIns := transactionService.Add(&data)
	if errIns != nil {
		http.Error(w, errIns.Error(), http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.TRANSACTION, db), 0)

	j, _ := json.MarshalContext(r.Context(), res)

	w.Write(j)
}

func DeleteTransaction(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	transactionService := services.NewTransactionSvcs(r.Context(), db)

	if errDel := transactionService.Delete(id); errDel != nil {
		http.Error(w, "error delete tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.TRANSACTION, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.TRANSACTION, db, id), 0)

	w.WriteHeader(http.StatusOK)
}

func BulkDeleteTransactions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var data BulkDeleteRequest
	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	if len(data.Ids) == 0 {
		http.Error(w, "No IDs provided for deletion", http.StatusBadRequest)
		return
	}

	transactionService := services.NewTransactionSvcs(r.Context(), db)

	var errors []string
	successCount := 0

	for _, id := range data.Ids {
		if err := transactionService.Delete(id); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to delete transaction %s: %s", id, err.Error()))
		} else {
			successCount++
			caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.TRANSACTION, db, id), 0)
		}
	}

	// Clear cache for all transactions
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.TRANSACTION, db), 0)

	response := BulkDeleteResponse{
		SuccessCount: successCount,
		TotalCount:   len(data.Ids),
		Errors:       errors,
	}

	j, _ := json.MarshalContext(r.Context(), response)
	w.Write(j)
}

func GetByUserId(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	userId := chi.URLParam(r, "id")
	_userId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	args, err := util.Paginate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	transactionService := services.NewTransactionSvcs(r.Context(), db)

	result, err := transactionService.Get(args, &_userId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}

func GetUserTransactions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	user := r.Context().Value(constants.CtxAuthKey).(common.Credentials)
	_userId, _ := primitive.ObjectIDFromHex(user.User.Id)

	args, err := util.Paginate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	service := services.NewTransactionSvcs(r.Context(), db)

	result, err := service.Get(args, &_userId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}
