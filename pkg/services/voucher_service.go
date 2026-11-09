package services

import (
	"context"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/database"
	"loyalty/pkg/models"
	"loyalty/pkg/query"
	"loyalty/pkg/util"
	"strconv"
	"strings"
	"time"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type VoucherService interface {
	GetOne(id string) (*models.VoucherResponse, error)
	GetOneByCode(code string, userId, partnerId *primitive.ObjectID) (*models.VoucherResponse, error)
	Get(args *util.ListArgs, userId *primitive.ObjectID, partnerId *primitive.ObjectID) (*models.VoucherWithPagination, error)
	GetAll(query string) ([]*models.Voucher, error)
	Add(data *models.Voucher) (*models.VoucherResponse, error)
	Update(id string, data *models.Voucher) (*models.VoucherResponse, error)
	Use(code string, userId, partnerId *primitive.ObjectID) (*models.VoucherResponse, error)
	Reject(code string, userId, partnerId *primitive.ObjectID) (*models.VoucherResponse, error)
	Delete(id string) error
	VoucherExpiredTxn(voucher *models.Voucher) error
	GetForManagement(args *util.ListArgs) (*models.VoucherManagementWithPagination, error)
	UpdateStatus(id string, status string, comments string) (*models.VoucherResponse, error)
	FilterVouchers(filter map[string]interface{}, args *util.ListArgs) (*models.VoucherWithPagination, error)
	GetForManagementV2(conditions *query.Conditions, skip, limit int) (*models.VoucherManagementWithPagination, error)
	FixVoucherStatusData() error
}
type VoucherSvc struct {
	MainService[*models.Voucher, *models.VoucherResponse]
}

var voucherPipe = []bson.M{
	{"$lookup": bson.M{
		"from":         "loyaltyServices",
		"localField":   "service_id",
		"foreignField": "_id",
		"as":           "service",
	}},
	{"$unwind": bson.M{
		"path":                       "$service",
		"preserveNullAndEmptyArrays": true,
	}},
}

func NewVoucherSvc(ctx context.Context, db string) VoucherService {
	return &VoucherSvc{
		MainService: MainService[*models.Voucher, *models.VoucherResponse]{
			ctx:       ctx,
			db:        db,
			collName:  "loyaltyVouchers",
			MongoPipe: voucherPipe,
		}}
}

// Override GetOne to ensure status defaults to pending
func (w *VoucherSvc) GetOne(id string) (*models.VoucherResponse, error) {
	result, err := w.MainService.GetOne(id)
	if err != nil {
		return nil, err
	}

	// Ensure status always has a value, default to pending if empty
	if result.Status == "" {
		result.Status = models.VoucherStatusPending
	}

	// Set the computed voucher status
	result.VoucherStatus = result.GetVoucherStatus()

	return result, nil
}

func (w *VoucherSvc) Get(args *util.ListArgs, userId *primitive.ObjectID, partnerId *primitive.ObjectID) (*models.VoucherWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	if userId != nil {
		args.Query["user_id"] = userId
	}
	if partnerId != nil {
		args.Query["partner_id"] = partnerId
	}
	pipeline := PaginationPipe("vouchers", args, voucherPipe...)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.VoucherWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		partnerIds := lo.Map(result.Vouchers, func(voucher models.VoucherResponse, _ int) string {
			return voucher.GetPartnerId()
		})
		partnerIds = lo.Filter(partnerIds, func(partnerId string, _ int) bool { return partnerId != "" })
		partners, _ := util.FetchUsers(partnerIds, w.db)
		res := util.MapPartnerToData(partners, lo.ToSlicePtr(result.Vouchers))
		result.Vouchers = util.FromSlicePtr(res)

		// Ensure all vouchers have a status, default to pending if empty
		for i := range result.Vouchers {
			if result.Vouchers[i].Status == "" {
				result.Vouchers[i].Status = models.VoucherStatusPending
			}

			// Set the computed voucher status
			result.Vouchers[i].VoucherStatus = result.Vouchers[i].GetVoucherStatus()
		}

		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *VoucherSvc) GetOneByCode(code string, userId, partnerId *primitive.ObjectID,
) (*models.VoucherResponse, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filter := bson.M{"code": code, "trash": bson.M{"$ne": true}}
	if userId != nil {
		filter["user_id"] = userId
	}
	if partnerId != nil {
		filter["partner_id"] = partnerId
	}
	filterStage := bson.M{"$match": filter}
	pipeline := append([]bson.M{filterStage}, w.MongoPipe...)
	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.VoucherResponse
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}

		// Ensure status always has a value, default to pending if empty
		if result.Status == "" {
			result.Status = models.VoucherStatusPending
		}

		// Set the computed voucher status
		result.VoucherStatus = result.GetVoucherStatus()

		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *VoucherSvc) Add(data *models.Voucher) (*models.VoucherResponse, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	user := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)
	userId, _ := primitive.ObjectIDFromHex(user.User.Id)
	data.SetCreatedBy(userId)
	data.Id = primitive.NewObjectID()
	data.Code = fmt.Sprintf("%d", util.NumStringFromOid(data.Id))

	// Ensure status always has a value, default to pending if empty
	if data.Status == "" {
		data.Status = models.VoucherStatusPending
	}

	_, err := coll.InsertOne(w.ctx, data)

	if err != nil {
		return nil, err
	}
	if data.IsUsed {
		go NotifyOnCheckIn(w.db, data.UserId.Hex(), false)
		w.VoucherUsedTxn(data)
	}
	return w.GetOne(data.Id.Hex())
}

