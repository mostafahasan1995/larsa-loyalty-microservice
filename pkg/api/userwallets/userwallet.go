package userwallets

import (
	"fmt"
	"loyalty/pkg/caching"
	"loyalty/pkg/constants"
	"loyalty/pkg/models"
	"loyalty/pkg/services"
	"loyalty/pkg/util"
	"net/http"
	"strconv"
	"strings"

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

func GetUsersWallets(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	rawQuery := r.URL.RawQuery
	path := r.URL.Path

	cacheKey := util.CacheKeyForMulti(constants.UserWallet, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		args, err := util.Paginate(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		userwalletService := services.NewUserWalletSvcs(r.Context(), db)

		result, err := userwalletService.Get(args)

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

func GetAllUsersWallets(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	//

	rawQuery := r.URL.RawQuery
	path := r.URL.Path
	query := r.URL.Query().Get("query")

	cacheKey := util.CacheKeyForMulti(constants.UserWallet, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		userwalletService := services.NewUserWalletSvcs(r.Context(), db)
		result, err := userwalletService.GetAll(query)

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

func VerifyCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	user := r.Context().Value(constants.CtxAuthKey).(common.Credentials)
	db := headerParams.Client

	var data models.VerifyCodeRequest

	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	userwalletService := services.NewUserWalletSvcs(r.Context(), db)
	res, err := userwalletService.VerifyCode(data.Code, user.User.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.UserWallet, db), 0)

	j, _ := json.MarshalContext(r.Context(), res)

	w.Write(j)
}

func CreateInvitation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	user := r.Context().Value(constants.CtxAuthKey).(common.Credentials)
	db := headerParams.Client

	userwalletService := services.NewUserWalletSvcs(r.Context(), db)
	res, err := userwalletService.CreateInvitation(user.User.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.UserWallet, db), 0)

	j, _ := json.MarshalContext(r.Context(), res)

	w.Write(j)
}

func GetWalletById(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	cacheKey := util.CacheKeyForOne(constants.UserWallet, db, id, lang)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		userwalletService := services.NewUserWalletSvcs(r.Context(), db)
		result, err := userwalletService.GetOne(id)

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

func CreateUserWallet(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var data models.UserWallet

	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}
	data.VisitCount = 1

	userwalletService := services.NewUserWalletSvcs(r.Context(), db)
	res, errIns := userwalletService.Add(&data)
	if errIns != nil {
		http.Error(w, errIns.Error(), http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.UserWallet, db), 0)

	j, _ := json.MarshalContext(r.Context(), res)

	w.Write(j)
}

func DeleteUserWallet(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	userwalletService := services.NewUserWalletSvcs(r.Context(), db)

	if errDel := userwalletService.Delete(id); errDel != nil {
		http.Error(w, "error delete tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.UserWallet, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.UserWallet, db, id), 0)

	w.WriteHeader(http.StatusOK)
}

func BulkDeleteUserWallets(w http.ResponseWriter, r *http.Request) {
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

	userwalletService := services.NewUserWalletSvcs(r.Context(), db)

	var errors []string
	successCount := 0

	for _, id := range data.Ids {
		if err := userwalletService.Delete(id); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to delete user wallet %s: %s", id, err.Error()))
		} else {
			successCount++
			caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.UserWallet, db, id), 0)
		}
	}

	// Clear cache for all user wallets
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.UserWallet, db), 0)

	response := BulkDeleteResponse{
		SuccessCount: successCount,
		TotalCount:   len(data.Ids),
		Errors:       errors,
	}

	j, _ := json.MarshalContext(r.Context(), response)
	w.Write(j)
}

func GetUserWallet(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	user := r.Context().Value(constants.CtxAuthKey).(common.Credentials)

	userwalletService := services.NewUserWalletSvcs(r.Context(), db)

	result, err := userwalletService.GetWalletByUserId(user.User.Id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)

}

func GetUserWalletById(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	userwalletService := services.NewUserWalletSvcs(r.Context(), db)

	result, err := userwalletService.GetWalletByUserId(id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)

}

func BuyPoints(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	var data models.UserWalletBuyPointsRequest

	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	userwalletService := services.NewUserWalletSvcs(r.Context(), db)
	res, err := userwalletService.BuyPoints(id, data.MoneyAmount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.UserWallet, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.UserWallet, db, id), 0)

	j, _ := json.MarshalContext(r.Context(), res)

	w.Write(j)
}

func GetPointsPerAED(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	moneyAmountStr := r.URL.Query().Get("money_amount")
	moneyAmount, err := strconv.ParseFloat(strings.TrimSpace(moneyAmountStr), 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userwalletService := services.NewUserWalletSvcs(r.Context(), db)
	res := userwalletService.PointsPerAED(moneyAmount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := json.MarshalContext(r.Context(), res)

	w.Write(j)
}

func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	user := r.Context().Value(constants.CtxAuthKey).(common.Credentials)

	userwalletService := services.NewUserWalletSvcs(r.Context(), db)
	result, err := userwalletService.UpdateProfile(user.User.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.UserWallet, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.UserWallet, db, result.Id.Hex()), 0)

	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}

func VisitUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	userwalletService := services.NewUserWalletSvcs(r.Context(), db)
	err := userwalletService.VisitUser(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := json.MarshalContext(r.Context(), true)

	w.Write(j)
}
