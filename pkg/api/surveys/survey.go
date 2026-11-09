package surveys

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

func GetSurvey(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	rawQuery := r.URL.RawQuery
	path := r.URL.Path

	cacheKey := util.CacheKeyForMulti(constants.SURVEY, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		args, err := util.Paginate(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		surveyService := services.NewSurveysvcs(r.Context(), db)

		result, err := surveyService.Get(args, nil, false, false)

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

func GetAllSurvey(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	//

	rawQuery := r.URL.RawQuery
	path := r.URL.Path
	query := r.URL.Query().Get("query")

	cacheKey := util.CacheKeyForMulti(constants.SURVEY, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		surveyService := services.NewSurveysvcs(r.Context(), db)
		result, err := surveyService.GetAll(query)

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

func GetSurveyById(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	cacheKey := util.CacheKeyForOne(constants.SURVEY, db, id, lang)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		surveyService := services.NewSurveysvcs(r.Context(), db)
		result, err := surveyService.GetOne(id)

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

func CreateSurvey(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var data models.Survey

	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	surveyService := services.NewSurveysvcs(r.Context(), db)

	if errIns := surveyService.Add(&data); errIns != nil {
		http.Error(w, errIns.Error(), http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.SURVEY, db), 0)

	j, _ := json.MarshalContext(r.Context(), data)

	w.Write(j)
}

func UpdateSurvey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	var data models.Survey
	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	service := services.NewSurveysvcs(r.Context(), db)
	res, err := service.Update(id, &data)
	if err != nil {
		http.Error(w, "error update tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.SURVEY, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.SURVEY, db, id), 0)

	j, _ := json.MarshalContext(r.Context(), res)

	w.Write(j)
}

func DeleteSurvey(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	surveyService := services.NewSurveysvcs(r.Context(), db)

	if errDel := surveyService.Delete(id); errDel != nil {
		http.Error(w, "error delete tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.SURVEY, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.SURVEY, db, id), 0)

	w.WriteHeader(http.StatusOK)
}

func SubmitSurvey(w http.ResponseWriter, r *http.Request) {

	caching.Rdb.FlushAll()
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var data models.SurveyAnswer

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

	surveyService := services.NewSurveysvcs(r.Context(), db)

	data.UserId = _id

	result, err := surveyService.Submit(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.SURVEY, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.SURVEY_ANSWER, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.SURVEY, db, data.SurveyId.Hex()), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.UserWallet, db), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func GetUserSurveys(w http.ResponseWriter, r *http.Request) {
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
	surveyService := services.NewSurveysvcs(r.Context(), db)
	//with archived
	result, err := surveyService.Get(args, &_userId, true, false)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}

func GetUserAvailableSurveys(w http.ResponseWriter, r *http.Request) {
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
	surveyService := services.NewSurveysvcs(r.Context(), db)
	//without archive
	result, err := surveyService.Get(args, &_userId, false, true)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}

func GetUserSurveysById(w http.ResponseWriter, r *http.Request) {
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
	surveyService := services.NewSurveysvcs(r.Context(), db)
	//with archive
	result, err := surveyService.Get(args, &_userId, true, false)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}