func (w *VoucherSvc) Use(code string, userId *primitive.ObjectID, partnerId *primitive.ObjectID) (*models.VoucherResponse, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	// First check if voucher exists and isn't trashed
	filter := bson.M{"code": code, "trash": bson.M{"$ne": true}}
	if userId != nil {
		filter["user_id"] = userId
	}
	if partnerId != nil {
		filter["partner_id"] = partnerId
	} else {
		filter["partner_id"] = bson.M{"$exists": false}
	}

	var existingVoucher models.Voucher
	err := coll.FindOne(w.ctx, filter).Decode(&existingVoucher)
	if err != nil {
		return nil, fmt.Errorf("voucher not found")
	}

	if existingVoucher.IsUsed {
		return nil, fmt.Errorf("voucher already used")
	}

	if !existingVoucher.ExpiredAt.IsZero() && existingVoucher.ExpiredAt.Before(time.Now()) {
		return nil, fmt.Errorf("voucher has expired")
	}

	isPartner := partnerId != nil

	// Update the voucher
	result := coll.FindOneAndUpdate(
		w.ctx,
		filter,
		bson.M{"$set": bson.M{"is_used": true, "updatedAt": time.Now()}},
	)

	var data models.Voucher
	if err := result.Decode(&data); err != nil {
		return nil, fmt.Errorf("error updating voucher: %v", err)
	}
	res, err := w.GetOne(data.Id.Hex())
	go NotifyOnCheckIn(w.db, data.UserId.Hex(), isPartner)
	w.VoucherUsedTxn(&data)

	return res, err
}

func (w *VoucherSvc) Reject(code string, userId *primitive.ObjectID, partnerId *primitive.ObjectID) (*models.VoucherResponse, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	return WithTransaction(w.ctx, func(sessCtx mongo.SessionContext) (*models.VoucherResponse, error) {
		filter := bson.M{"code": code, "trash": bson.M{"$ne": true}, "is_used": bson.M{"$ne": true}}

		if userId != nil {
			filter["user_id"] = userId
		}
		if partnerId != nil {
			filter["partner_id"] = partnerId
		} else {
			filter["partner_id"] = bson.M{"$exists": false}
		}

		// First check if voucher exists and validate its state
		var existingVoucher models.Voucher
		err := coll.FindOne(sessCtx, filter).Decode(&existingVoucher)
		if err != nil {
			return nil, fmt.Errorf("voucher not found")
		}

		if existingVoucher.IsUsed {
			return nil, fmt.Errorf("voucher already used")
		}

		if !existingVoucher.ExpiredAt.IsZero() && existingVoucher.ExpiredAt.Before(time.Now()) {
			return nil, fmt.Errorf("voucher has expired")
		}

		// Update the voucher
		result := coll.FindOneAndUpdate(
			sessCtx,
			filter,
			bson.M{"$set": bson.M{"updatedAt": time.Now(), "trash": true}},
		)

		var data models.Voucher
		if decErr := result.Decode(&data); decErr != nil {
			return nil, fmt.Errorf("error updating voucher")
		}
		res, err := w.GetOne(data.Id.Hex())
		if err != nil {
			return nil, err
		}
		isPartner := partnerId != nil
		go NotifyOnRejectVoucher(w.db, data.UserId.Hex(), isPartner)

		walletService := NewUserWalletSvcs(sessCtx, w.db)
		wallet, err := walletService.GetWalletByUserId(data.UserId.Hex())
		if err != nil {
			return nil, fmt.Errorf("wallet not found: %w", err)
		}
		wallet.Points += res.Points

		createdByUser, _ := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)
		createdById, _ := primitive.ObjectIDFromHex(createdByUser.User.Id)
		wallet.SetUpdatedBy(createdById)
		_, err = walletService.Update(wallet.Id.Hex(), &wallet.UserWallet)
		if err != nil {
			return nil, err
		}

		return res, nil
	})
}

