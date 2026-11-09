package services

import (
	"context"
	"fmt"
	"loyalty/pkg/caching"
	"loyalty/pkg/constants"
	"loyalty/pkg/database"
	"loyalty/pkg/models"
	"loyalty/pkg/util"
	"math"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DashboardNotificationService interface {
	GetOne(id string) (*models.DashboardNotificationResponse, error)
	Get(args *util.ListArgs, targetRole string) (*models.DashboardNotificationWithPagination, error)
	GetPending(targetRole string, limit int64) (*models.DashboardNotificationWithPagination, error)
	GetStats(targetRole string) (*models.DashboardNotificationStats, error)
	Add(data *models.DashboardNotification) (*models.DashboardNotificationResponse, error)
	Update(id string, data *models.DashboardNotification) (*models.DashboardNotificationResponse, error)
	MarkAsAcknowledged(id string) error
	MarkAsResolved(id string) error
	BulkMarkAsRead(ids []string) error
	Delete(id string) error
	GetUnreadCount(targetRole string) (int64, error)
	CleanupOldNotifications(olderThanDays int) error
}

type dashboardNotificationSvc struct {
	MainService[*models.DashboardNotification, *models.DashboardNotificationResponse]
}

func NewDashboardNotificationSvc(ctx context.Context, db string) DashboardNotificationService {
	return &dashboardNotificationSvc{
		MainService: MainService[*models.DashboardNotification, *models.DashboardNotificationResponse]{
			ctx:      ctx,
			db:       db,
			collName: "loyaltyDashboardNotifications",
		},
	}
}

func (s *dashboardNotificationSvc) GetOne(id string) (*models.DashboardNotificationResponse, error) {
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	coll := database.Mdb.Database(s.db).Collection(s.collName)
	pipeline := []bson.M{
		{"$match": bson.M{"_id": _id, "trash": bson.M{"$ne": true}}},
		{"$lookup": bson.M{
			"from":         "users",
			"localField":   "user_id",
			"foreignField": "_id",
			"as":           "user",
		}},
		{"$unwind": bson.M{"path": "$user", "preserveNullAndEmptyArrays": true}},
	}

	cursor, err := coll.Aggregate(s.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	var result models.DashboardNotificationResponse
	if cursor.Next(s.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
	} else {
		return nil, mongo.ErrNoDocuments
	}

	return &result, nil
}

func (s *dashboardNotificationSvc) Get(args *util.ListArgs, targetRole string) (*models.DashboardNotificationWithPagination, error) {
	coll := database.Mdb.Database(s.db).Collection(s.collName)

	// Calculate limit and skip from PerPage and Page
	limit, _ := strconv.ParseInt(args.PerPage, 10, 64)
	pageNum, _ := strconv.ParseInt(args.Page, 10, 32)
	skip := (pageNum - 1) * limit

	filter := bson.M{"trash": bson.M{"$ne": true}}
	if targetRole != "" {
		filter["$or"] = []bson.M{
			{"target_role": targetRole},
			{"target_role": models.DashboardNotificationTargetBoth},
		}
	}

	if args.Search != "" {
		filter["$or"] = []bson.M{
			{"title.en": bson.M{"$regex": args.Search, "$options": "i"}},
			{"title.ar": bson.M{"$regex": args.Search, "$options": "i"}},
			{"message.en": bson.M{"$regex": args.Search, "$options": "i"}},
			{"message.ar": bson.M{"$regex": args.Search, "$options": "i"}},
			{"user_name": bson.M{"$regex": args.Search, "$options": "i"}},
		}
	}

	pipeline := []bson.M{
		{"$match": filter},
		{"$lookup": bson.M{
			"from":         "users",
			"localField":   "user_id",
			"foreignField": "_id",
			"as":           "user",
		}},
		{"$unwind": bson.M{"path": "$user", "preserveNullAndEmptyArrays": true}},
		{"$sort": bson.M{"createdAt": -1}},
	}

	if skip > 0 {
		pipeline = append(pipeline, bson.M{"$skip": skip})
	}
	if limit > 0 {
		pipeline = append(pipeline, bson.M{"$limit": limit})
	}

	cursor, err := coll.Aggregate(s.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	var notifications []models.DashboardNotificationResponse
	if err := cursor.All(s.ctx, &notifications); err != nil {
		return nil, err
	}

	totalCount, err := coll.CountDocuments(s.ctx, filter)
	if err != nil {
		return nil, err
	}

	totalPages := float64(0)
	if limit > 0 {
		totalPages = math.Ceil(float64(totalCount) / float64(limit))
	}

	pagination := models.Pagination{
		TotalCount: totalCount,
		PerPage:    limit,
		TotalPages: totalPages,
	}

	return &models.DashboardNotificationWithPagination{
		Notifications: notifications,
		Pagination:    pagination,
	}, nil
}

func (s *dashboardNotificationSvc) GetPending(targetRole string, limit int64) (*models.DashboardNotificationWithPagination, error) {
	coll := database.Mdb.Database(s.db).Collection(s.collName)

	filter := bson.M{
		"status": models.DashboardNotificationStatusPending,
		"trash":  bson.M{"$ne": true},
	}

	if targetRole != "" {
		filter["$or"] = []bson.M{
			{"target_role": targetRole},
			{"target_role": models.DashboardNotificationTargetBoth},
		}
	}

	pipeline := []bson.M{
		{"$match": filter},
		{"$lookup": bson.M{
			"from":         "users",
			"localField":   "user_id",
			"foreignField": "_id",
			"as":           "user",
		}},
		{"$unwind": bson.M{"path": "$user", "preserveNullAndEmptyArrays": true}},
		{"$sort": bson.M{"priority": 1, "createdAt": -1}},
	}

	if limit > 0 {
		pipeline = append(pipeline, bson.M{"$limit": limit})
	}

	cursor, err := coll.Aggregate(s.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	var notifications []models.DashboardNotificationResponse
	if err := cursor.All(s.ctx, &notifications); err != nil {
		return nil, err
	}

	totalCount, err := coll.CountDocuments(s.ctx, filter)
	if err != nil {
		return nil, err
	}

	totalPages := float64(0)
	if limit > 0 {
		totalPages = math.Ceil(float64(totalCount) / float64(limit))
	}

	return &models.DashboardNotificationWithPagination{
		Notifications: notifications,
		Pagination: models.Pagination{
			TotalCount: totalCount,
			PerPage:    limit,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *dashboardNotificationSvc) GetStats(targetRole string) (*models.DashboardNotificationStats, error) {
	coll := database.Mdb.Database(s.db).Collection(s.collName)

	baseFilter := bson.M{
		"status": models.DashboardNotificationStatusPending,
		"trash":  bson.M{"$ne": true},
	}

	if targetRole != "" {
		baseFilter["$or"] = []bson.M{
			{"target_role": targetRole},
			{"target_role": models.DashboardNotificationTargetBoth},
		}
	}

	pipeline := []bson.M{
		{"$match": baseFilter},
		{"$group": bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": 1},
			"high_priority": bson.M{"$sum": bson.M{"$cond": []interface{}{
				bson.M{"$eq": []string{"$priority", models.DashboardNotificationPriorityHigh}},
				1, 0,
			}}},
			"medium_priority": bson.M{"$sum": bson.M{"$cond": []interface{}{
				bson.M{"$eq": []string{"$priority", models.DashboardNotificationPriorityMedium}},
				1, 0,
			}}},
			"low_priority": bson.M{"$sum": bson.M{"$cond": []interface{}{
				bson.M{"$eq": []string{"$priority", models.DashboardNotificationPriorityLow}},
				1, 0,
			}}},
		}},
	}

	cursor, err := coll.Aggregate(s.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	stats := &models.DashboardNotificationStats{
		ByType: make(map[string]int64),
	}

	if cursor.Next(s.ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}

		stats.TotalPending = result["total"].(int64)
		stats.HighPriority = result["high_priority"].(int64)
		stats.MediumPriority = result["medium_priority"].(int64)
		stats.LowPriority = result["low_priority"].(int64)
	}

	typesPipeline := []bson.M{
		{"$match": baseFilter},
		{"$group": bson.M{
			"_id":   "$action_type",
			"count": bson.M{"$sum": 1},
		}},
	}

	typesCursor, err := coll.Aggregate(s.ctx, typesPipeline)
	if err != nil {
		return nil, err
	}
	defer typesCursor.Close(s.ctx)

	for typesCursor.Next(s.ctx) {
		var typeResult bson.M
		if err := typesCursor.Decode(&typeResult); err != nil {
			continue
		}
		actionType := typeResult["_id"].(string)
		count := typeResult["count"].(int64)
		stats.ByType[actionType] = count
	}

	return stats, nil
}

func (s *dashboardNotificationSvc) Add(data *models.DashboardNotification) (*models.DashboardNotificationResponse, error) {
	coll := database.Mdb.Database(s.db).Collection(s.collName)

	data.Id = primitive.NewObjectID()
	data.Status = models.DashboardNotificationStatusPending
	data.SetCreatedBy(data.UserId)

	_, err := coll.InsertOne(s.ctx, data)
	if err != nil {
		return nil, err
	}

	return s.GetOne(data.Id.Hex())
}

func (s *dashboardNotificationSvc) Update(id string, data *models.DashboardNotification) (*models.DashboardNotificationResponse, error) {
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	coll := database.Mdb.Database(s.db).Collection(s.collName)
	filter := bson.M{"_id": _id, "trash": bson.M{"$ne": true}}
	update := bson.M{"$set": data}

	data.SetUpdatedBy(data.UserId)

	_, err = coll.UpdateOne(s.ctx, filter, update)
	if err != nil {
		return nil, err
	}

	return s.GetOne(id)
}

func (s *dashboardNotificationSvc) MarkAsAcknowledged(id string) error {
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	coll := database.Mdb.Database(s.db).Collection(s.collName)
	filter := bson.M{"_id": _id, "trash": bson.M{"$ne": true}}
	now := time.Now()
	update := bson.M{"$set": bson.M{
		"status":  models.DashboardNotificationStatusAcknowledged,
		"read_at": now,
	}}

	_, err = coll.UpdateOne(s.ctx, filter, update)
	return err
}

func (s *dashboardNotificationSvc) MarkAsResolved(id string) error {
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	coll := database.Mdb.Database(s.db).Collection(s.collName)
	filter := bson.M{"_id": _id, "trash": bson.M{"$ne": true}}
	now := time.Now()
	update := bson.M{"$set": bson.M{
		"status":      models.DashboardNotificationStatusResolved,
		"resolved_at": now,
	}}

	_, err = coll.UpdateOne(s.ctx, filter, update)
	return err
}

func (s *dashboardNotificationSvc) BulkMarkAsRead(ids []string) error {
	objectIds := make([]primitive.ObjectID, len(ids))
	for i, id := range ids {
		_id, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return err
		}
		objectIds[i] = _id
	}

	coll := database.Mdb.Database(s.db).Collection(s.collName)
	filter := bson.M{"_id": bson.M{"$in": objectIds}, "trash": bson.M{"$ne": true}}
	now := time.Now()
	update := bson.M{"$set": bson.M{
		"status":  models.DashboardNotificationStatusAcknowledged,
		"read_at": now,
	}}

	_, err := coll.UpdateMany(s.ctx, filter, update)
	return err
}

func (s *dashboardNotificationSvc) Delete(id string) error {
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	coll := database.Mdb.Database(s.db).Collection(s.collName)
	filter := bson.M{"_id": _id}
	update := bson.M{"$set": bson.M{"trash": true}}

	_, err = coll.UpdateOne(s.ctx, filter, update)
	return err
}

func (s *dashboardNotificationSvc) GetUnreadCount(targetRole string) (int64, error) {
	coll := database.Mdb.Database(s.db).Collection(s.collName)

	filter := bson.M{
		"status": models.DashboardNotificationStatusPending,
		"trash":  bson.M{"$ne": true},
	}

	if targetRole != "" {
		filter["$or"] = []bson.M{
			{"target_role": targetRole},
			{"target_role": models.DashboardNotificationTargetBoth},
		}
	}

	return coll.CountDocuments(s.ctx, filter)
}

func (s *dashboardNotificationSvc) CleanupOldNotifications(olderThanDays int) error {
	coll := database.Mdb.Database(s.db).Collection(s.collName)
	cutoffDate := time.Now().AddDate(0, 0, -olderThanDays)

	filter := bson.M{
		"status": bson.M{"$in": []string{
			models.DashboardNotificationStatusAcknowledged,
			models.DashboardNotificationStatusResolved,
		}},
		"createdAt": bson.M{"$lt": cutoffDate},
	}

	update := bson.M{"$set": bson.M{"trash": true}}
	_, err := coll.UpdateMany(s.ctx, filter, update)
	return err
}

func CreateDashboardNotification(ctx context.Context, db string, notification *models.DashboardNotification) error {
	service := NewDashboardNotificationSvc(ctx, db)
	_, err := service.Add(notification)
	if err != nil {
		return err
	}

	// Add cache invalidation after successful operation
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.DASHBOARD_NOTIFICATIONS, db), 0)

	return nil
}

func NotifyDashboardOnBooking(clientId, userId, userName, bookingId, serviceType string) {
	ctx := context.Background()
	var titleAr, titleEn, messageAr, messageEn string
	var priority string

	switch serviceType {
	case models.DashboardNotificationActionSmileTestDrive:
		titleAr = "حجز  جديدة"
		titleEn = "New  Booking"
		messageAr = fmt.Sprintf("المريض %s حجز موعد ويحتاج تأكيد", userName)
		messageEn = fmt.Sprintf("Patient %s booked t and needs confirmation", userName)
		priority = models.DashboardNotificationPriorityHigh
	case models.DashboardNotificationActionConsultation:
		titleAr = "حجز استشارة أونلاين جديدة"
		titleEn = "New Online Consultation Booking"
		messageAr = fmt.Sprintf("المريض %s حجز استشارة أونلاين ويحتاج تأكيد", userName)
		messageEn = fmt.Sprintf("Patient %s booked online consultation and needs confirmation", userName)
		priority = models.DashboardNotificationPriorityHigh
	default:
		titleAr = "حجز موعد جديد"
		titleEn = "New Appointment Booking"
		messageAr = fmt.Sprintf("المريض %s حجز موعد جديد ويحتاج تأكيد", userName)
		messageEn = fmt.Sprintf("Patient %s made new appointment booking and needs confirmation", userName)
		priority = models.DashboardNotificationPriorityHigh
	}

	notification := &models.DashboardNotification{
		Title: models.LocalizableString{
			Ar: titleAr,
			En: titleEn,
		},
		Message: models.LocalizableString{
			Ar: messageAr,
			En: messageEn,
		},
		ActionType: serviceType,
		EntityType: models.DashboardNotificationEntityBooking,
		Priority:   priority,
		TargetRole: models.DashboardNotificationTargetBoth,
		UserName:   userName,
		Metadata: map[string]interface{}{
			"service_type":          serviceType,
			"requires_confirmation": true,
		},
	}

	if entityId, err := primitive.ObjectIDFromHex(bookingId); err == nil {
		notification.EntityId = entityId
	}
	if userObjectId, err := primitive.ObjectIDFromHex(userId); err == nil {
		notification.UserId = userObjectId
	}

	go CreateDashboardNotification(ctx, clientId, notification)

	util.LocalizeNotify(
		util.LocalizeNotificationMsg{
			XClient: clientId,
			Header: models.LocalizableString{
				Ar: titleAr,
				En: titleEn,
			},
			Content: models.LocalizableString{
				Ar: messageAr,
				En: messageEn,
			},
			TopicType: constants.NotificationTopicTypeDashboard,
			Topic:     "booking",
			Type:      "dashboard",
			Data: util.NotificationPayload{
				Id:    bookingId,
				Model: "booking",
			},
		},
	)
}

func NotifyDashboardOnSocialMoments(clientId string, postId string, userId string, userName string) {
	ctx := context.Background()

	// Use constant body structure matching the API handler
	notification := &models.DashboardNotification{
		Title: models.LocalizableString{
			Ar: "لحظة اجتماعية جديدة من الحكمة",
			En: "New Al Hikma Social Moment",
		},
		Message: models.LocalizableString{
			Ar: fmt.Sprintf("المريض %s شارك لحظة اجتماعية جديدة تحتاج مراجعة", userName),
			En: fmt.Sprintf("Patient %s shared new social moment requiring review", userName),
		},
		ActionType: models.DashboardNotificationActionSocialMoments,
		EntityType: models.DashboardNotificationEntityCommunityPost,
		TargetRole: models.DashboardNotificationTargetAdmin,
		Priority:   models.DashboardNotificationPriorityHigh,
		Status:     models.DashboardNotificationStatusPending,
		UserName:   userName,
		Metadata: map[string]interface{}{
			"post_type":           "social_moment",
			"requires_moderation": true,
		},
	}

	// Convert string IDs to ObjectIDs
	if entityId, err := primitive.ObjectIDFromHex(postId); err == nil {
		notification.EntityId = entityId
	}
	if userObjectId, err := primitive.ObjectIDFromHex(userId); err == nil {
		notification.UserId = userObjectId
	}

	// Create dashboard notification
	go CreateDashboardNotification(ctx, clientId, notification)

	// Send localized notification
	util.LocalizeNotify(
		util.LocalizeNotificationMsg{
			XClient: clientId,
			Header: models.LocalizableString{
				Ar: "لحظة اجتماعية جديدة من الحكمة",
				En: "New Al Hikma Social Moment",
			},
			Content: models.LocalizableString{
				Ar: fmt.Sprintf("المريض %s شارك لحظة اجتماعية جديدة", userName),
				En: fmt.Sprintf("Patient %s shared new social moment", userName),
			},
			TopicType: constants.NotificationTopicTypeAdmin,
			Topic:     "admin",
			Type:      "dashboard",
			Data: util.NotificationPayload{
				Id:    postId,
				Model: "community_post",
			},
		},
	)
}
