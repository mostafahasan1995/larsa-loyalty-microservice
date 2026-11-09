package vouchers

import (
	"fmt"
	"loyalty/pkg/caching"
	"loyalty/pkg/constants"
	"loyalty/pkg/models"
	"loyalty/pkg/services"
	"loyalty/pkg/util"
	"net/http"
	"strconv"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"time"

	"loyalty/pkg/query"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"github.com/go-chi/chi"
	"go.mongodb.org/mongo-driver/bson"
)

func GetVoucher(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	rawQuery := r.URL.RawQuery
	path := r.URL.Path

	cacheKey := util.CacheKeyForMulti(constants.VOUCHER, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		args, err := util.Paginate(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		voucherService := services.NewVoucherSvc(r.Context(), db)

		result, err := voucherService.Get(args, nil, nil)

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

func GetAllVoucher(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	//

	rawQuery := r.URL.RawQuery
	path := r.URL.Path
	query := r.URL.Query().Get("query")

	cacheKey := util.CacheKeyForMulti(constants.VOUCHER, db, path, lang, rawQuery)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		voucherService := services.NewVoucherSvc(r.Context(), db)
		result, err := voucherService.GetAll(query)

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

func GetVoucherById(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("accept-language")
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	cacheKey := util.CacheKeyForOne(constants.VOUCHER, db, id, lang)

	cache, cacheErr := caching.Rdb.GetByKey(cacheKey)

	if cacheErr != nil {
		voucherService := services.NewVoucherSvc(r.Context(), db)
		result, err := voucherService.GetOne(id)

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

func CreateVoucher(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	var data models.Voucher

	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	voucherService := services.NewVoucherSvc(r.Context(), db)
	res, errIns := voucherService.Add(&data)
	if errIns != nil {
		http.Error(w, errIns.Error(), http.StatusInternalServerError)
		return
	}
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.VOUCHER, db), 0)

	j, _ := json.MarshalContext(r.Context(), res)

	w.Write(j)
}

func UpdateVoucher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	var data models.Voucher
	if errDec := json.NewDecoder(r.Body).Decode(&data); errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
		return
	}

	service := services.NewVoucherSvc(r.Context(), db)
	res, errUpdate := service.Update(id, &data)
	if errUpdate != nil {
		http.Error(w, "error update tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.VOUCHER, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.VOUCHER, db, id), 0)

	j, _ := json.MarshalContext(r.Context(), res)

	w.Write(j)
}

func DeleteVoucher(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")

	voucherService := services.NewVoucherSvc(r.Context(), db)

	if errDel := voucherService.Delete(id); errDel != nil {
		http.Error(w, "error delete tax", http.StatusInternalServerError)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.VOUCHER, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.VOUCHER, db, id), 0)

	w.WriteHeader(http.StatusOK)
}

func GetUserVouchers(w http.ResponseWriter, r *http.Request) {
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
	voucherService := services.NewVoucherSvc(r.Context(), db)

	result, err := voucherService.Get(args, lo.ToPtr(_userId), nil)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}

func PartnerGetVouchers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	user := r.Context().Value(constants.CtxAuthKey).(common.Credentials)
	partnerId, _ := primitive.ObjectIDFromHex(user.User.Id)

	args, err := util.Paginate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	voucherService := services.NewVoucherSvc(r.Context(), db)

	result, err := voucherService.Get(args, nil, lo.ToPtr(partnerId))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}

func GetUserVouchersById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	userId := chi.URLParam(r, "userId")
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
	voucherService := services.NewVoucherSvc(r.Context(), db)

	result, err := voucherService.Get(args, lo.ToPtr(_userId), nil)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)

	w.Write(j)
}

func GetVoucherByCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	code := chi.URLParam(r, "code")

	voucherService := services.NewVoucherSvc(r.Context(), db)
	result, err := voucherService.GetOneByCode(code, nil, nil)

	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func GetUserVoucherByCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	code := chi.URLParam(r, "code")
	userId := chi.URLParam(r, "userId")
	_userId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	voucherService := services.NewVoucherSvc(r.Context(), db)
	result, err := voucherService.GetOneByCode(code, &_userId, nil)

	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func PartnerGetVoucherByCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	partnerId := r.Context().Value(constants.CtxAuthKey).(common.Credentials).User.Id
	_partnerId, _ := primitive.ObjectIDFromHex(partnerId)

	code := chi.URLParam(r, "code")

	voucherService := services.NewVoucherSvc(r.Context(), db)
	result, err := voucherService.GetOneByCode(code, nil, lo.ToPtr(_partnerId))

	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func UseVoucher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	code := chi.URLParam(r, "code")

	userId := chi.URLParam(r, "userId")
	_userId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	voucherService := services.NewVoucherSvc(r.Context(), db)
	result, err := voucherService.Use(code, lo.ToPtr(_userId), nil)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.VOUCHER, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.VOUCHER, db, result.Id.Hex()), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func RejectVoucher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	code := chi.URLParam(r, "code")

	userId := chi.URLParam(r, "userId")
	_userId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	voucherService := services.NewVoucherSvc(r.Context(), db)
	result, err := voucherService.Reject(code, lo.ToPtr(_userId), nil)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.VOUCHER, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.VOUCHER, db, result.Id.Hex()), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func PartnerUseVoucher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	code := chi.URLParam(r, "code")

	partnerId := r.Context().Value(constants.CtxAuthKey).(common.Credentials).User.Id
	_partnerId, _ := primitive.ObjectIDFromHex(partnerId)

	voucherService := services.NewVoucherSvc(r.Context(), db)
	result, err := voucherService.Use(code, nil, lo.ToPtr(_partnerId))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.VOUCHER, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.VOUCHER, db, result.Id.Hex()), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

func PartnerRejectVoucher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	code := chi.URLParam(r, "code")

	partnerId := r.Context().Value(constants.CtxAuthKey).(common.Credentials).User.Id
	_partnerId, _ := primitive.ObjectIDFromHex(partnerId)

	voucherService := services.NewVoucherSvc(r.Context(), db)
	result, err := voucherService.Reject(code, nil, lo.ToPtr(_partnerId))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.VOUCHER, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.VOUCHER, db, result.Id.Hex()), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

