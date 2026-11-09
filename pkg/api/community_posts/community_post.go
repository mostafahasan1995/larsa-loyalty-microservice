package communityPosts

import (
	"context"
	"fmt"
	"log"
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

func GetCommunityPost(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	rawQuery := r.URL.RawQuery
	path := r.URL.Path

	cacheKey := util.CacheKeyForMulti(constants.COMMUNITY_POST, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		args, err := util.Paginate(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		communityPostService := services.NewCommunityPostSvc(r.Context(), db)

		result, err := communityPostService.Get(args, nil)

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

func GetAllCommunityPost(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	//

	rawQuery := r.URL.RawQuery
	path := r.URL.Path
	query := r.URL.Query().Get("query")

	cacheKey := util.CacheKeyForMulti(constants.COMMUNITY_POST, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		communityPostService := services.NewCommunityPostSvc(r.Context(), db)
		result, err := communityPostService.GetAll(query)

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

func GetCommunityPostById(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	cacheKey := util.CacheKeyForOne(constants.COMMUNITY_POST, db, id, lang)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		communityPostService := services.NewCommunityPostSvc(r.Context(), db)
		result, err := communityPostService.GetOne(id)

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

func CreateCommunityPost(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var data models.CommunityPost

	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	if data.UserId.IsZero() {
		user := r.Context().Value(constants.CtxAuthKey).(common.Credentials)
		userId, _ := primitive.ObjectIDFromHex(user.User.Id)
		data.UserId = userId
	}

	communityPostService := services.NewCommunityPostSvc(r.Context(), db)
	res, errIns := communityPostService.Add(&data)
	if errIns != nil {
		http.Error(w, errIns.Error(), http.StatusInternalServerError)
		return
	}
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.COMMUNITY_POST, db), 0)

	// Trigger social moment notification directly via service layer
	go func() {
		if res != nil && !res.Id.IsZero() {
			// Extract user context
			userContext := r.Context().Value(constants.CtxAuthKey)
			if userContext == nil {
				log.Printf("Error: No user context found for social moment notification")
				return
			}

			user, ok := userContext.(common.Credentials)
			if !ok {
				log.Printf("Error: Invalid user context type for social moment notification")
				return
			}

			// Convert entity_id to ObjectID
			entityID := res.Id
			userObjectId, err := primitive.ObjectIDFromHex(user.User.Id)
			if err != nil {
				log.Printf("Error converting user ID to ObjectID: %v", err)
				return
			}

			// Get user name for message
			userName := user.User.FirstName
			if userName == "" {
				userName = "User"
			}

			// Create notification with same structure as API handler
			notification := models.DashboardNotification{
				Title: models.LocalizableString{
					Ar: "لحظة اجتماعية جديدة من الحكمة",
					En: "New Al Hikma Social Moment",
				},
				Message: models.LocalizableString{
					Ar: fmt.Sprintf("المريض %s شارك لحظة اجتماعية جديدة تحتاج مراجعة", userName),
					En: fmt.Sprintf("Patient %s shared new social moment requiring review", userName),
				},
				ActionType: "social_moments",
				EntityType: "community_post",
				EntityId:   entityID,
				TargetRole: "admin",
				Priority:   models.DashboardNotificationPriorityHigh,
				Status:     models.DashboardNotificationStatusPending,
				UserId:     userObjectId,
			}

			// Create a background context to avoid cancellation when HTTP request completes
			ctx := context.Background()
			dashboardNotificationService := services.NewDashboardNotificationSvc(ctx, db)
			_, err = dashboardNotificationService.Add(&notification)
			if err != nil {
				log.Printf("Error creating social moment notification: %v", err)
				return
			}

			// Invalidate cache
			caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)
			log.Printf("Successfully created social moment notification for post ID: %s", res.Id.Hex())
		}
	}()

	j, _ := json.MarshalContext(r.Context(), res)

	w.Write(j)
}

func UpdateCommunityPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	var data models.CommunityPost
	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	service := services.NewCommunityPostSvc(r.Context(), db)
	res, errUpdate := service.Update(id, &data)
	if errUpdate != nil {
		http.Error(w, "error update tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.COMMUNITY_POST, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.COMMUNITY_POST, db, id), 0)

	j, _ := json.MarshalContext(r.Context(), res)

	w.Write(j)
}

func DeleteCommunityPost(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	communityPostService := services.NewCommunityPostSvc(r.Context(), db)

	if errDel := communityPostService.Delete(id); errDel != nil {
		http.Error(w, "error delete tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.COMMUNITY_POST, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.COMMUNITY_POST, db, id), 0)

	w.WriteHeader(http.StatusOK)
}

func GetUserCommunityPosts(w http.ResponseWriter, r *http.Request) {
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
	communityPostService := services.NewCommunityPostSvc(r.Context(), db)

	result, err := communityPostService.Get(args, &_userId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}

func GetUserCommunityPostsById(w http.ResponseWriter, r *http.Request) {
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
	communityPostService := services.NewCommunityPostSvc(r.Context(), db)

	result, err := communityPostService.Get(args, &_userId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}

func React(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	communityPostService := services.NewCommunityPostSvc(r.Context(), db)

	result, err := communityPostService.React(id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.COMMUNITY_POST, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.COMMUNITY_POST, db, id), 0)

	w.Write(j)
}

func UnReact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	communityPostService := services.NewCommunityPostSvc(r.Context(), db)

	result, err := communityPostService.UnReact(id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.COMMUNITY_POST, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.COMMUNITY_POST, db, id), 0)

	w.Write(j)
}
