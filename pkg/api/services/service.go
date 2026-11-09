package services

import (
	"fmt"
	"loyalty/pkg/caching"
	"loyalty/pkg/constants"
	"loyalty/pkg/models"
	"loyalty/pkg/services"
	"loyalty/pkg/util"
	"net/http"
	_ "time"

	"github.com/goccy/go-json"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"github.com/go-chi/chi"
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

func GetServices(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	rawQuery := r.URL.RawQuery
	path := r.URL.Path

	cacheKey := util.CacheKeyForMulti(constants.SERVICE, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		args, err := util.Paginate(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		itemService := services.NewServiceSvcs(r.Context(), db)

		result, err := itemService.Get(args)

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

func GetAllServices(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	//

	rawQuery := r.URL.RawQuery
	path := r.URL.Path
	query := r.URL.Query().Get("query")

	cacheKey := util.CacheKeyForMulti(constants.SERVICE, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		itemService := services.NewServiceSvcs(r.Context(), db)
		result, err := itemService.GetAll(query)

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

func GetServiceById(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	cacheKey := util.CacheKeyForOne(constants.SERVICE, db, id, lang)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		itemService := services.NewServiceSvcs(r.Context(), db)
		result, err := itemService.GetOne(id)

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

func CreateService(w http.ResponseWriter, r *http.Request) {
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var payload models.Service

	if errDec := json.NewDecoder(r.Body).Decode(&payload); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	itemService := services.NewServiceSvcs(r.Context(), db)
	res, errIns := itemService.Add(&payload)
	if errIns != nil {
		http.Error(w, errIns.Error(), http.StatusInternalServerError)
		return
	}
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.SERVICE, db), 0)

	j, _ := json.MarshalContext(r.Context(), res)

	w.Write(j)
}

func UpdateService(w http.ResponseWriter, r *http.Request) {
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	var data models.Service
	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	service := services.NewServiceSvcs(r.Context(), db)
	res, err := service.Update(id, &data)
	if err != nil {
		http.Error(w, "error update tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.SERVICE, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.SERVICE, db, id), 0)

	j, _ := json.MarshalContext(r.Context(), res)

	w.Write(j)
}

func DeleteService(w http.ResponseWriter, r *http.Request) {
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	taxService := services.NewServiceSvcs(r.Context(), db)

	if errDel := taxService.Delete(id); errDel != nil {
		http.Error(w, "error delete tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.SERVICE, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.SERVICE, db, id), 0)

	w.WriteHeader(http.StatusOK)
}

func BulkDeleteServices(w http.ResponseWriter, r *http.Request) {
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

	serviceService := services.NewServiceSvcs(r.Context(), db)

	var errors []string
	successCount := 0

	for _, id := range data.Ids {
		if err := serviceService.Delete(id); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to delete service %s: %s", id, err.Error()))
		} else {
			successCount++
			caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.SERVICE, db, id), 0)
		}
	}

	// Clear cache for all services
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.SERVICE, db), 0)

	response := BulkDeleteResponse{
		SuccessCount: successCount,
		TotalCount:   len(data.Ids),
		Errors:       errors,
	}

	j, _ := json.MarshalContext(r.Context(), response)
	w.Write(j)
}

func RedeemService(w http.ResponseWriter, r *http.Request) {
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	service := services.NewServiceSvcs(r.Context(), db)

	var data models.RedeemRequest

	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	userId := r.Context().Value(constants.CtxAuthKey).(common.Credentials)
	_id := userId.User.Id
	isUsed := false
	if data.UserId != "" {
		_id = data.UserId
		isUsed = true
	}
	wallet, err := service.Redeem(data, _id, isUsed)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.UserWallet, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.UserWallet, db, wallet.Id.String()), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.VOUCHER, db), 0)

	j, _ := json.MarshalContext(r.Context(), wallet)
	w.Write(j)
}

func Reorder(w http.ResponseWriter, r *http.Request) {
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	serviceId := chi.URLParam(r, "serviceid")

	service := services.NewServiceSvcs(r.Context(), db)

	var data models.ServiceReorder
	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	//userId := r.Context().Value(constants.CtxAuthKey).(common.Credentials)

	if err := service.Reorder(serviceId, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.SERVICE, db), 0)

	w.WriteHeader(http.StatusOK)
}