func NotifyOnCheckIn(clientId, userId string, isPartner bool) {
	if !isPartner {
		util.LocalizeNotify(
			util.LocalizeNotificationMsg{
				XClient: clientId,
				Header: models.LocalizableString{
					Ar: "مرحبًا بك في مركز الحكمة الطبي!",
					En: "Welcome to Hikma Medical Center!",
				},
				Content: models.LocalizableString{
					Ar: "تم تسجيل وصولك بنجاح. نتمنى لك زيارة مريحة.",
					En: "You have successfully checked in. We wish you a comfortable visit",
				},
				TopicType: constants.NotificationTopicTypeUser,
				Topic:     userId,
				Type:      "fcm",
				Data: util.NotificationPayload{
					Model: "voucher",
				},
			},
		)
	}
}

func NotifyOnRejectVoucher(clientId, userId string, isPartner bool) {
	if !isPartner {
		util.LocalizeNotify(
			util.LocalizeNotificationMsg{
				XClient: clientId,
				Header: models.LocalizableString{
					Ar: "مرحبًا بك في مركز الحكمة الطبي!",
					En: "Welcome to Hikma Medical Center!",
				},
				Content: models.LocalizableString{
					Ar: "لقد تم رفض الخدمة التي تم حجزها من قبلك",
					En: "Your voucher has rejected.",
				},
				TopicType: constants.NotificationTopicTypeUser,
				Topic:     userId,
				Type:      "fcm",
				Data: util.NotificationPayload{
					Model: "voucher",
				},
			},
		)
	}
}

func (w *VoucherSvc) VoucherExpiredTxn(voucher *models.Voucher) error {
	transactionService := NewTransactionSvcs(w.ctx, w.db)
	newTransaction := &models.Transaction{
		UserId:   voucher.UserId,
		Activity: "Voucher Expired",
		Type:     constants.TXN_VOUCHER_EXPIRED,
		TypeName: constants.TXN_VOUCHER_EXPIRED.String(),
		Amount:   -float64(voucher.Points),
		RefId:    &voucher.Id,
	}
	newTransaction.SetCreatedBy(*voucher.CreatedBy)
	_, err := transactionService.Add(newTransaction)
	return err
}

