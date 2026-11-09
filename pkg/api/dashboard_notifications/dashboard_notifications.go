package dashboard_notifications

import (
	"fmt"
	"log"
	"loyalty/pkg/caching"
	"loyalty/pkg/constants"
	"loyalty/pkg/models"
	"loyalty/pkg/services"
	"loyalty/pkg/util"
	"net/http"
	"strconv"
	"time"

	"github.com/goccy/go-json"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"github.com/go-chi/chi"
)

func GetDashboardNotifications(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	rawQuery := r.URL.RawQuery
	path := r.URL.Path

	cacheKey := util.CacheKeyForMulti(constants.DASHBOARD_NOTIFICATIONS, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		args, err := util.Paginate(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		targetRole := r.URL.Query().Get("target_role")
		dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

		result, err := dashboardNotificationService.Get(args, targetRole)

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

func GetDashboardNotificationById(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	cacheKey := util.CacheKeyForOne(constants.DASHBOARD_NOTIFICATIONS, db, id, lang)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

		result, err := dashboardNotificationService.GetOne(id)

		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
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

func GetPendingDashboardNotifications(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	result, err := dashboardNotificationService.GetPending("", 50)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func GetDashboardNotificationStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	result, err := dashboardNotificationService.GetStats("")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	result, err := dashboardNotificationService.GetUnreadCount("")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func CreateDashboardNotification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var notification models.DashboardNotification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Extract userId from authentication token
	log.Printf("Extracting user context...")
	userContext := r.Context().Value(constants.CtxAuthKey)
	if userContext == nil {
		log.Printf("Error: No user context found")
		http.Error(w, "Authentication context not found", http.StatusUnauthorized)
		return
	}

	user, ok := userContext.(common.Credentials)
	if !ok {
		log.Printf("Error: Invalid user context type")
		http.Error(w, "Invalid authentication context", http.StatusUnauthorized)
		return
	}

	log.Printf("User ID from context: '%s'", user.User.Id)
	log.Printf("User object: %+v", user.User)
	log.Printf("Full credentials: %+v", user)

	if user.User.Id == "" {
		log.Printf("Error: User ID is empty in authentication context")
		http.Error(w, "User ID not found in authentication context", http.StatusUnauthorized)
		return
	}

	_id, err := primitive.ObjectIDFromHex(user.User.Id)
	if err != nil {
		log.Printf("Error converting user ID to ObjectID: %v", err)
		http.Error(w, "Invalid user ID format: "+err.Error(), http.StatusInternalServerError)
		return
	}
	notification.UserId = _id

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	result, err := dashboardNotificationService.Add(&notification)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

// CreateDashboardNotificationSimple - New endpoint that handles entity_id as string
func CreateDashboardNotificationSimple(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	// Define a struct for the incoming JSON with entity_id as string
	type NotificationRequest struct {
		EntityId   string            `json:"entity_id"`
		CreatedAt  string            `json:"created_at"`
		ClientId   string            `json:"client_id"`
		Title      map[string]string `json:"title"`
		Message    map[string]string `json:"message"`
		ActionType string            `json:"action_type"`
		EntityType string            `json:"entity_type"`
		Priority   string            `json:"priority"`
		TargetRole string            `json:"target_role"`
	}

	var req NotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert entity_id string to ObjectID
	entityObjectId, err := primitive.ObjectIDFromHex(req.EntityId)
	if err != nil {
		http.Error(w, "Invalid entity_id format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Parse created_at
	createdAt, err := time.Parse(time.RFC3339, req.CreatedAt)
	if err != nil {
		createdAt = time.Now() // fallback to current time
	}

	// Extract userId from authentication token
	user := r.Context().Value(constants.CtxAuthKey).(common.Credentials)
	userObjectId, err := primitive.ObjectIDFromHex(user.User.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert title and message maps to LocalizableString
	title := models.LocalizableString{
		En: req.Title["en"],
		Ar: req.Title["ar"],
	}
	message := models.LocalizableString{
		En: req.Message["en"],
		Ar: req.Message["ar"],
	}

	// Create the notification object
	notification := models.DashboardNotification{
		EntityId:   entityObjectId,
		Title:      title,
		Message:    message,
		ActionType: req.ActionType,
		EntityType: req.EntityType,
		Priority:   req.Priority,
		TargetRole: req.TargetRole,
		UserId:     userObjectId,
		Status:     models.DashboardNotificationStatusPending,
		BaseDbModel: models.BaseDbModel{
			CreatedAt: createdAt,
			UpdatedAt: time.Now(),
		},
	}

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	result, err := dashboardNotificationService.Add(&notification)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

// CreateBookingNotification - Create notification for new booking
func CreateBookingNotification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	// Log incoming request for debugging
	log.Printf("CreateBookingNotification called with entity_id: %s", chi.URLParam(r, "entity_id"))

	entityId := chi.URLParam(r, "entity_id")
	if entityId == "" {
		log.Printf("Error: entity_id is required")
		http.Error(w, "entity_id is required", http.StatusBadRequest)
		return
	}

	// Convert entity_id to ObjectID
	entityObjectId, err := primitive.ObjectIDFromHex(entityId)
	if err != nil {
		log.Printf("Error converting entity_id to ObjectID: %v", err)
		http.Error(w, "Invalid entity_id format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Extract userId from authentication token
	log.Printf("Extracting user context...")
	userContext := r.Context().Value(constants.CtxAuthKey)
	if userContext == nil {
		log.Printf("Error: No user context found")
		http.Error(w, "Authentication context not found", http.StatusUnauthorized)
		return
	}

	user, ok := userContext.(common.Credentials)
	if !ok {
		log.Printf("Error: Invalid user context type")
		http.Error(w, "Invalid authentication context", http.StatusUnauthorized)
		return
	}

	log.Printf("User ID from context: '%s'", user.User.Id)
	log.Printf("User object: %+v", user.User)

	if user.User.Id == "" {
		log.Printf("Error: User ID is empty in authentication context")
		http.Error(w, "User ID not found in authentication context", http.StatusUnauthorized)
		return
	}

	_id, err := primitive.ObjectIDFromHex(user.User.Id)
	if err != nil {
		log.Printf("Error converting user ID to ObjectID: %v", err)
		http.Error(w, "Invalid user ID format: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get user name for message replacement
	userName := user.User.FirstName
	if userName == "" {
		userName = user.User.FirstName
	}
	if userName == "" {
		userName = "User"
	}

	// Create notification with constant JSON structure
	notification := models.DashboardNotification{
		Title: models.LocalizableString{
			Ar: "حجز موعد جديد",
			En: "New Booking",
		},
		Message: models.LocalizableString{
			Ar: fmt.Sprintf("المريض %s حجز موعد تجربة ويحتاج تأكيد", userName),
			En: fmt.Sprintf("Patient %s booked appointment and needs confirmation", userName),
		},
		ActionType: "booking",
		EntityType: "booking",
		EntityId:   entityObjectId,
		TargetRole: "admin",
		Priority:   models.DashboardNotificationPriorityHigh,
		Status:     models.DashboardNotificationStatusPending,
		UserId:     _id,
	}

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	result, err := dashboardNotificationService.Add(&notification)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

// CreateOnlineConsultationNotification - Create notification for new online consultation
func CreateOnlineConsultationNotification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	// Log incoming request for debugging
	log.Printf("CreateOnlineConsultationNotification called with entity_id: %s", chi.URLParam(r, "entity_id"))

	entityId := chi.URLParam(r, "entity_id")
	if entityId == "" {
		log.Printf("Error: entity_id is required")
		http.Error(w, "entity_id is required", http.StatusBadRequest)
		return
	}

	// Convert entity_id to ObjectID
	entityObjectId, err := primitive.ObjectIDFromHex(entityId)
	if err != nil {
		log.Printf("Error converting entity_id to ObjectID: %v", err)
		http.Error(w, "Invalid entity_id format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Extract userId from authentication token
	log.Printf("Extracting user context...")
	userContext := r.Context().Value(constants.CtxAuthKey)
	if userContext == nil {
		log.Printf("Error: No user context found")
		http.Error(w, "Authentication context not found", http.StatusUnauthorized)
		return
	}

	user, ok := userContext.(common.Credentials)
	if !ok {
		log.Printf("Error: Invalid user context type")
		http.Error(w, "Invalid authentication context", http.StatusUnauthorized)
		return
	}

	log.Printf("User ID from context: '%s'", user.User.Id)
	log.Printf("User object: %+v", user.User)

	if user.User.Id == "" {
		log.Printf("Error: User ID is empty in authentication context")
		http.Error(w, "User ID not found in authentication context", http.StatusUnauthorized)
		return
	}

	_id, err := primitive.ObjectIDFromHex(user.User.Id)
	if err != nil {
		log.Printf("Error converting user ID to ObjectID: %v", err)
		http.Error(w, "Invalid user ID format: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get user name for message replacement
	userName := user.User.FirstName
	if userName == "" {
		userName = user.User.FirstName
	}
	if userName == "" {
		userName = "User"
	}

	// Create notification with constant JSON structure
	notification := models.DashboardNotification{
		Title: models.LocalizableString{
			Ar: "حجز استشارة أونلاين جديدة",
			En: "New Online Consultation Booking",
		},
		Message: models.LocalizableString{
			Ar: fmt.Sprintf("المريض %s حجز استشارة أونلاين ويحتاج تأكيد", userName),
			En: fmt.Sprintf("Patient %s booked online consultation and needs confirmation", userName),
		},
		ActionType: "consultation",
		EntityType: "consultation",
		EntityId:   entityObjectId,
		TargetRole: "admin",
		Priority:   models.DashboardNotificationPriorityHigh,
		Status:     models.DashboardNotificationStatusPending,
		UserId:     _id,
	}

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	result, err := dashboardNotificationService.Add(&notification)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func UpdateDashboardNotification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	var notification models.DashboardNotification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	notificationId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	notification.Id = notificationId

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	result, err := dashboardNotificationService.Update(id, &notification)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.DASHBOARD_NOTIFICATIONS, db, id), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func MarkAsAcknowledged(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	err := dashboardNotificationService.MarkAsAcknowledged(id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.DASHBOARD_NOTIFICATIONS, db, id), 0)

	j, _ := json.MarshalContext(r.Context(), map[string]string{"status": "acknowledged"})
	w.Write(j)
}

func MarkAsResolved(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	err := dashboardNotificationService.MarkAsResolved(id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.DASHBOARD_NOTIFICATIONS, db, id), 0)

	j, _ := json.MarshalContext(r.Context(), map[string]string{"status": "resolved"})
	w.Write(j)
}

func BulkMarkAsRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var request struct {
		IDs []string `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	err := dashboardNotificationService.BulkMarkAsRead(request.IDs)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)
	for _, id := range request.IDs {
		caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.DASHBOARD_NOTIFICATIONS, db, id), 0)
	}

	j, _ := json.MarshalContext(r.Context(), map[string]string{"status": "marked_as_read"})
	w.Write(j)
}

func DeleteDashboardNotification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	err := dashboardNotificationService.Delete(id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.DASHBOARD_NOTIFICATIONS, db, id), 0)

	w.WriteHeader(http.StatusNoContent)
}

func CleanupOldNotifications(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	daysStr := r.URL.Query().Get("days")
	days := 30 // default
	if daysStr != "" {
		if parsedDays, err := strconv.Atoi(daysStr); err == nil {
			days = parsedDays
		}
	}

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	err := dashboardNotificationService.CleanupOldNotifications(days)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)

	j, _ := json.MarshalContext(r.Context(), map[string]string{"status": "cleanup_completed"})
	w.Write(j)
}

func CreateSmileTestDriveNotification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	// Log incoming request for debugging
	log.Printf("CreateSmileTestDriveNotification called with entity_id: %s", chi.URLParam(r, "entity_id"))

	entityId := chi.URLParam(r, "entity_id")
	if entityId == "" {
		log.Printf("Error: entity_id is required")
		http.Error(w, "entity_id is required", http.StatusBadRequest)
		return
	}

	// Convert entity_id to ObjectID
	entityObjectId, err := primitive.ObjectIDFromHex(entityId)
	if err != nil {
		log.Printf("Error converting entity_id to ObjectID: %v", err)
		http.Error(w, "Invalid entity_id format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Create notification with constant JSON structure
	notification := models.DashboardNotification{
		Title: models.LocalizableString{
			Ar: "حجز تجربة ابتسامة جديدة",
			En: "New Smile Test Drive Booking",
		},
		Message: models.LocalizableString{
			Ar: "المريض [name] حجز موعد تجربة ابتسامة ويحتاج تأكيد",
			En: "Patient [name] booked smile test drive appointment and needs confirmation",
		},
		ActionType: "smile_test_drive",
		EntityType: "smiletestdrive",
		EntityId:   entityObjectId,
		TargetRole: "admin",
		Priority:   models.DashboardNotificationPriorityHigh,
		Status:     models.DashboardNotificationStatusPending,
	}

	// Extract userId from authentication token
	log.Printf("Extracting user context...")
	userContext := r.Context().Value(constants.CtxAuthKey)
	if userContext == nil {
		log.Printf("Error: No user context found")
		http.Error(w, "Authentication context not found", http.StatusUnauthorized)
		return
	}

	user, ok := userContext.(common.Credentials)
	if !ok {
		log.Printf("Error: Invalid user context type")
		http.Error(w, "Invalid authentication context", http.StatusUnauthorized)
		return
	}

	log.Printf("User ID from context: '%s'", user.User.Id)
	log.Printf("User object: %+v", user.User)
	log.Printf("Full credentials: %+v", user)

	if user.User.Id == "" {
		log.Printf("Error: User ID is empty in authentication context")
		http.Error(w, "User ID not found in authentication context", http.StatusUnauthorized)
		return
	}

	_id, err := primitive.ObjectIDFromHex(user.User.Id)
	if err != nil {
		log.Printf("Error converting user ID to ObjectID: %v", err)
		http.Error(w, "Invalid user ID format: "+err.Error(), http.StatusInternalServerError)
		return
	}
	notification.UserId = _id

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	result, err := dashboardNotificationService.Add(&notification)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

// CreateSocialMomentNotification - Create notification for new social moment
func CreateSocialMomentNotification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	// Log incoming request for debugging
	log.Printf("CreateSocialMomentNotification called with entity_id: %s", chi.URLParam(r, "entity_id"))

	entityId := chi.URLParam(r, "entity_id")
	if entityId == "" {
		log.Printf("Error: entity_id is required")
		http.Error(w, "entity_id is required", http.StatusBadRequest)
		return
	}

	// Convert entity_id to ObjectID
	entityObjectId, err := primitive.ObjectIDFromHex(entityId)
	if err != nil {
		log.Printf("Error converting entity_id to ObjectID: %v", err)
		http.Error(w, "Invalid entity_id format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Extract userId from authentication token
	log.Printf("Extracting user context...")
	userContext := r.Context().Value(constants.CtxAuthKey)
	if userContext == nil {
		log.Printf("Error: No user context found")
		http.Error(w, "Authentication context not found", http.StatusUnauthorized)
		return
	}

	user, ok := userContext.(common.Credentials)
	if !ok {
		log.Printf("Error: Invalid user context type")
		http.Error(w, "Invalid authentication context", http.StatusUnauthorized)
		return
	}

	log.Printf("User ID from context: '%s'", user.User.Id)
	log.Printf("User object: %+v", user.User)

	if user.User.Id == "" {
		log.Printf("Error: User ID is empty in authentication context")
		http.Error(w, "User ID not found in authentication context", http.StatusUnauthorized)
		return
	}

	_id, err := primitive.ObjectIDFromHex(user.User.Id)
	if err != nil {
		log.Printf("Error converting user ID to ObjectID: %v", err)
		http.Error(w, "Invalid user ID format: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get user name for message replacement
	userName := user.User.FirstName
	if userName == "" {
		userName = user.User.FirstName
	}
	if userName == "" {
		userName = "User"
	}

	// Create notification with constant JSON structure
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
		EntityId:   entityObjectId,
		TargetRole: "admin",
		Priority:   models.DashboardNotificationPriorityHigh,
		Status:     models.DashboardNotificationStatusPending,
		UserId:     _id,
	}

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	result, err := dashboardNotificationService.Add(&notification)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

// CreateVoucherRedemptionNotification - Create notification for voucher redemption
func CreateVoucherRedemptionNotification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	// Log incoming request for debugging
	log.Printf("CreateVoucherRedemptionNotification called with entity_id: %s", chi.URLParam(r, "entity_id"))

	entityId := chi.URLParam(r, "entity_id")
	if entityId == "" {
		log.Printf("Error: entity_id is required")
		http.Error(w, "entity_id is required", http.StatusBadRequest)
		return
	}

	// Convert entity_id to ObjectID
	entityObjectId, err := primitive.ObjectIDFromHex(entityId)
	if err != nil {
		log.Printf("Error converting entity_id to ObjectID: %v", err)
		http.Error(w, "Invalid entity_id format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Extract userId from authentication token
	log.Printf("Extracting user context...")
	userContext := r.Context().Value(constants.CtxAuthKey)
	if userContext == nil {
		log.Printf("Error: No user context found")
		http.Error(w, "Authentication context not found", http.StatusUnauthorized)
		return
	}

	user, ok := userContext.(common.Credentials)
	if !ok {
		log.Printf("Error: Invalid user context type")
		http.Error(w, "Invalid authentication context", http.StatusUnauthorized)
		return
	}

	log.Printf("User ID from context: '%s'", user.User.Id)
	log.Printf("User object: %+v", user.User)

	if user.User.Id == "" {
		log.Printf("Error: User ID is empty in authentication context")
		http.Error(w, "User ID not found in authentication context", http.StatusUnauthorized)
		return
	}

	_id, err := primitive.ObjectIDFromHex(user.User.Id)
	if err != nil {
		log.Printf("Error converting user ID to ObjectID: %v", err)
		http.Error(w, "Invalid user ID format: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get user name for message replacement
	userName := user.User.FirstName
	if userName == "" {
		userName = user.User.FirstName
	}
	if userName == "" {
		userName = "User"
	}

	// Create notification with constant JSON structure
	notification := models.DashboardNotification{
		Title: models.LocalizableString{
			Ar: "استرداد قسيمة جديدة",
			En: "New Voucher Redemption",
		},
		Message: models.LocalizableString{
			Ar: fmt.Sprintf("المريض %s استرد قسيمة ويحتاج مراجعة", userName),
			En: fmt.Sprintf("Patient %s redeemed voucher and needs review", userName),
		},
		ActionType: models.DashboardNotificationActionVoucherRedemption,
		EntityType: models.DashboardNotificationEntityVoucher,
		EntityId:   entityObjectId,
		TargetRole: models.DashboardNotificationTargetAdmin,
		Priority:   models.DashboardNotificationPriorityMedium,
		Status:     models.DashboardNotificationStatusPending,
		UserId:     _id,
	}

	dashboardNotificationService := services.NewDashboardNotificationSvc(r.Context(), db)

	result, err := dashboardNotificationService.Add(&notification)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}