// GetVouchersForManagement - Get vouchers with user information for management interface
func GetVouchersForManagement(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	args, err := util.Paginate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add filtering support
	queryParams := r.URL.Query()

	// Parse JSON query parameter if provided
	if queryParam := queryParams.Get("query"); queryParam != "" {
		var queryMap map[string]interface{}
		if err := json.Unmarshal([]byte(queryParam), &queryMap); err == nil {
			// Handle special cases for status filtering
			if statusValue, exists := queryMap["status"]; exists {
				if statusStr, ok := statusValue.(string); ok && statusStr == "pending" {
					// For pending status, include both explicit "pending" and empty status fields
					queryMap["$or"] = []bson.M{
						{"status": "pending"},
						{"status": bson.M{"$exists": false}},
						{"status": ""},
					}
					delete(queryMap, "status") // Remove the original status field
				}
			}

			// Merge the JSON query with existing args.Query
			for key, value := range queryMap {
				args.Query[key] = value
			}
		}
	}

	// Filter by status
	if status := queryParams.Get("status"); status != "" {
		if status == "pending" {
			// For pending status, include both explicit "pending" and empty status fields
			args.Query["$or"] = []bson.M{
				{"status": "pending"},
				{"status": bson.M{"$exists": false}},
				{"status": ""},
			}
		} else {
			args.Query["status"] = status
		}
	}

	// Filter by voucher_status (computed field)
	if voucherStatus := queryParams.Get("voucher_status"); voucherStatus != "" {
		switch voucherStatus {
		case "used":
			args.Query["is_used"] = true
		case "active":
			args.Query["is_used"] = false
			// For active vouchers, we need to ensure they're not expired
			args.Query["$or"] = []bson.M{
				{"expired_at": bson.M{"$exists": false}},
				{"expired_at": bson.M{"$gt": time.Now()}},
			}
		case "expired":
			args.Query["is_used"] = false
			args.Query["expired_at"] = bson.M{"$lt": time.Now()}
		}
	}

	// Filter by is_used (independent filter)
	if isUsed := queryParams.Get("is_used"); isUsed != "" {
		if isUsed == "true" {
			args.Query["is_used"] = true
		} else if isUsed == "false" {
			args.Query["is_used"] = false
		}
	}

	// Filter by name (user first name or last name)
	if name := queryParams.Get("name"); name != "" {
		// Store name filter for later processing in service layer
		args.Query["user_name_filter"] = name
	}

	// Filter by email (user email)
	if email := queryParams.Get("email"); email != "" {
		// Store email filter for later processing in service layer
		args.Query["user_email_filter"] = email
	}

	// Search functionality
	if search := queryParams.Get("search"); search != "" {
		// Create regex for case-insensitive search
		searchRegex := bson.M{"$regex": search, "$options": "i"}

		// Search in voucher code first (simple field search)
		searchConditions := []bson.M{
			{"code": searchRegex},
		}

		args.Query["$or"] = searchConditions
	}

	// Filter by user_id
	if userId := queryParams.Get("user_id"); userId != "" {
		if userObjId, err := primitive.ObjectIDFromHex(userId); err == nil {
			args.Query["user_id"] = userObjId
		}
	}

	// Filter by service_id
	if serviceId := queryParams.Get("service_id"); serviceId != "" {
		if serviceObjId, err := primitive.ObjectIDFromHex(serviceId); err == nil {
			args.Query["service_id"] = serviceObjId
		}
	}

	// Filter by date range
	if fromDate := queryParams.Get("from"); fromDate != "" {
		if parsedDate, err := time.Parse("2006-01-02", fromDate); err == nil {
			if args.Query["createdAt"] == nil {
				args.Query["createdAt"] = bson.M{}
			}
			args.Query["createdAt"].(bson.M)["$gte"] = parsedDate
		}
	}

	if toDate := queryParams.Get("to"); toDate != "" {
		if parsedDate, err := time.Parse("2006-01-02", toDate); err == nil {
			// Add one day to include the entire day
			endDate := parsedDate.AddDate(0, 0, 1)
			if args.Query["createdAt"] == nil {
				args.Query["createdAt"] = bson.M{}
			}
			args.Query["createdAt"].(bson.M)["$lt"] = endDate
		}
	}

	// Add sorting support
	sortBy := queryParams.Get("sort_by")
	sortOrder := queryParams.Get("sort_order")

	if sortBy != "" {
		order := 1 // ascending by default
		if sortOrder == "desc" {
			order = -1
		}

		// Map frontend sort fields to database fields
		switch sortBy {
		case "date", "created_at":
			args.Sort = bson.M{"createdAt": order}
		case "status":
			args.Sort = bson.M{"status": order}
		case "voucher_status":
			// For voucher_status sorting, we need to sort by is_used first, then by expired_at
			if order == 1 {
				args.Sort = bson.M{"is_used": 1, "expired_at": 1}
			} else {
				args.Sort = bson.M{"is_used": -1, "expired_at": -1}
			}
		case "points":
			args.Sort = bson.M{"points": order}
		case "is_used":
			args.Sort = bson.M{"is_used": order}
		case "expired_at":
			args.Sort = bson.M{"expired_at": order}
		default:
			args.Sort = bson.M{"createdAt": -1} // default sort
		}
	} else {
		args.Sort = bson.M{"createdAt": -1} // default sort by newest first
	}

	voucherService := services.NewVoucherSvc(r.Context(), db)

	result, err := voucherService.GetForManagement(args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate proper pagination

	perPage := 10
	if args.Page != "" {
		if p, err := strconv.Atoi(args.Page); err == nil && p > 0 {

		}
	}
	if args.PerPage != "" {
		if pp, err := strconv.Atoi(args.PerPage); err == nil && pp > 0 {
			perPage = pp
		}
	}

	// Count total vouchers with same filters
	countArgs := &util.ListArgs{
		Query:   args.Query,
		PerPage: "1000000", // Large number to get all
		Page:    "1",
	}
	allResult, _ := voucherService.GetForManagement(countArgs)
	totalCount := len(allResult.Vouchers)
	totalPages := (totalCount + perPage - 1) / perPage

	// Create custom response with correct pagination
	response := map[string]interface{}{
		"vouchers": result.Vouchers,
		"pagination": map[string]interface{}{
			"totalPages": totalPages,
			"perPage":    perPage,
			"totalCount": totalCount,
			// "currentPage": page,
		},
	}

	j, _ := json.MarshalContext(r.Context(), response)
	w.Write(j)
}

// GetAllVouchersForManagement - Get all vouchers without pagination for management interface
func GetAllVouchersForManagement(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	args, err := util.Paginate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add filtering support (same as paginated version)
	queryParams := r.URL.Query()

	// Parse JSON query parameter if provided
	if queryParam := queryParams.Get("query"); queryParam != "" {
		var queryMap map[string]interface{}
		if err := json.Unmarshal([]byte(queryParam), &queryMap); err == nil {
			// Handle special cases for status filtering
			if statusValue, exists := queryMap["status"]; exists {
				if statusStr, ok := statusValue.(string); ok && statusStr == "pending" {
					// For pending status, include both explicit "pending" and empty status fields
					queryMap["$or"] = []bson.M{
						{"status": "pending"},
						{"status": bson.M{"$exists": false}},
						{"status": ""},
					}
					delete(queryMap, "status") // Remove the original status field
				}
			}

			// Merge the JSON query with existing args.Query
			for key, value := range queryMap {
				args.Query[key] = value
			}
		}
	}

	// Filter by status
	if status := queryParams.Get("status"); status != "" {
		if status == "pending" {
			// For pending status, include both explicit "pending" and empty status fields
			args.Query["$or"] = []bson.M{
				{"status": "pending"},
				{"status": bson.M{"$exists": false}},
				{"status": ""},
			}
		} else {
			args.Query["status"] = status
		}
	}

	// Filter by voucher_status (computed field)
	if voucherStatus := queryParams.Get("voucher_status"); voucherStatus != "" {
		switch voucherStatus {
		case "used":
			args.Query["is_used"] = true
		case "active":
			args.Query["is_used"] = false
			// For active vouchers, we need to ensure they're not expired
			args.Query["$or"] = []bson.M{
				{"expired_at": bson.M{"$exists": false}},
				{"expired_at": bson.M{"$gt": time.Now()}},
			}
		case "expired":
			args.Query["is_used"] = false
			args.Query["expired_at"] = bson.M{"$lt": time.Now()}
		}
	}

	// Filter by is_used (independent filter)
	if isUsed := queryParams.Get("is_used"); isUsed != "" {
		if isUsed == "true" {
			args.Query["is_used"] = true
		} else if isUsed == "false" {
			args.Query["is_used"] = false
		}
	}

	// Filter by name (user first name or last name)
	if name := queryParams.Get("name"); name != "" {
		// Store name filter for later processing in service layer
		args.Query["user_name_filter"] = name
	}

	// Filter by email (user email)
	if email := queryParams.Get("email"); email != "" {
		// Store email filter for later processing in service layer
		args.Query["user_email_filter"] = email
	}

	// Filter by user_id
	if userId := queryParams.Get("user_id"); userId != "" {
		if userObjId, err := primitive.ObjectIDFromHex(userId); err == nil {
			args.Query["user_id"] = userObjId
		}
	}

	// Filter by service_id
	if serviceId := queryParams.Get("service_id"); serviceId != "" {
		if serviceObjId, err := primitive.ObjectIDFromHex(serviceId); err == nil {
			args.Query["service_id"] = serviceObjId
		}
	}

	// Filter by date range
	if fromDate := queryParams.Get("from"); fromDate != "" {
		if parsedDate, err := time.Parse("2006-01-02", fromDate); err == nil {
			if args.Query["createdAt"] == nil {
				args.Query["createdAt"] = bson.M{}
			}
			args.Query["createdAt"].(bson.M)["$gte"] = parsedDate
		}
	}

	if toDate := queryParams.Get("to"); toDate != "" {
		if parsedDate, err := time.Parse("2006-01-02", toDate); err == nil {
			// Add one day to include the entire day
			endDate := parsedDate.AddDate(0, 0, 1)
			if args.Query["createdAt"] == nil {
				args.Query["createdAt"] = bson.M{}
			}
			args.Query["createdAt"].(bson.M)["$lt"] = endDate
		}
	}

	// Search functionality
	if search := queryParams.Get("search"); search != "" {
		// Create regex for case-insensitive search
		searchRegex := bson.M{"$regex": search, "$options": "i"}

		// Search in voucher code first (simple field search)
		searchConditions := []bson.M{
			{"code": searchRegex},
		}

		args.Query["$or"] = searchConditions
	}

	// Add sorting support
	sortBy := queryParams.Get("sort_by")
	sortOrder := queryParams.Get("sort_order")

	if sortBy != "" {
		order := 1 // ascending by default
		if sortOrder == "desc" {
			order = -1
		}

		// Map frontend sort fields to database fields
		switch sortBy {
		case "date", "created_at":
			args.Sort = bson.M{"createdAt": order}
		case "status":
			args.Sort = bson.M{"status": order}
		case "voucher_status":
			// For voucher_status sorting, we need to sort by is_used first, then by expired_at
			if order == 1 {
				args.Sort = bson.M{"is_used": 1, "expired_at": 1}
			} else {
				args.Sort = bson.M{"is_used": -1, "expired_at": -1}
			}
		case "points":
			args.Sort = bson.M{"points": order}
		case "is_used":
			args.Sort = bson.M{"is_used": order}
		case "expired_at":
			args.Sort = bson.M{"expired_at": order}
		default:
			args.Sort = bson.M{"createdAt": -1} // default sort
		}
	} else {
		args.Sort = bson.M{"createdAt": -1} // default sort by newest first
	}

	// Set to get all results without pagination
	args.PerPage = "1000000" // Large number to get all results
	args.Page = "1"

	voucherService := services.NewVoucherSvc(r.Context(), db)

	result, err := voucherService.GetForManagement(args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return just the vouchers array without pagination info
	response := map[string]interface{}{
		"vouchers": result.Vouchers,
		"total":    len(result.Vouchers),
	}

	j, _ := json.MarshalContext(r.Context(), response)
	w.Write(j)
}

// PostVouchersForManagement - POST endpoint for management vouchers with filters in request body
func PostVouchersForManagement(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	// Parse the filter from request body
	var filter map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		http.Error(w, "Invalid JSON in request body", http.StatusBadRequest)
		return
	}

	args, err := util.Paginate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Merge the provided filter with args.Query
	for key, value := range filter {
		args.Query[key] = value
	}

	// Handle special cases for status filtering (same logic as GET endpoint)
	if statusValue, exists := filter["status"]; exists {
		if statusStr, ok := statusValue.(string); ok && statusStr == "pending" {
			// For pending status, include both explicit "pending" and empty status fields
			args.Query["$or"] = []bson.M{
				{"status": "pending"},
				{"status": bson.M{"$exists": false}},
				{"status": ""},
			}
			delete(args.Query, "status") // Remove the original status field
		}
	}

	// Handle voucher_status filtering (computed field)
	if voucherStatusValue, exists := filter["voucher_status"]; exists {
		if voucherStatusStr, ok := voucherStatusValue.(string); ok {
			switch voucherStatusStr {
			case "used":
				args.Query["is_used"] = true
			case "active":
				args.Query["is_used"] = false
				// For active vouchers, we need to ensure they're not expired
				args.Query["$or"] = []bson.M{
					{"expired_at": bson.M{"$exists": false}},
					{"expired_at": bson.M{"$gt": time.Now()}},
				}
			case "expired":
				args.Query["is_used"] = false
				args.Query["expired_at"] = bson.M{"$lt": time.Now()}
			}
			delete(args.Query, "voucher_status") // Remove the original field
		}
	}

	// Handle is_used filtering
	if isUsedValue, exists := filter["is_used"]; exists {
		if isUsedStr, ok := isUsedValue.(string); ok {
			if isUsedStr == "true" {
				args.Query["is_used"] = true
			} else if isUsedStr == "false" {
				args.Query["is_used"] = false
			}
		} else if isUsedBool, ok := isUsedValue.(bool); ok {
			args.Query["is_used"] = isUsedBool
		}
		delete(args.Query, "is_used") // Remove the original field
	}

	// Handle user_id conversion to ObjectID
	if userIdValue, exists := filter["user_id"]; exists {
		if userIdStr, ok := userIdValue.(string); ok {
			if userObjId, err := primitive.ObjectIDFromHex(userIdStr); err == nil {
				args.Query["user_id"] = userObjId
			}
		}
		delete(args.Query, "user_id") // Remove the original field
	}

	// Handle service_id conversion to ObjectID
	if serviceIdValue, exists := filter["service_id"]; exists {
		if serviceIdStr, ok := serviceIdValue.(string); ok {
			if serviceObjId, err := primitive.ObjectIDFromHex(serviceIdStr); err == nil {
				args.Query["service_id"] = serviceObjId
			}
		}
		delete(args.Query, "service_id") // Remove the original field
	}

	// Handle date range filtering
	if fromDateValue, exists := filter["from"]; exists {
		if fromDateStr, ok := fromDateValue.(string); ok {
			if parsedDate, err := time.Parse("2006-01-02", fromDateStr); err == nil {
				if args.Query["createdAt"] == nil {
					args.Query["createdAt"] = bson.M{}
				}
				args.Query["createdAt"].(bson.M)["$gte"] = parsedDate
			}
		}
		delete(args.Query, "from") // Remove the original field
	}

	if toDateValue, exists := filter["to"]; exists {
		if toDateStr, ok := toDateValue.(string); ok {
			if parsedDate, err := time.Parse("2006-01-02", toDateStr); err == nil {
				// Add one day to include the entire day
				endDate := parsedDate.AddDate(0, 0, 1)
				if args.Query["createdAt"] == nil {
					args.Query["createdAt"] = bson.M{}
				}
				args.Query["createdAt"].(bson.M)["$lt"] = endDate
			}
		}
		delete(args.Query, "to") // Remove the original field
	}

	// Handle search functionality
	if searchValue, exists := filter["search"]; exists {
		if searchStr, ok := searchValue.(string); ok && searchStr != "" {
			// Create regex for case-insensitive search
			searchRegex := bson.M{"$regex": searchStr, "$options": "i"}

			// Search in voucher code first (simple field search)
			searchConditions := []bson.M{
				{"code": searchRegex},
			}

			args.Query["$or"] = searchConditions
		}
		delete(args.Query, "search") // Remove the original field
	}

	// Handle name filtering (user first name or last name)
	if nameValue, exists := filter["name"]; exists {
		if nameStr, ok := nameValue.(string); ok && nameStr != "" {
			args.Query["user_name_filter"] = nameStr
		}
		delete(args.Query, "name") // Remove the original field
	}

	// Handle email filtering (user email)
	if emailValue, exists := filter["email"]; exists {
		if emailStr, ok := emailValue.(string); ok && emailStr != "" {
			args.Query["user_email_filter"] = emailStr
		}
		delete(args.Query, "email") // Remove the original field
	}

	// Handle sorting
	if sortByValue, exists := filter["sort_by"]; exists {
		if sortByStr, ok := sortByValue.(string); ok {
			sortOrderValue, hasSortOrder := filter["sort_order"]
			sortOrder := "desc" // default to descending
			if hasSortOrder {
				if sortOrderStr, ok := sortOrderValue.(string); ok {
					sortOrder = sortOrderStr
				}
			}

			order := 1 // ascending by default
			if sortOrder == "desc" {
				order = -1
			}

			// Map frontend sort fields to database fields
			switch sortByStr {
			case "date", "created_at":
				args.Sort = bson.M{"createdAt": order}
			case "status":
				args.Sort = bson.M{"status": order}
			case "voucher_status":
				// For voucher_status sorting, we need to sort by is_used first, then by expired_at
				if order == 1 {
					args.Sort = bson.M{"is_used": 1, "expired_at": 1}
				} else {
					args.Sort = bson.M{"is_used": -1, "expired_at": -1}
				}
			case "points":
				args.Sort = bson.M{"points": order}
			case "is_used":
				args.Sort = bson.M{"is_used": order}
			case "expired_at":
				args.Sort = bson.M{"expired_at": order}
			default:
				args.Sort = bson.M{"createdAt": -1} // default sort
			}
		}
		delete(args.Query, "sort_by")    // Remove the original field
		delete(args.Query, "sort_order") // Remove the original field
	} else {
		args.Sort = bson.M{"createdAt": -1} // default sort by newest first
	}

	voucherService := services.NewVoucherSvc(r.Context(), db)

	result, err := voucherService.GetForManagement(args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate proper pagination
	perPage := 10
	if args.Page != "" {
		if p, err := strconv.Atoi(args.Page); err == nil && p > 0 {
			// page number is handled in pagination calculation
		}
	}
	if args.PerPage != "" {
		if pp, err := strconv.Atoi(args.PerPage); err == nil && pp > 0 {
			perPage = pp
		}
	}

	// Count total vouchers with same filters
	countArgs := &util.ListArgs{
		Query:   args.Query,
		PerPage: "1000000", // Large number to get all
		Page:    "1",
	}
	allResult, _ := voucherService.GetForManagement(countArgs)
	totalCount := len(allResult.Vouchers)
	totalPages := (totalCount + perPage - 1) / perPage

	// Create custom response with correct pagination
	response := map[string]interface{}{
		"vouchers": result.Vouchers,
		"pagination": map[string]interface{}{
			"totalPages": totalPages,
			"perPage":    perPage,
			"totalCount": totalCount,
		},
	}

	j, _ := json.MarshalContext(r.Context(), response)
	w.Write(j)
}

// UpdateVoucherStatus - Update voucher status and comments
func UpdateVoucherStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	type UpdateStatusRequest struct {
		Status   string `json:"status"`
		Comments string `json:"comments"`
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate status
	validStatuses := []string{
		models.VoucherStatusPending,
		models.VoucherStatusChecked,
		models.VoucherStatusInProgress,
		models.VoucherStatusCanceled,
		models.VoucherStatusBooked,
		models.VoucherStatusNotBooked,
	}

	isValidStatus := false
	for _, validStatus := range validStatuses {
		if req.Status == validStatus {
			isValidStatus = true
			break
		}
	}

	if !isValidStatus {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	voucherService := services.NewVoucherSvc(r.Context(), db)

	result, err := voucherService.UpdateStatus(id, req.Status, req.Comments)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Clear cache
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteMul(constants.VOUCHER, db), 0)
	caching.Rdb.BatchDeletionKeysByPattern(util.CacheKeyDeleteOne(constants.VOUCHER, db, id), 0)

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

// GetVoucherStatusOptions - Get available voucher status options for frontend dropdowns
func GetVoucherStatusOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	statusOptions := []map[string]string{
		{"value": models.VoucherStatusPending, "label": "Pending"},
		{"value": models.VoucherStatusChecked, "label": "Checked"},
		{"value": models.VoucherStatusInProgress, "label": "In Progress"},
		{"value": models.VoucherStatusCanceled, "label": "Canceled"},
		{"value": models.VoucherStatusBooked, "label": "Booked"},
		{"value": models.VoucherStatusNotBooked, "label": "Not Booked"},
	}

	voucherStatusOptions := []map[string]string{
		{"value": "active", "label": "Active"},
		{"value": "used", "label": "Used"},
		{"value": "expired", "label": "Expired"},
	}

	response := map[string]interface{}{
		"status_options":         statusOptions,
		"voucher_status_options": voucherStatusOptions,
		"used_options": []map[string]interface{}{
			{"value": true, "label": "Used"},
			{"value": false, "label": "Not Used"},
		},
		"sort_options": []map[string]string{
			{"value": "created_at", "label": "Date Created"},
			{"value": "status", "label": "Status"},
			{"value": "voucher_status", "label": "Voucher Status"},
			{"value": "points", "label": "Points"},
			{"value": "is_used", "label": "Usage Status"},
			{"value": "expired_at", "label": "Expiry Date"},
		},
	}

	j, _ := json.MarshalContext(r.Context(), response)
	w.Write(j)
}

// GetVoucherStats - Get voucher statistics for management dashboard
func GetVoucherStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	voucherService := services.NewVoucherSvc(r.Context(), db)

	// Get total vouchers count by getting all vouchers without pagination limit
	allArgs := &util.ListArgs{
		Query:   bson.M{"trash": bson.M{"$ne": true}},
		PerPage: "1000000", // Large number to get all
		Page:    "1",
	}
	allResult, _ := voucherService.GetForManagement(allArgs)
	totalVouchers := int64(len(allResult.Vouchers))

	// Count different types
	usedVouchers := int64(0)
	expiredVouchers := int64(0)
	activeVouchers := int64(0)
	statusCounts := make(map[string]int64)

	// Initialize status counts
	statusOptions := []string{
		models.VoucherStatusPending,
		models.VoucherStatusChecked,
		models.VoucherStatusInProgress,
		models.VoucherStatusCanceled,
		models.VoucherStatusBooked,
		models.VoucherStatusNotBooked,
	}
	for _, status := range statusOptions {
		statusCounts[status] = 0
	}

	// Count by iterating through results
	now := time.Now()
	for _, voucher := range allResult.Vouchers {
		if voucher.IsUsed {
			usedVouchers++
		} else if !voucher.ExpiredAt.IsZero() && voucher.ExpiredAt.Before(now) {
			expiredVouchers++
		} else {
			activeVouchers++
		}

		if voucher.Status != "" {
			statusCounts[voucher.Status]++
		} else {
			statusCounts[models.VoucherStatusPending]++
		}
	}

	// Calculate usage rate
	usageRate := float64(0)
	if totalVouchers > 0 {
		usageRate = float64(usedVouchers) / float64(totalVouchers) * 100
	}

	response := map[string]interface{}{
		"total_vouchers":   totalVouchers,
		"used_vouchers":    usedVouchers,
		"unused_vouchers":  totalVouchers - usedVouchers,
		"expired_vouchers": expiredVouchers,
		"active_vouchers":  activeVouchers,
		"usage_rate":       usageRate,
		"status_breakdown": statusCounts,
		"voucher_status_breakdown": map[string]int64{
			"active":  activeVouchers,
			"used":    usedVouchers,
			"expired": expiredVouchers,
		},
		"summary": map[string]interface{}{
			"pending_action": statusCounts[models.VoucherStatusPending] + statusCounts[models.VoucherStatusInProgress],
			"completed":      statusCounts[models.VoucherStatusChecked] + statusCounts[models.VoucherStatusBooked],
			"cancelled":      statusCounts[models.VoucherStatusCanceled] + statusCounts[models.VoucherStatusNotBooked],
		},
	}

	j, _ := json.MarshalContext(r.Context(), response)
	w.Write(j)
}

// FilterVouchers - General filter endpoint that accepts any fields in POST body
func FilterVouchers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	// Parse the filter from request body
	var filter map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		http.Error(w, "Invalid JSON in request body", http.StatusBadRequest)
		return
	}

	// Get pagination arguments
	args, err := util.Paginate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	voucherService := services.NewVoucherSvc(r.Context(), db)
	result, err := voucherService.FilterVouchers(filter, args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

// GetVouchersForManagementV2 - POST endpoint for management vouchers with structured query conditions
func GetVouchersForManagementV2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	// Get pagination parameters
	args, err := util.Paginate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Extract skip and limit from args
	skip := 0
	limit := 10
	if args.Page != "" {
		if p, err := strconv.Atoi(args.Page); err == nil && p > 0 {
			skip = (p - 1) * limit
		}
	}
	if args.PerPage != "" {
		if pp, err := strconv.Atoi(args.PerPage); err == nil && pp > 0 {
			limit = pp
			// Recalculate skip with new limit
			if args.Page != "" {
				if p, err := strconv.Atoi(args.Page); err == nil && p > 0 {
					skip = (p - 1) * limit
				}
			}
		}
	}

	// Parse the query conditions from request body
	var conditions query.Conditions
	if err := json.NewDecoder(r.Body).Decode(&conditions); err != nil {
		http.Error(w, "Invalid JSON in request body", http.StatusBadRequest)
		return
	}

	// Validate the conditions
	if err := conditions.CheckValid(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid query conditions: %v", err), http.StatusBadRequest)
		return
	}

	voucherService := services.NewVoucherSvc(r.Context(), db)
	result, err := voucherService.GetForManagementV2(&conditions, skip, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

// FixVoucherStatusData - Admin endpoint to fix voucher status data inconsistency
func FixVoucherStatusData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client

	voucherService := services.NewVoucherSvc(r.Context(), db)
	err := voucherService.FixVoucherStatusData()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "Voucher status data has been fixed successfully",
		"success": true,
	}

	j, _ := json.MarshalContext(r.Context(), response)
	w.Write(j)
}
