package users

import (
	"context"
	"fmt"
	"loyalty/pkg/caching"
	"loyalty/pkg/constants"
	"loyalty/pkg/models"
	"loyalty/pkg/services"
	"loyalty/pkg/util"
	"net/http"

	"github.com/goccy/go-json"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
)

// UserWithPendingPoints represents a user with their wallet and pending points count
type UserWithPendingPoints struct {
	User          interface{} `json:"user"`
	Wallet        interface{} `json:"wallet"`
	PendingPoints int         `json:"pendingPoints"`
}

// UsersWithPendingPointsResponse represents the response structure
type UsersWithPendingPointsResponse struct {
	Users      []UserWithPendingPoints `json:"users"`
	Pagination models.Pagination       `json:"pagination"`
}

func GetUsersWithPendingPoints(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	rawQuery := r.URL.RawQuery
	path := r.URL.Path

	cacheKey := util.CacheKeyForMulti(constants.USERS_WITH_PENDING_POINTS, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		// Get pagination parameters
		args, err := util.Paginate(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Get filter parameters
		pendingOnly := r.URL.Query().Get("pendingOnly") == "true"

		// Add pending points filter
		minPendingPoints := r.URL.Query().Get("minPendingPoints")
		maxPendingPoints := r.URL.Query().Get("maxPendingPoints")

		// Get sort parameters with defaults
		sortBy := r.URL.Query().Get("sortBy")
		if sortBy == "" {
			sortBy = "pendingPoints" // Default sort by pending points
		}

		sortOrder := r.URL.Query().Get("sortOrder")
		if sortOrder == "" {
			sortOrder = "desc" // Default sort order is descending (max to min)
		}

		// Get user field filter parameters
		firstName := r.URL.Query().Get("firstName")
		lastName := r.URL.Query().Get("lastName")
		gender := r.URL.Query().Get("gender")
		email := r.URL.Query().Get("email")

		// Create context with access token
		ctx := r.Context()
		if accessToken := r.Header.Get("x-access-token"); accessToken != "" {
			ctx = context.WithValue(ctx, "x-access-token", accessToken)
		} else if auth := r.Header.Get("Authorization"); auth != "" {
			ctx = context.WithValue(ctx, "Authorization", auth)
		}

		// Get users service
		usersService := services.NewUsersService(ctx, db)
		result, err := usersService.GetUsersWithPendingPoints(db, args, pendingOnly, sortBy, sortOrder, minPendingPoints, maxPendingPoints, firstName, lastName, gender, email)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		j, _ := json.MarshalContext(r.Context(), result)
		// Add to cache
		caching.CacheByteByKey(cacheKey, j, 0)

		w.Write(j)
	} else {
		fmt.Println("serve from cache")
		w.Write([]byte(cache))
	}
}

func GetAllUsersWithPendingPoints(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	rawQuery := r.URL.RawQuery
	path := r.URL.Path
	query := r.URL.Query().Get("query")

	cacheKey := util.CacheKeyForMulti(constants.USERS_WITH_PENDING_POINTS_ALL, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		// Get filter parameters
		pendingOnly := r.URL.Query().Get("pendingOnly") == "true"

		// Add pending points filter
		minPendingPoints := r.URL.Query().Get("minPendingPoints")
		maxPendingPoints := r.URL.Query().Get("maxPendingPoints")

		// Get sort parameters with defaults
		sortBy := r.URL.Query().Get("sortBy")
		if sortBy == "" {
			sortBy = "pendingPoints" // Default sort by pending points
		}

		sortOrder := r.URL.Query().Get("sortOrder")
		if sortOrder == "" {
			sortOrder = "desc" // Default sort order is descending (max to min)
		}

		// Get user field filter parameters
		firstName := r.URL.Query().Get("firstName")
		lastName := r.URL.Query().Get("lastName")
		gender := r.URL.Query().Get("gender")
		email := r.URL.Query().Get("email")

		// Create context with access token
		ctx := r.Context()
		if accessToken := r.Header.Get("x-access-token"); accessToken != "" {
			ctx = context.WithValue(ctx, "x-access-token", accessToken)
		} else if auth := r.Header.Get("Authorization"); auth != "" {
			ctx = context.WithValue(ctx, "Authorization", auth)
		}

		// Get users service
		usersService := services.NewUsersService(ctx, db)
		result, err := usersService.GetAllUsersWithPendingPoints(db, query, pendingOnly, sortBy, sortOrder, minPendingPoints, maxPendingPoints, firstName, lastName, gender, email)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		j, _ := json.MarshalContext(r.Context(), result)
		// Add to cache
		caching.CacheByteByKey(cacheKey, j, 0)

		w.Write(j)
	} else {
		w.Write([]byte(cache))
	}
}

// ClearUsersCache clears all users-related cache
func ClearUsersCache(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	// Clear users with pending points cache
	usersPattern := util.CacheKeyDeleteMul(constants.USERS_WITH_PENDING_POINTS, db)
	usersAllPattern := util.CacheKeyDeleteMul(constants.USERS_WITH_PENDING_POINTS_ALL, db)

	// Delete cache patterns
	caching.Rdb.BatchDeletionKeysByPattern(usersPattern, 0)
	caching.Rdb.BatchDeletionKeysByPattern(usersAllPattern, 0)

	response := map[string]string{
		"message": "Users cache cleared successfully",
	}

	j, _ := json.MarshalContext(r.Context(), response)
	w.Write(j)
}