func (w *VoucherSvc) GetForManagement(args *util.ListArgs) (*models.VoucherManagementWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	// Build base filter - always exclude trashed items
	baseFilter := bson.M{"trash": bson.M{"$ne": true}}

	// Handle name and email filters by finding matching user IDs first
	var nameFilter, emailFilter string
	if nameFilterVal, exists := args.Query["user_name_filter"]; exists {
		nameFilter = nameFilterVal.(string)
		delete(args.Query, "user_name_filter")
	}
	if emailFilterVal, exists := args.Query["user_email_filter"]; exists {
		emailFilter = emailFilterVal.(string)
		delete(args.Query, "user_email_filter")
	}

	// If we have name or email filters, we need to find matching user IDs first
	if nameFilter != "" || emailFilter != "" {
		// First, get all vouchers to extract all user IDs
		allVouchersPipeline := []bson.M{
			{"$match": bson.M{"trash": bson.M{"$ne": true}}},
			{"$group": bson.M{
				"_id": "$user_id",
			}},
		}

		allVouchersCursor, err := coll.Aggregate(w.ctx, allVouchersPipeline)
		if err != nil {
			return nil, err
		}
		defer allVouchersCursor.Close(w.ctx)

		var userIDResults []bson.M
		if err := allVouchersCursor.All(w.ctx, &userIDResults); err != nil {
			return nil, err
		}

		// Extract all user IDs
		var allUserIds []string
		for _, result := range userIDResults {
			if userID, ok := result["_id"].(primitive.ObjectID); ok {
				allUserIds = append(allUserIds, userID.Hex())
			}
		}

		// Fetch all users to filter by name/email
		if len(allUserIds) > 0 {
			allUsers, err := util.FetchUsers(allUserIds, w.db)
			if err == nil {
				var matchingUserIds []primitive.ObjectID
				for _, user := range allUsers {
					includeUser := true

					// Filter by name
					if nameFilter != "" {
						firstName := strings.ToLower(user.FirstName)
						lastName := strings.ToLower(user.LastName)
						searchName := strings.ToLower(nameFilter)

						if !strings.Contains(firstName, searchName) && !strings.Contains(lastName, searchName) {
							includeUser = false
						}
					}

					// Filter by email
					if emailFilter != "" && includeUser {
						userEmail := strings.ToLower(user.Email)
						searchEmail := strings.ToLower(emailFilter)

						if !strings.Contains(userEmail, searchEmail) {
							includeUser = false
						}
					}

					if includeUser {
						if userObjId, err := primitive.ObjectIDFromHex(user.Id); err == nil {
							matchingUserIds = append(matchingUserIds, userObjId)
						}
					}
				}

				// If we found matching users, add their IDs to the filter
				if len(matchingUserIds) > 0 {
					baseFilter["user_id"] = bson.M{"$in": matchingUserIds}
				} else {
					// If no users match the criteria, return empty result
					return &models.VoucherManagementWithPagination{
						Vouchers:   []models.VoucherManagementResponse{},
						Pagination: models.Pagination{},
					}, nil
				}
			}
		}
	}

	// Add any additional filters from args.Query
	for key, value := range args.Query {
		baseFilter[key] = value
	}

	// Get pagination values for data retrieval
	skip := w.getSkipValue(args)
	perPage := w.getLimitValue(args)

	// Get the data with pagination
	dataPipeline := []bson.M{
		{"$match": baseFilter},

		// Join with services
		{"$lookup": bson.M{
			"from":         "loyaltyServices",
			"localField":   "service_id",
			"foreignField": "_id",
			"as":           "service",
		}},
		{"$unwind": bson.M{
			"path":                       "$service",
			"preserveNullAndEmptyArrays": true,
		}},

		// Join with service groups
		{"$lookup": bson.M{
			"from":         "loyaltyServiceGroups",
			"localField":   "service.group",
			"foreignField": "_id",
			"as":           "service_group",
		}},
		{"$unwind": bson.M{
			"path":                       "$service_group",
			"preserveNullAndEmptyArrays": true,
		}},

		// Sort, skip, and limit
		{"$sort": w.getSortOrder(args)},
		{"$skip": skip},
		{"$limit": perPage},
	}

	dataCursor, err := coll.Aggregate(w.ctx, dataPipeline)
	if err != nil {
		return nil, err
	}
	defer dataCursor.Close(w.ctx)

	var vouchers []models.VoucherManagementResponse
	if err := dataCursor.All(w.ctx, &vouchers); err != nil {
		return nil, err
	}

	// Create simple result structure
	result := &models.VoucherManagementWithPagination{
		Vouchers:   vouchers,
		Pagination: models.Pagination{}, // Initialize with empty pagination for now
	}

	// Fetch user information for each voucher
	userIds := make([]string, 0, len(result.Vouchers))
	for _, voucher := range result.Vouchers {
		if voucher.UserId.Hex() != "" {
			userIds = append(userIds, voucher.UserId.Hex())
		}
	}

	if len(userIds) > 0 {
		users, _ := util.FetchUsers(userIds, w.db)
		userMap := make(map[string]*models.VoucherUserInfo)
		for _, user := range users {
			// Extract phone from user meta or set empty string
			phone := ""
			// You can uncomment and modify this if phone is stored in Meta field
			// for _, meta := range user.Meta {
			// 	if meta.Key == "phone" {
			// 		phone = meta.Value
			// 		break
			// 	}
			// }

			userMap[user.Id] = &models.VoucherUserInfo{
				Id:        user.Id,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Email:     user.Email,
				Phone:     phone,
			}
		}

		// Attach user information to vouchers
		for i := range result.Vouchers {
			if user, exists := userMap[result.Vouchers[i].UserId.Hex()]; exists {
				result.Vouchers[i].User = user
			}
		}
	}

	// Ensure all vouchers have a status, default to pending if empty
	for i := range result.Vouchers {
		if result.Vouchers[i].Status == "" {
			result.Vouchers[i].Status = models.VoucherStatusPending
		}

		// Set the computed voucher status
		result.Vouchers[i].VoucherStatus = result.Vouchers[i].GetVoucherStatus()
	}

	return result, nil
}

// Helper methods for pagination calculations
func (w *VoucherSvc) getPageNumber(args *util.ListArgs) int {
	if args.Page == "" {
		return 1
	}
	page := 1
	if p, err := strconv.Atoi(args.Page); err == nil && p > 0 {
		page = p
	}
	return page
}

func (w *VoucherSvc) getLimitValue(args *util.ListArgs) int {
	if args.PerPage == "" {
		return 10
	}
	limit := 10
	if l, err := strconv.Atoi(args.PerPage); err == nil && l > 0 {
		limit = l
	}
	return limit
}

func (w *VoucherSvc) getSkipValue(args *util.ListArgs) int {
	page := w.getPageNumber(args)
	limit := w.getLimitValue(args)
	return (page - 1) * limit
}

func (w *VoucherSvc) getSortOrder(args *util.ListArgs) bson.M {
	if args.Sort != nil && len(args.Sort) > 0 {
		return args.Sort
	}
	// Default sort by creation date, newest first
	return bson.M{"createdAt": -1}
}

