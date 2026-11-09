package quizzes

import (
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

func GetQuizzes(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	rawQuery := r.URL.RawQuery
	path := r.URL.Path

	cacheKey := util.CacheKeyForMulti(constants.QUIZ, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		args, err := util.Paginate(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		quizService := services.NewQuizSvcs(r.Context(), db)

		result, err := quizService.Get(args, nil, false)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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

func GetAllQuizzes(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	rawQuery := r.URL.RawQuery
	path := r.URL.Path
	query := r.URL.Query().Get("query")

	cacheKey := util.CacheKeyForMulti(constants.QUIZ, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		quizService := services.NewQuizSvcs(r.Context(), db)
		result, err := quizService.GetAll(query)

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

func GetQuizById(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	cacheKey := util.CacheKeyForOne(constants.QUIZ, db, id, lang)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		quizService := services.NewQuizSvcs(r.Context(), db)
		result, err := quizService.GetOne(id)

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

func CreateQuiz(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var data models.Quiz

	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	quizService := services.NewQuizSvcs(r.Context(), db)

	if errIns := quizService.Add(&data); errIns != nil {
		http.Error(w, errIns.Error(), http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.QUIZ, db), 0)

	j, _ := json.MarshalContext(r.Context(), data)

	w.Write(j)
}

func UpdateQuiz(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	var data models.Quiz
	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	taxService := services.NewQuizSvcs(r.Context(), db)

	if errUpdate := taxService.Update(id, &data); errUpdate != nil {
		http.Error(w, "error update tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.QUIZ, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.QUIZ, db, id), 0)

	j, _ := json.MarshalContext(r.Context(), data)

	w.Write(j)
}

func DeleteQuiz(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	quizService := services.NewQuizSvcs(r.Context(), db)

	if errDel := quizService.Delete(id); errDel != nil {
		http.Error(w, "error delete tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.QUIZ, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.QUIZ, db, id), 0)

	w.WriteHeader(http.StatusOK)
}

func SubmitQuiz(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var payload *models.QuizAnswer

	if errDec := json.NewDecoder(r.Body).Decode(&payload); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	user := r.Context().Value(constants.CtxAuthKey).(common.Credentials)

	quizService := services.NewQuizSvcs(r.Context(), db)

	_userId, err := primitive.ObjectIDFromHex(user.User.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	payload.UserId = _userId

	result, err := quizService.Submit(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.QUIZ, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.QUIZ, db, payload.QuizId.Hex()), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.UserWallet, db), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func GetUserQuizzes(w http.ResponseWriter, r *http.Request) {
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
	quizService := services.NewQuizSvcs(r.Context(), db)

	result, err := quizService.Get(args, &_userId, true)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}

func GetUserAvailableQuizzes(w http.ResponseWriter, r *http.Request) {
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
	quizService := services.NewQuizSvcs(r.Context(), db)

	result, err := quizService.Get(args, &_userId, false)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}

func GetUserQuizzesById(w http.ResponseWriter, r *http.Request) {
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
	quizService := services.NewQuizSvcs(r.Context(), db)

	result, err := quizService.Get(args, &_userId, true)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}
