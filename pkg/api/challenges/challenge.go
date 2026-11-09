package challenges

import (
	"fmt"
	"loyalty/pkg/caching"
	"loyalty/pkg/constants"
	"loyalty/pkg/models"
	"loyalty/pkg/services"
	"loyalty/pkg/util"
	"net/http"

	"github.com/goccy/go-json"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"github.com/go-chi/chi"
)

func GetChallenge(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	rawQuery := r.URL.RawQuery
	path := r.URL.Path

	cacheKey := util.CacheKeyForMulti(constants.CHALLENGE, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		args, err := util.Paginate(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		challengeService := services.NewChallengeSvcs(r.Context(), db)

		result, err := challengeService.Get(args, nil, false)

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

func GetAllChallenge(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	//

	rawQuery := r.URL.RawQuery
	path := r.URL.Path
	query := r.URL.Query().Get("query")

	cacheKey := util.CacheKeyForMulti(constants.CHALLENGE, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		challengeService := services.NewChallengeSvcs(r.Context(), db)
		result, err := challengeService.GetAll(query)

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

func GetChallengeById(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	cacheKey := util.CacheKeyForOne(constants.CHALLENGE, db, id, lang)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		challengeService := services.NewChallengeSvcs(r.Context(), db)
		result, err := challengeService.GetOne(id)

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

func CreateChallenge(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var data models.Challenge

	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	challengeService := services.NewChallengeSvcs(r.Context(), db)

	if errIns := challengeService.Add(&data); errIns != nil {
		http.Error(w, errIns.Error(), http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.CHALLENGE, db), 0)

	j, _ := json.MarshalContext(r.Context(), data)

	w.Write(j)
}

func UpdateChallenge(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	var data models.Challenge
	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	service := services.NewChallengeSvcs(r.Context(), db)

	if errUpdate := service.Update(id, &data); errUpdate != nil {
		http.Error(w, "error update tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.CHALLENGE, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.CHALLENGE, db, id), 0)

	j, _ := json.MarshalContext(r.Context(), data)

	w.Write(j)
}

func DeleteChallenge(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	challengeService := services.NewChallengeSvcs(r.Context(), db)

	if errDel := challengeService.Delete(id); errDel != nil {
		http.Error(w, "error delete tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.CHALLENGE, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.CHALLENGE, db, id), 0)

	w.WriteHeader(http.StatusOK)
}

func SubmitChallenge(w http.ResponseWriter, r *http.Request) {

	caching.Rdb.FlushAll()
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var data models.ChallengeAnswer

	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	user := r.Context().Value(constants.CtxAuthKey).(common.Credentials)

	_id, err := primitive.ObjectIDFromHex(user.User.Id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	challengeService := services.NewChallengeSvcs(r.Context(), db)

	data.UserId = _id

	result, err := challengeService.Submit(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.CHALLENGE, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.CHALLENGE_ANSWER, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.UserWallet, db), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func GetUserChallenges(w http.ResponseWriter, r *http.Request) {
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
	challengeService := services.NewChallengeSvcs(r.Context(), db)

	result, err := challengeService.Get(args, &_userId, true)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}

func GetUserAvailableChallenges(w http.ResponseWriter, r *http.Request) {
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
	challengeService := services.NewChallengeSvcs(r.Context(), db)

	result, err := challengeService.Get(args, &_userId, false)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}

func GetUserChallengesById(w http.ResponseWriter, r *http.Request) {
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
	challengeService := services.NewChallengeSvcs(r.Context(), db)

	result, err := challengeService.Get(args, &_userId, true)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}