func (w *VoucherSvc) UpdateStatus(id string, status string, comments string) (*models.VoucherResponse, error) {
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	coll := database.Mdb.Database(w.db).Collection(w.collName)
	filter := bson.M{"_id": _id, "trash": bson.M{"$ne": true}}

	updateData := bson.M{
		"status":    status,
		"comments":  comments,
		"updatedAt": time.Now(),
	}

	update := bson.M{"$set": updateData}

	_, err = coll.UpdateOne(w.ctx, filter, update)
	if err != nil {
		return nil, err
	}

	return w.GetOne(id)
}

func (w *VoucherSvc) VoucherUsedTxn(voucher *models.Voucher) error {
	transactionService := NewTransactionSvcs(w.ctx, w.db)
	newTransaction := &models.Transaction{
		UserId:   voucher.UserId,
		Activity: "Voucher Used",
		Type:     constants.TXN_VOUCHER_USED,
		TypeName: constants.TXN_VOUCHER_USED.String(),
		Amount:   -float64(voucher.Points),
		RefId:    &voucher.Id,
	}
	newTransaction.SetCreatedBy(*voucher.CreatedBy)
	_, err := transactionService.Add(newTransaction)
	return err
}

// FilterVouchers - General filter method that accepts any fields in the filter map
func (w *VoucherSvc) FilterVouchers(filter map[string]interface{}, args *util.ListArgs) (*models.VoucherWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	// Build base filter - always exclude trashed items
	baseFilter := bson.M{"trash": bson.M{"$ne": true}}

	// Merge the provided filter with base filter
	for key, value := range filter {
		baseFilter[key] = value
	}

	// Add any additional filters from args.Query
	for key, value := range args.Query {
		baseFilter[key] = value
	}

	// Get pagination values
	skip := w.getSkipValue(args)
	perPage := w.getLimitValue(args)

	// Build pipeline for data retrieval
	dataPipeline := []bson.M{
		{"$match": baseFilter},
	}

	// Add the standard voucher pipeline (service lookup)
	dataPipeline = append(dataPipeline, voucherPipe...)

	// Add sort, skip, and limit
	dataPipeline = append(dataPipeline, []bson.M{
		{"$sort": w.getSortOrder(args)},
		{"$skip": skip},
		{"$limit": perPage},
	}...)

	// Get total count for pagination
	countPipeline := []bson.M{
		{"$match": baseFilter},
		{"$count": "total"},
	}

	countCursor, err := coll.Aggregate(w.ctx, countPipeline)
	if err != nil {
		return nil, err
	}
	defer countCursor.Close(w.ctx)

	var countResult []bson.M
	if err := countCursor.All(w.ctx, &countResult); err != nil {
		return nil, err
	}

	total := 0
	if len(countResult) > 0 {
		if totalVal, ok := countResult[0]["total"].(int32); ok {
			total = int(totalVal)
		}
	}

	// Get the data
	dataCursor, err := coll.Aggregate(w.ctx, dataPipeline)
	if err != nil {
		return nil, err
	}
	defer dataCursor.Close(w.ctx)

	var vouchers []models.VoucherResponse
	if err := dataCursor.All(w.ctx, &vouchers); err != nil {
		return nil, err
	}

	// Fetch partner information for vouchers (same pattern as existing code)
	partnerIds := lo.Map(vouchers, func(voucher models.VoucherResponse, _ int) string {
		return voucher.GetPartnerId()
	})
	partnerIds = lo.Filter(partnerIds, func(partnerId string, _ int) bool { return partnerId != "" })
	partners, _ := util.FetchUsers(partnerIds, w.db)
	res := util.MapPartnerToData(partners, lo.ToSlicePtr(vouchers))
	vouchers = util.FromSlicePtr(res)

	// Ensure all vouchers have a status, default to pending if empty
	for i := range vouchers {
		if vouchers[i].Status == "" {
			vouchers[i].Status = models.VoucherStatusPending
		}

		// Set the computed voucher status
		vouchers[i].VoucherStatus = vouchers[i].GetVoucherStatus()
	}

	// Calculate pagination info
	limit := w.getLimitValue(args)
	totalPages := float64(total) / float64(limit)
	if total%limit > 0 {
		totalPages = float64(int(totalPages) + 1)
	}

	pagination := models.Pagination{
		TotalPages: totalPages,
		PerPage:    int64(limit),
		TotalCount: int64(total),
	}

	return &models.VoucherWithPagination{
		Vouchers:   vouchers,
		Pagination: pagination,
	}, nil
}

// GetForManagementV2 - Get vouchers for management with structured query conditions
func (w *VoucherSvc) GetForManagementV2(conditions *query.Conditions, skip, limit int) (*models.VoucherManagementWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	// Handle special filters that require user lookup or computed fields
	userNameFilter := ""
	userEmailFilter := ""
	voucherStatusFilter := ""
	serviceGroupFilter := ""
	serviceIdFilter := ""    // For service._id filtering
	statusFilter := ""       // For handling empty status filtering
	hasStatusFilter := false // Track if status filter was provided

	// Extract special filters and remove them from the main filter
	newColumns := []query.Column{}
	for _, col := range conditions.Columns {
		switch col.Name {
		case "user_name_filter":
			if str, ok := col.Value.(string); ok {
				userNameFilter = str
			}
		case "user_email_filter":
			if str, ok := col.Value.(string); ok {
				userEmailFilter = str
			}
		case "voucher_status":
			if str, ok := col.Value.(string); ok {
				voucherStatusFilter = str
			}
		case "service.group":
			// Don't add service.group to the main filter - handle it after lookup
			if str, ok := col.Value.(string); ok {
				serviceGroupFilter = str
			}
		case "service._id":
			// Don't add service._id to the main filter - handle it after lookup
			if str, ok := col.Value.(string); ok {
				serviceIdFilter = str
			}
		case "status":
			// Handle status filtering specially to account for empty/null/missing values
			hasStatusFilter = true
			if str, ok := col.Value.(string); ok {
				statusFilter = str
			}
		case "service_id":
			// Convert service_id string to ObjectID
			if str, ok := col.Value.(string); ok {
				if objID, err := primitive.ObjectIDFromHex(str); err == nil {
					col.Value = objID
				}
			}
			newColumns = append(newColumns, col)
		default:
			// Keep other columns for normal MongoDB filtering
			newColumns = append(newColumns, col)
		}
	}

	// Update conditions with only the filterable columns
	filteredConditions := &query.Conditions{Columns: newColumns}

	// Convert filtered conditions to MongoDB filter
	filter, err := filteredConditions.ConvertToMongo()
	if err != nil {
		return nil, fmt.Errorf("failed to convert conditions to filter: %w", err)
	}

	// Always exclude trashed items
	filter["trash"] = bson.M{"$ne": true}

	// Handle status filtering
	if hasStatusFilter {
		if statusFilter == "" {
			// When filtering for empty status, look for documents where status is "", null, or doesn't exist
			// This handles the legacy data that hasn't been migrated yet
			filter["$or"] = []bson.M{
				{"status": ""},
				{"status": bson.M{"$exists": false}},
				{"status": nil},
			}
		} else {
			// For non-empty status values, use exact match
			filter["status"] = statusFilter
		}
	}

	// Handle voucher_status filter by converting to MongoDB conditions
	if voucherStatusFilter != "" {
		// If we already have a $or from status filtering, we need to handle this differently
		if _, hasOr := filter["$or"]; hasOr {
			// Convert the existing filter to use $and to combine multiple conditions
			existingOr := filter["$or"]
			delete(filter, "$or")

			now := time.Now()
			switch voucherStatusFilter {
			case "used":
				filter["is_used"] = true
			case "active":
				filter["is_used"] = false
				// Combine status $or with voucher_status $or using $and
				filter["$and"] = []bson.M{
					{"$or": existingOr.([]bson.M)},
					{"$or": []bson.M{
						{"expired_at": bson.M{"$exists": false}},
						{"expired_at": bson.M{"$gt": time.Now()}},
					}},
				}
			case "expired":
				filter["is_used"] = false
				filter["expired_at"] = bson.M{
					"$exists": true,
					"$ne":     time.Time{}, // Not zero time (0001-01-01)
					"$lte":    now,
				}
				// For expired, we also need to combine with the status $or
				filter["$and"] = []bson.M{
					{"$or": existingOr.([]bson.M)},
				}
			}
		} else {
			// No existing $or, handle normally
			now := time.Now()
			switch voucherStatusFilter {
			case "used":
				filter["is_used"] = true
			case "active":
				filter["is_used"] = false
				filter["$or"] = []bson.M{
					{"expired_at": bson.M{"$exists": false}},
					{"expired_at": bson.M{"$gt": time.Now()}},
				}
			case "expired":
				filter["is_used"] = false
				filter["expired_at"] = bson.M{
					"$exists": true,
					"$ne":     time.Time{}, // Not zero time (0001-01-01)
					"$lte":    now,
				}
			}
		}
	}

	// Handle user name/email filtering BEFORE the main query
	if userNameFilter != "" || userEmailFilter != "" {
		// First, get all vouchers to extract all user IDs
		allVouchersPipeline := []bson.M{
			{"$match": bson.M{"trash": bson.M{"$ne": true}}},
			{"$group": bson.M{
				"_id": "$user_id",
			}},
		}

		allVouchersCursor, err := coll.Aggregate(w.ctx, allVouchersPipeline)
		if err != nil {
			return nil, err
		}
		defer allVouchersCursor.Close(w.ctx)

		var userIDResults []bson.M
		if err := allVouchersCursor.All(w.ctx, &userIDResults); err != nil {
			return nil, err
		}

		// Extract all user IDs
		var allUserIds []string
		for _, result := range userIDResults {
			if userID, ok := result["_id"].(primitive.ObjectID); ok {
				allUserIds = append(allUserIds, userID.Hex())
			}
		}

		// Fetch all users to filter by name/email
		if len(allUserIds) > 0 {
			allUsers, err := util.FetchUsers(allUserIds, w.db)
			if err == nil {
				var matchingUserIds []primitive.ObjectID
				for _, user := range allUsers {
					includeUser := true

					// Filter by name
					if userNameFilter != "" {
						firstName := strings.ToLower(user.FirstName)
						lastName := strings.ToLower(user.LastName)
						searchName := strings.ToLower(userNameFilter)

						if !strings.Contains(firstName, searchName) && !strings.Contains(lastName, searchName) {
							includeUser = false
						}
					}

					// Filter by email
					if userEmailFilter != "" && includeUser {
						userEmail := strings.ToLower(user.Email)
						searchEmail := strings.ToLower(userEmailFilter)

						if !strings.Contains(userEmail, searchEmail) {
							includeUser = false
						}
					}

					if includeUser {
						if userObjId, err := primitive.ObjectIDFromHex(user.Id); err == nil {
							matchingUserIds = append(matchingUserIds, userObjId)
						}
					}
				}

				// If we found matching users, add their IDs to the filter
				if len(matchingUserIds) > 0 {
					filter["user_id"] = bson.M{"$in": matchingUserIds}
				} else {
					// If no users match the criteria, return empty result
					return &models.VoucherManagementWithPagination{
						Vouchers:   []models.VoucherManagementResponse{},
						Pagination: models.Pagination{},
					}, nil
				}
			}
		}
	}

	// Create the count pipeline AFTER all filters are applied
	countPipeline := []bson.M{
		{"$match": filter},
	}

	// Add service group and service ID filters to count pipeline if they exist
	if serviceGroupFilter != "" {
		if objID, err := primitive.ObjectIDFromHex(serviceGroupFilter); err == nil {
			// Add service lookup stages to count pipeline for service.group filtering
			countPipeline = append(countPipeline, []bson.M{
				{"$lookup": bson.M{
					"from":         "loyaltyServices",
					"localField":   "service_id",
					"foreignField": "_id",
					"as":           "service",
				}},
				{"$unwind": bson.M{
					"path":                       "$service",
					"preserveNullAndEmptyArrays": true,
				}},
				{"$match": bson.M{"service.group": objID}},
			}...)
		}
	}

	if serviceIdFilter != "" {
		if objID, err := primitive.ObjectIDFromHex(serviceIdFilter); err == nil {
			// If count pipeline doesn't have service lookup yet, add it
			if serviceGroupFilter == "" {
				countPipeline = append(countPipeline, []bson.M{
					{"$lookup": bson.M{
						"from":         "loyaltyServices",
						"localField":   "service_id",
						"foreignField": "_id",
						"as":           "service",
					}},
					{"$unwind": bson.M{
						"path":                       "$service",
						"preserveNullAndEmptyArrays": true,
					}},
				}...)
			}
			countPipeline = append(countPipeline, bson.M{"$match": bson.M{"service._id": objID}})
		}
	}

	// Add the count stage
	countPipeline = append(countPipeline, bson.M{"$count": "total"})

	// Get total count for pagination
	countCursor, err := coll.Aggregate(w.ctx, countPipeline)
	if err != nil {
		return nil, err
	}
	defer countCursor.Close(w.ctx)

	var countResult []bson.M
	if err := countCursor.All(w.ctx, &countResult); err != nil {
		return nil, err
	}

	total := 0
	if len(countResult) > 0 {
		if totalVal, ok := countResult[0]["total"].(int32); ok {
			total = int(totalVal)
		}
	}

	// Build pipeline for data retrieval with complete service and service group lookups
	dataPipeline := []bson.M{
		{"$match": filter},

		// Join with services
		{"$lookup": bson.M{
			"from":         "loyaltyServices",
			"localField":   "service_id",
			"foreignField": "_id",
			"as":           "service",
		}},
		{"$unwind": bson.M{
			"path":                       "$service",
			"preserveNullAndEmptyArrays": true,
		}},

		// Join with service groups
		{"$lookup": bson.M{
			"from":         "loyaltyServiceGroups",
			"localField":   "service.group",
			"foreignField": "_id",
			"as":           "service_group",
		}},
		{"$unwind": bson.M{
			"path":                       "$service_group",
			"preserveNullAndEmptyArrays": true,
		}},

		// Sort, skip, and limit
		{"$sort": bson.M{"createdAt": -1}}, // Default sort by creation date, newest first
		{"$skip": skip},
		{"$limit": limit},
	}

	// Add service group filter after lookups if specified
	if serviceGroupFilter != "" {
		if objID, err := primitive.ObjectIDFromHex(serviceGroupFilter); err == nil {
			// Insert the service group match after the lookups but before sort/skip/limit
			serviceGroupMatch := bson.M{"$match": bson.M{"service.group": objID}}
			// Insert at position before sort (index -3)
			insertIndex := len(dataPipeline) - 3
			dataPipeline = append(dataPipeline[:insertIndex], append([]bson.M{serviceGroupMatch}, dataPipeline[insertIndex:]...)...)
		}
	}

	// Add service._id filter after lookups if specified
	if serviceIdFilter != "" {
		if objID, err := primitive.ObjectIDFromHex(serviceIdFilter); err == nil {
			// Insert the service._id match after the lookups but before sort/skip/limit
			serviceIdMatch := bson.M{"$match": bson.M{"service._id": objID}}
			// Insert at position before sort (index -3)
			insertIndex := len(dataPipeline) - 3
			dataPipeline = append(dataPipeline[:insertIndex], append([]bson.M{serviceIdMatch}, dataPipeline[insertIndex:]...)...)
		}
	}

	// Get the data
	dataCursor, err := coll.Aggregate(w.ctx, dataPipeline)
	if err != nil {
		return nil, err
	}
	defer dataCursor.Close(w.ctx)

	var vouchers []models.VoucherManagementResponse
	if err := dataCursor.All(w.ctx, &vouchers); err != nil {
		return nil, err
	}

	// Note: service.group filter is handled in the pipeline after lookups, not here

	// Fetch user information for each voucher
	userIds := make([]string, 0, len(vouchers))
	for _, voucher := range vouchers {
		if voucher.UserId.Hex() != "" {
			userIds = append(userIds, voucher.UserId.Hex())
		}
	}

	if len(userIds) > 0 {
		users, _ := util.FetchUsers(userIds, w.db)
		userMap := make(map[string]*models.VoucherUserInfo)
		for _, user := range users {
			// Extract phone from user meta or set empty string
			phone := ""
			// You can uncomment and modify this if phone is stored in Meta field
			// for _, meta := range user.Meta {
			// 	if meta.Key == "phone" {
			// 		phone = meta.Value
			// 		break
			// 	}
			// }

			userMap[user.Id] = &models.VoucherUserInfo{
				Id:        user.Id,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Email:     user.Email,
				Phone:     phone,
			}
		}

		// Attach user information to vouchers
		for i := range vouchers {
			if user, exists := userMap[vouchers[i].UserId.Hex()]; exists {
				vouchers[i].User = user
			}
		}
	}

	// Ensure all vouchers have a status, default to pending if empty
	for i := range vouchers {
		if vouchers[i].Status == "" {
			vouchers[i].Status = models.VoucherStatusPending
		}

		// Set the computed voucher status
		vouchers[i].VoucherStatus = vouchers[i].GetVoucherStatus()
	}

	// Calculate pagination info
	totalPages := float64(total) / float64(limit)
	if total%limit > 0 {
		totalPages = float64(int(totalPages) + 1)
	}

	pagination := models.Pagination{
		TotalPages: totalPages,
		PerPage:    int64(limit),
		TotalCount: int64(total),
	}

	return &models.VoucherManagementWithPagination{
		Vouchers:   vouchers,
		Pagination: pagination,
	}, nil
}

// FixVoucherStatusData - One-time method to fix vouchers with empty/null status to have "pending" status
// This should be called once to migrate the data and fix the inconsistency
func (w *VoucherSvc) FixVoucherStatusData() error {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	// Find all vouchers with empty, null, or missing status that are not trashed
	filter := bson.M{
		"trash": bson.M{"$ne": true},
		"$or": []bson.M{
			{"status": ""},
			{"status": bson.M{"$exists": false}},
			{"status": nil},
		},
	}

	// Update them to have "pending" status
	update := bson.M{
		"$set": bson.M{
			"status":    models.VoucherStatusPending,
			"updatedAt": time.Now(),
		},
	}

	result, err := coll.UpdateMany(w.ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update voucher statuses: %w", err)
	}

	fmt.Printf("Updated %d vouchers to have 'pending' status\n", result.ModifiedCount)
	return nil
}
