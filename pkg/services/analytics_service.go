package services

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"loyalty/pkg/constants"
	"loyalty/pkg/database"
	"loyalty/pkg/models"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AnalyticsService interface {
	GetDashboardAnalytics(client string) (*models.DashboardAnalytics, error)
	GetUserCount(accessToken string, client string, fromDate, toDate time.Time) ([]map[string]interface{}, error)
	GetTransactionAnalytics(filters map[string]interface{}) (*models.TransactionAnalytics, error)
	GetSurveyAnalytics(filters map[string]interface{}) (*models.SurveyAnalytics, error)
	GetTransactionsByService(filters map[string]interface{}) ([]models.TransactionServiceStats, error)
	GetPointsPerDay(filters map[string]interface{}) ([]models.DailyPointsStats, error)
	GetEnhancedTransactionAnalytics(filters map[string]interface{}) (*models.EnhancedTransactionAnalytics, error)

	GetTimeSeriesEnhancedTransactionAnalytics(fromDate, toDate time.Time) (*models.TimeSeriesEnhancedTransactionAnalytics, error)
	GetTimeSeriesTransactionAnalytics(fromDate, toDate time.Time) (*models.TimeSeriesTransactionAnalytics, error)
	GetTimeSeriesSurveyAnalytics(fromDate, toDate time.Time) (*models.TimeSeriesSurveyAnalytics, error)

	GetTimeSeriesExternalAnalytics(xclient, accessToken string, fromDate, toDate time.Time) (*models.TimeSeriesExternalAnalytics, error)
	GetTimeSeriesBookingAnalytics(xclient, accessToken string, fromDate, toDate time.Time) (*models.TimeSeriesBookingAnalytics, error)
	GetTimeSeriesOnlineConsultationAnalytics(xclient, accessToken string, fromDate, toDate time.Time) (*models.TimeSeriesOnlineConsultationAnalytics, error)
	GetTimeSeriesSmileTestDriveAnalytics(xclient, accessToken string, fromDate, toDate time.Time) (*models.TimeSeriesSmileTestDriveAnalytics, error)
	GetTotalBookings(xclient, accessToken string, query string) (interface{}, error)
	GetTotalSmileTestDrives(xclient, accessToken string, query string) (interface{}, error)
	GetTotalOnlineConsultations(xclient, accessToken string, query string) (interface{}, error)
	GetTotalCounts(xclient, accessToken string, query string) (interface{}, error)
	GetAppSharingAnalytics(fromDate, toDate time.Time) (int64, error)
	GetVoucherRedemptionAnalytics(fromDate, toDate time.Time) (int64, error)
	GetVoucherRedemptionByServiceType(filters map[string]interface{}) (*models.VoucherRedemptionAnalytics, error)
	GetTimeSeriesVoucherRedemptionAnalytics(fromDate, toDate time.Time) (*models.TimeSeriesVoucherRedemptionAnalytics, error)
	GetVoucherUsageByServiceType(filters map[string]interface{}) (*models.VoucherUsageAnalytics, error)
	GetTimeSeriesVoucherUsageAnalytics(fromDate, toDate time.Time) (*models.TimeSeriesVoucherUsageAnalytics, error)
}

type analyticsSvc struct {
	ctx context.Context
	db  string
}

func NewAnalyticsService(ctx context.Context, db string) AnalyticsService {
	return &analyticsSvc{ctx: ctx, db: db}
}

func (s *analyticsSvc) GetDashboardAnalytics(client string) (*models.DashboardAnalytics, error) {
	// Get user analytics from auth service
	totalUsers, err := s.getTotalUsersCount(client)
	if err != nil {
		// Fallback to counting wallets if auth service fails
		userWalletColl := database.Mdb.Database(s.db).Collection("loyaltyUserWallets")
		totalUsers, _ = userWalletColl.CountDocuments(s.ctx, bson.M{"trash": bson.M{"$ne": true}})
	}

	// Get transaction analytics
	transactionColl := database.Mdb.Database(s.db).Collection("loyaltyTransactions")
	totalRedemptions, _ := transactionColl.CountDocuments(s.ctx, bson.M{
		"type":  bson.M{"$in": []constants.TxnType{constants.TXN_VOUCHER_USED}},
		"trash": bson.M{"$ne": true},
	})

	appShares, _ := transactionColl.CountDocuments(s.ctx, bson.M{
		"type":  constants.TXN_APP_SHARED,
		"trash": bson.M{"$ne": true},
	})

	// Get transactions by service
	transactionsByService, err := s.getTransactionsByService(transactionColl)
	if err != nil {
		// Initialize empty map if there's an error
		transactionsByService = make(map[string]models.TransactionServiceStats)
	}

	// Get points transferred per day
	pointsPerDay, err := s.getPointsTransferredPerDay(transactionColl)
	if err != nil {
		// Initialize empty map if there's an error
		pointsPerDay = make(map[string]float64)
	}

	// Get pending points
	pendingPointsCount, totalPendingPoints, err := s.getPendingPointsStats()
	if err != nil {
		// Set defaults if there's an error
		pendingPointsCount = 0
		totalPendingPoints = 0
	}

	// Get user wallet collection for user analytics
	userWalletColl := database.Mdb.Database(s.db).Collection("loyaltyUserWallets")

	// Get voucher redemption analytics by service type
	voucherRedemptionAnalytics, err := s.GetVoucherRedemptionByServiceType(nil)
	if err != nil {
		// Set defaults if there's an error
		voucherRedemptionAnalytics = &models.VoucherRedemptionAnalytics{
			SpecialOccasionDiscounts: models.VoucherDiscountTypeStats{
				TotalRedemptions: 0,
				TotalUsages:      0,
				ServiceGroups:    make(map[string]models.VoucherServiceGroup),
			},
			RedeemableDiscounts: models.VoucherDiscountTypeStats{
				TotalRedemptions: 0,
				TotalUsages:      0,
				ServiceGroups:    make(map[string]models.VoucherServiceGroup),
			},
			TotalRedemptions: 0,
		}
	}

	// Get voucher usage analytics by service type (total usage count)
	voucherUsageAnalytics, err := s.GetVoucherUsageByServiceType(nil)
	if err != nil {
		// Set defaults if there's an error
		voucherUsageAnalytics = &models.VoucherUsageAnalytics{
			SpecialOccasionDiscounts: models.VoucherDiscountTypeStats{
				TotalRedemptions: 0,
				TotalUsages:      0,
				ServiceGroups:    make(map[string]models.VoucherServiceGroup),
			},
			RedeemableDiscounts: models.VoucherDiscountTypeStats{
				TotalRedemptions: 0,
				TotalUsages:      0,
				ServiceGroups:    make(map[string]models.VoucherServiceGroup),
			},
			TotalUsages: 0,
		}
	}

	// Get survey analytics
	surveyAnswerColl := database.Mdb.Database(s.db).Collection("loyaltySurveyAnswers")
	surveysByCategory := make(map[string]models.SurveyCategoryStats)

	surveyPipeline := []bson.M{
		{"$match": bson.M{"trash": bson.M{"$ne": true}}},
		{"$lookup": bson.M{
			"from":         "loyaltySurveys",
			"localField":   "survey_id",
			"foreignField": "_id",
			"as":           "survey",
		}},
		{"$unwind": "$survey"},
		{"$group": bson.M{
			"_id":   "$survey.title.en",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := surveyAnswerColl.Aggregate(s.ctx, surveyPipeline)
	if cursor != nil {
		defer cursor.Close(s.ctx)
		for cursor.Next(s.ctx) {
			var result bson.M
			if cursor.Decode(&result) == nil {
				category := result["_id"].(string)
				var count int64
				// Handle both int32 and int64 for count
				if c, ok := result["count"].(int32); ok {
					count = int64(c)
				} else if c, ok := result["count"].(int64); ok {
					count = c
				}

				categoryKey := "other"
				if strings.Contains(strings.ToLower(category), "facility") {
					categoryKey = "facility"
				} else if strings.Contains(strings.ToLower(category), "dentist") {
					categoryKey = "dentist"
				} else if strings.Contains(strings.ToLower(category), "customer") || strings.Contains(strings.ToLower(category), "service") {
					categoryKey = "customer_service"
				}

				if existing, exists := surveysByCategory[categoryKey]; exists {
					surveysByCategory[categoryKey] = models.SurveyCategoryStats{
						Count: existing.Count + count,
						// AverageRating: existing.AverageRating,
					}
				} else {
					surveysByCategory[categoryKey] = models.SurveyCategoryStats{
						Count: count,
						// AverageRating: 0.0,
					}
				}
			}
		}
	}

	// Get time-based analytics
	now := time.Now()
	last30Days := now.AddDate(0, 0, -30)
	last12Months := now.AddDate(0, -12, 0)

	// Users by month
	usersByMonth := make(map[string]int64)
	usersByMonthPipeline := []bson.M{
		{"$match": bson.M{
			"createdAt": bson.M{"$gte": last12Months},
			"trash":     bson.M{"$ne": true},
		}},
		{"$group": bson.M{
			"_id": bson.M{
				"year":  bson.M{"$year": "$createdAt"},
				"month": bson.M{"$month": "$createdAt"},
			},
			"count": bson.M{"$sum": 1},
		}},
	}

	usersByMonthCursor, err := userWalletColl.Aggregate(s.ctx, usersByMonthPipeline)
	if err == nil {
		defer usersByMonthCursor.Close(s.ctx)
		for usersByMonthCursor.Next(s.ctx) {
			var result bson.M
			if err := usersByMonthCursor.Decode(&result); err != nil {
				continue
			}

			id := result["_id"].(bson.M)
			year := id["year"].(int32)
			month := id["month"].(int32)

			var count int64
			if val, ok := result["count"].(int32); ok {
				count = int64(val)
			} else if val, ok := result["count"].(int64); ok {
				count = val
			}

			key := fmt.Sprintf("%d-%02d", year, month)
			usersByMonth[key] = count
		}
	}

	// New users by day
	newUsersByDay := make(map[string]int64)
	newUsersByDayPipeline := []bson.M{
		{"$match": bson.M{
			"createdAt": bson.M{"$gte": last30Days},
			"trash":     bson.M{"$ne": true},
		}},
		{"$group": bson.M{
			"_id": bson.M{
				"year":  bson.M{"$year": "$createdAt"},
				"month": bson.M{"$month": "$createdAt"},
				"day":   bson.M{"$dayOfMonth": "$createdAt"},
			},
			"count": bson.M{"$sum": 1},
		}},
	}

	newUsersByDayCursor, err := userWalletColl.Aggregate(s.ctx, newUsersByDayPipeline)
	if err == nil {
		defer newUsersByDayCursor.Close(s.ctx)
		for newUsersByDayCursor.Next(s.ctx) {
			var result bson.M
			if err := newUsersByDayCursor.Decode(&result); err != nil {
				continue
			}

			id := result["_id"].(bson.M)
			year := id["year"].(int32)
			month := id["month"].(int32)
			day := id["day"].(int32)

			var count int64
			if val, ok := result["count"].(int32); ok {
				count = int64(val)
			} else if val, ok := result["count"].(int64); ok {
				count = val
			}

			key := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC).Format("2006-01-02")
			newUsersByDay[key] = count
		}
	}

	return &models.DashboardAnalytics{
		UserAnalytics: models.UserAnalytics{
			TotalUsers:     totalUsers,
			UsersByMonth:   usersByMonth,
			NewUsersByDay:  newUsersByDay,
			TotalAppShares: appShares,
		},
		TransactionAnalytics: models.TransactionAnalytics{
			TotalRedemptions:      totalRedemptions,
			TransactionsByService: transactionsByService,
			PointsPerDay:          pointsPerDay,
			PendingPointsCount:    pendingPointsCount,
			TotalPendingPoints:    totalPendingPoints,
		},
		SurveyAnalytics: models.SurveyAnalytics{
			SurveysByCategory: surveysByCategory,
		},
		VoucherRedemptionAnalytics: *voucherRedemptionAnalytics,
		VoucherUsageAnalytics:      *voucherUsageAnalytics,
	}, nil
}

// GetTotalBookings gets total bookings count from types service
func (s *analyticsSvc) GetTotalBookings(xclient, accessToken string, query string) (interface{}, error) {
	return s.getTotalCountFromTypesService(xclient, `https://services.larsa.io/types/booking`, query, "total_bookings", accessToken)
}

// GetTotalSmileTestDrives gets total smile test drives count from types service
func (s *analyticsSvc) GetTotalSmileTestDrives(xclient, accessToken string, query string) (interface{}, error) {
	return s.getTotalCountFromTypesService(xclient, "https://services.larsa.io/types/smiletestdrive", query, "total_smile_test_drives", accessToken)
}

// GetTotalOnlineConsultations gets total online consultations count from types service
func (s *analyticsSvc) GetTotalOnlineConsultations(xclient, accessToken string, query string) (interface{}, error) {
	return s.getTotalCountFromTypesService(xclient, "https://services.larsa.io/types/onlineconsultation", query, "total_online_consultations", accessToken)
}

// GetTotalCounts gets all total counts from types service
func (s *analyticsSvc) GetTotalCounts(xclient, accessToken string, query string) (interface{}, error) {
	bookings, err := s.GetTotalBookings(xclient, accessToken, query)
	if err != nil {
		return nil, err
	}

	smileTestDrives, err := s.GetTotalSmileTestDrives(xclient, accessToken, query)
	if err != nil {
		return nil, err
	}

	onlineConsultations, err := s.GetTotalOnlineConsultations(xclient, accessToken, query)
	if err != nil {
		return nil, err
	}

	// Extract values from responses
	bookingsCount := 0
	smileTestDrivesCount := 0
	onlineConsultationsCount := 0

	if bookingsMap, ok := bookings.(map[string]interface{}); ok {
		if count, exists := bookingsMap["total_bookings"]; exists {
			if countInt, ok := count.(float64); ok {
				bookingsCount = int(countInt)
			}
		}
	}

	if smileMap, ok := smileTestDrives.(map[string]interface{}); ok {
		if count, exists := smileMap["total_smile_test_drives"]; exists {
			if countInt, ok := count.(float64); ok {
				smileTestDrivesCount = int(countInt)
			}
		}
	}

	if consultationsMap, ok := onlineConsultations.(map[string]interface{}); ok {
		if count, exists := consultationsMap["total_online_consultations"]; exists {
			if countInt, ok := count.(float64); ok {
				onlineConsultationsCount = int(countInt)
			}
		}
	}

	return map[string]interface{}{
		"total_bookings":             bookingsCount,
		"total_online_consultations": onlineConsultationsCount,
		"total_smile_test_drives":    smileTestDrivesCount,
	}, nil
}

// getTotalCountFromTypesService is a helper function to get total count from types service
func (s *analyticsSvc) getTotalCountFromTypesService(xclient string, url, query, responseKey, accessToken string) (interface{}, error) {
	newUrl := url + `?` + query
	req, err := http.NewRequest("GET", newUrl, nil)
	if err != nil {
		return nil, err
	}

	// Set headers as requested
	req.Header.Set("x-client", xclient)
	req.Header.Set("x-access-token", accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{
			"error":  err.Error(),
			"status": 0,
		}, nil
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{
			"error":  "Failed to read response body: " + err.Error(),
			"status": resp.StatusCode,
		}, nil
	}

	// Parse JSON response
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return map[string]interface{}{
			"error": "Failed to parse JSON response: " + err.Error(),
		}, nil
	}

	// Extract totalCount from pagination
	if pagination, ok := jsonResponse["pagination"].(map[string]interface{}); ok {
		if totalCount, exists := pagination["totalCount"]; exists {
			return map[string]interface{}{
				responseKey: totalCount,
			}, nil
		}
	}

	// If pagination or totalCount not found, return error
	return map[string]interface{}{
		"error": "totalCount not found in response",
	}, nil
}

// Helper function to get map keys for debugging
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// getTotalUsersCount fetches the total number of users from the auth service
func (s *analyticsSvc) getTotalUsersCount(client string) (int64, error) {
	// Build URL for users service
	usersServiceURL := os.Getenv("USERS_URL")
	if usersServiceURL == "" {
		usersServiceURL = "http://localhost:3201"
	}

	url := fmt.Sprintf("%s/users/count", usersServiceURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-client", client)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		// If it's a subscription error, log it and let the caller handle fallback
		if resp.StatusCode == 500 && strings.Contains(string(body), "subscription") {
			// Users service subscription issue - caller will handle fallback
		}

		return 0, fmt.Errorf("users service returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var response struct {
		Success bool  `json:"success"`
		Count   int64 `json:"count"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return 0, err
	}

	if !response.Success {
		return 0, fmt.Errorf("users service returned unsuccessful response")
	}

	return response.Count, nil
}

// countUsersFromDatabase counts users from the database as a fallback
func (s *analyticsSvc) countUsersFromDatabase(ctx context.Context) (int64, error) {
	// Fallback to counting users from the database
	userWalletColl := database.Mdb.Database(s.db).Collection("loyaltyUserWallets")
	count, err := userWalletColl.CountDocuments(ctx, bson.M{"trash": bson.M{"$ne": true}})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// countUsersFromWalletsWithFilters counts users from local wallet collection with date filters
func (s *analyticsSvc) countUsersFromWalletsWithFilters(filters map[string]interface{}) (int64, error) {
	userWalletColl := database.Mdb.Database(s.db).Collection("loyaltyUserWallets")

	// Build match filter based on provided filters
	matchFilter := bson.M{"trash": bson.M{"$ne": true}}

	if filters != nil {
		if startDate, ok := filters["start_date"].(time.Time); ok {
			matchFilter["createdAt"] = bson.M{"$gte": startDate}
		}
		if endDate, ok := filters["end_date"].(time.Time); ok {
			if existingFilter, exists := matchFilter["createdAt"]; exists {
				matchFilter["createdAt"] = bson.M{"$gte": existingFilter.(bson.M)["$gte"], "$lte": endDate}
			} else {
				matchFilter["createdAt"] = bson.M{"$lte": endDate}
			}
		}
		if year, ok := filters["year"].(int); ok {
			startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
			endOfYear := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
			matchFilter["createdAt"] = bson.M{"$gte": startOfYear, "$lt": endOfYear}
		}
		if month, ok := filters["month"].(int); ok {
			year := time.Now().Year()
			if y, hasYear := filters["year"].(int); hasYear {
				year = y
			}
			startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
			endOfMonth := startOfMonth.AddDate(0, 1, 0)
			matchFilter["createdAt"] = bson.M{"$gte": startOfMonth, "$lt": endOfMonth}
		}
		if days, ok := filters["days"].(int); ok {
			since := time.Now().AddDate(0, 0, -days)
			matchFilter["createdAt"] = bson.M{"$gte": since}
		}
	}

	count, err := userWalletColl.CountDocuments(s.ctx, matchFilter)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// getTransactionsByService gets transaction statistics grouped by service
func (s *analyticsSvc) getTransactionsByService(transactionColl *mongo.Collection) (map[string]models.TransactionServiceStats, error) {
	// Pipeline to group transactions by service
	pipeline := []bson.M{
		{"$match": bson.M{
			"trash": bson.M{"$ne": true},
			"type": bson.M{"$in": []constants.TxnType{
				constants.TXN_VOUCHER_USED,
				constants.TXN_SERVICE_REDEEMED,
				constants.TXN_POINTS_ADDED_TO_WALLET,
			}},
		}},
		{"$lookup": bson.M{
			"from":         "loyaltyServices",
			"localField":   "service_id",
			"foreignField": "_id",
			"as":           "service",
		}},
		{"$unwind": bson.M{
			"path":                       "$service",
			"preserveNullAndEmptyArrays": false,
		}},
		{"$group": bson.M{
			"_id":          "$service.name",
			"count":        bson.M{"$sum": 1},
			"total_amount": bson.M{"$sum": "$amount"},
			"type":         bson.M{"$first": "$type"},
		}},
	}

	cursor, err := transactionColl.Aggregate(s.ctx, pipeline)
	if err != nil {
		return make(map[string]models.TransactionServiceStats), nil
	}
	defer cursor.Close(s.ctx)

	result := make(map[string]models.TransactionServiceStats)
	for cursor.Next(s.ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		serviceName, ok := doc["_id"].(string)
		if !ok {
			continue
		}

		count := int64(0)
		if c, ok := doc["count"].(int32); ok {
			count = int64(c)
		} else if c, ok := doc["count"].(int64); ok {
			count = c
		}

		totalAmount := float64(0)
		if amt, ok := doc["total_amount"].(float64); ok {
			totalAmount = amt
		} else if amt, ok := doc["total_amount"].(int32); ok {
			totalAmount = float64(amt)
		} else if amt, ok := doc["total_amount"].(int64); ok {
			totalAmount = float64(amt)
		}

		txnType := constants.TxnType(0)
		if t, ok := doc["type"].(constants.TxnType); ok {
			txnType = t
		} else if t, ok := doc["type"].(int32); ok {
			txnType = constants.TxnType(t)
		} else if t, ok := doc["type"].(int64); ok {
			txnType = constants.TxnType(t)
		}

		result[serviceName] = models.TransactionServiceStats{
			Count:       count,
			TotalAmount: totalAmount,
			Type:        txnType,
		}
	}

	return result, nil
}

// getPointsTransferredPerDay gets the total points transferred per day
func (s *analyticsSvc) getPointsTransferredPerDay(transactionColl *mongo.Collection) (map[string]float64, error) {
	// Get points transferred in the last 30 days
	now := time.Now()
	last30Days := now.AddDate(0, 0, -30)

	pipeline := []bson.M{
		{"$match": bson.M{
			"createdAt": bson.M{"$gte": last30Days},
			"trash":     bson.M{"$ne": true},
			"type": bson.M{"$in": []constants.TxnType{
				constants.TXN_POINTS_ADDED_TO_WALLET,
				constants.TXN_VOUCHER_USED,
			}},
		}},
		{"$group": bson.M{
			"_id": bson.M{
				"year":  bson.M{"$year": "$createdAt"},
				"month": bson.M{"$month": "$createdAt"},
				"day":   bson.M{"$dayOfMonth": "$createdAt"},
			},
			"total_points": bson.M{"$sum": "$amount"},
		}},
		{"$sort": bson.M{"_id.year": 1, "_id.month": 1, "_id.day": 1}},
	}

	cursor, err := transactionColl.Aggregate(s.ctx, pipeline)
	if err != nil {
		return make(map[string]float64), nil
	}
	defer cursor.Close(s.ctx)

	result := make(map[string]float64)
	for cursor.Next(s.ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		id := doc["_id"].(bson.M)
		year := id["year"].(int32)
		month := id["month"].(int32)
		day := id["day"].(int32)
		date := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC).Format("2006-01-02")

		totalPoints := float64(0)
		if points, ok := doc["total_points"].(float64); ok {
			totalPoints = points
		} else if points, ok := doc["total_points"].(int32); ok {
			totalPoints = float64(points)
		} else if points, ok := doc["total_points"].(int64); ok {
			totalPoints = float64(points)
		}
		result[date] = totalPoints
	}

	return result, nil
}

// getPendingPointsStats gets statistics about pending challenge submissions (not yet accepted)
func (s *analyticsSvc) getPendingPointsStats() (int64, float64, error) {
	// Get challenge answers collection to find pending (non-accepted) submissions
	challengeAnswerColl := database.Mdb.Database(s.db).Collection("loyaltyChallengeAnswers")

	// Pipeline to get pending challenge answers and their potential points
	pendingPointsPipeline := []bson.M{
		{
			// Match non-accepted, non-trashed challenge answers
			"$match": bson.M{
				"is_accepted": false,
				"trash":       bson.M{"$ne": true},
			},
		},
		{
			// Lookup challenge details to get points
			"$lookup": bson.M{
				"from":         "loyaltyChallenges",
				"localField":   "challenge_id",
				"foreignField": "_id",
				"as":           "challenge",
			},
		},
		{
			// Unwind challenge array
			"$unwind": bson.M{
				"path":                       "$challenge",
				"preserveNullAndEmptyArrays": false, // Only include answers with valid challenges
			},
		},
		{
			// Group to get count and total pending points
			"$group": bson.M{
				"_id":           nil,
				"total_pending": bson.M{"$sum": "$challenge.points"},
				"count":         bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := challengeAnswerColl.Aggregate(s.ctx, pendingPointsPipeline)
	if err != nil {
		return 0, 0, err
	}
	defer cursor.Close(s.ctx)

	var pendingPointsCount int64
	var totalPendingPoints float64
	if cursor.Next(s.ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return 0, 0, err
		}

		// Handle both int32 and int64 for count
		if count, ok := result["count"].(int32); ok {
			pendingPointsCount = int64(count)
		} else if count, ok := result["count"].(int64); ok {
			pendingPointsCount = count
		}

		// Handle both int32 and int64 for total_pending
		if total, ok := result["total_pending"].(int32); ok {
			totalPendingPoints = float64(total)
		} else if total, ok := result["total_pending"].(int64); ok {
			totalPendingPoints = float64(total)
		} else if total, ok := result["total_pending"].(float64); ok {
			totalPendingPoints = total
		}
	}

	return pendingPointsCount, totalPendingPoints, nil
}

// GetUserCount returns the incremental count of users created within each specific time period
func (s *analyticsSvc) GetUserCount(accessToken string, client string, fromDate, toDate time.Time) ([]map[string]interface{}, error) {
	// Build URL for users service
	usersServiceURL := os.Getenv("USERS_URL")
	if usersServiceURL == "" {
		usersServiceURL = "https://services.larsa.io/auth"
	}

	// Get ALL users by handling pagination
	var allUsers []struct {
		ID        string      `json:"_id"`
		CreatedAt interface{} `json:"createdAt"`
	}

	offset := 0
	perPage := 40 // Use the same page size as shown in the example (40 per page)
	totalCount := int64(0)

	for {
		// Build query parameters for this page
		params := url.Values{}
		// Try different pagination parameter names
		page := (offset / perPage) + 1
		params.Add("page", fmt.Sprintf("%d", page))     // Try 'page' instead of 'offset'
		params.Add("limit", fmt.Sprintf("%d", perPage)) // Try 'limit' instead of 'perPage'

		// Make request to users service
		requestURL := fmt.Sprintf("%s/users?%s", usersServiceURL, params.Encode())

		req, err := http.NewRequest("GET", requestURL, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-access-token", accessToken)
		req.Header.Set("x-client", client)

		httpClient := &http.Client{Timeout: 15 * time.Second}
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error calling users service: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("users service returned status %d: %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading response: %v", err)
		}

		var response struct {
			Users []struct {
				ID        string      `json:"_id"`
				CreatedAt interface{} `json:"createdAt"`
			} `json:"users"`
			Pagination struct {
				TotalCount int64 `json:"totalCount"`
				TotalPages int   `json:"totalPages"`
				PerPage    int   `json:"perPage"`
			} `json:"pagination"`
		}

		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}

		// Store pagination info from first response
		if offset == 0 {
			totalCount = response.Pagination.TotalCount
		}

		// Add users from this page to our collection
		allUsers = append(allUsers, response.Users...)

		// Check if we've got all users
		if len(response.Users) == 0 || len(allUsers) >= int(totalCount) {
			break
		}

		offset += perPage
	}

	// Remove duplicates for processing
	userIDMap := make(map[string]bool)
	duplicateCount := 0

	for _, user := range allUsers {
		// Check for duplicates
		if userIDMap[user.ID] {
			duplicateCount++
		} else {
			userIDMap[user.ID] = true
		}
	}

	if duplicateCount > 0 {
		// Remove duplicates for processing
		uniqueUsers := make([]struct {
			ID        string      `json:"_id"`
			CreatedAt interface{} `json:"createdAt"`
		}, 0, len(userIDMap))

		processedIDs := make(map[string]bool)
		for _, user := range allUsers {
			if !processedIDs[user.ID] {
				uniqueUsers = append(uniqueUsers, user)
				processedIDs[user.ID] = true
			}
		}
		allUsers = uniqueUsers
	}

	// Generate 10 time intervals from the date range
	timeIntervals := s.generateTimeIntervals(fromDate, toDate)

	// Initialize result array
	var result []map[string]interface{}

	// For each time interval, count users created only within that specific period (incremental)
	for i, interval := range timeIntervals {
		incrementalUserCount := int64(0)

		// Parse interval start and end dates
		intervalStart, _ := time.Parse("2006-01-02", interval.FromDate)
		intervalEnd, _ := time.Parse("2006-01-02", interval.ToDate)
		// Make end date inclusive (end of day)
		intervalEnd = intervalEnd.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		// Count only users created within this specific period
		for _, user := range allUsers {
			// Extract creation date using the helper function
			createdAt, err := extractUserCreationDate(user)
			if err != nil {
				// Skip users where we can't extract creation date
				continue
			}

			// Count users created only within this specific period (incremental counts)
			if (createdAt.After(intervalStart) || createdAt.Equal(intervalStart)) &&
				(createdAt.Before(intervalEnd) || createdAt.Equal(intervalEnd)) {
				incrementalUserCount++
			}
		}

		// Add this period's data to result
		result = append(result, map[string]interface{}{
			"range":       i + 1,
			"count-users": incrementalUserCount,
			"date":        interval.FromDate,
			"end_date":    interval.ToDate,
		})
	}

	return result, nil
}

// GetTransactionAnalytics returns transaction analytics with optional filters
func (s *analyticsSvc) GetTransactionAnalytics(filters map[string]interface{}) (*models.TransactionAnalytics, error) {
	transactionColl := database.Mdb.Database(s.db).Collection("loyaltyTransactions")

	// Build match filter based on provided filters
	matchFilter := bson.M{"trash": bson.M{"$ne": true}}
	if filters != nil {
		if startDate, ok := filters["start_date"].(time.Time); ok {
			matchFilter["createdAt"] = bson.M{"$gte": startDate}
		}
		if endDate, ok := filters["end_date"].(time.Time); ok {
			if existingFilter, exists := matchFilter["createdAt"]; exists {
				matchFilter["createdAt"] = bson.M{"$gte": existingFilter.(bson.M)["$gte"], "$lte": endDate}
			} else {
				matchFilter["createdAt"] = bson.M{"$lte": endDate}
			}
		}
		if serviceID, ok := filters["service_id"].(string); ok {
			matchFilter["service_id"] = serviceID
		}
		if txnType, ok := filters["type"].(string); ok {
			matchFilter["type"] = txnType
		}
		if userID, ok := filters["user_id"].(string); ok {
			matchFilter["user_id"] = userID
		}
		if minAmount, ok := filters["min_amount"].(float64); ok {
			matchFilter["amount"] = bson.M{"$gte": minAmount}
		}
		if maxAmount, ok := filters["max_amount"].(float64); ok {
			if existingFilter, exists := matchFilter["amount"]; exists {
				matchFilter["amount"] = bson.M{"$gte": existingFilter.(bson.M)["$gte"], "$lte": maxAmount}
			} else {
				matchFilter["amount"] = bson.M{"$lte": maxAmount}
			}
		}
	}

	// Get total redemptions
	redemptionFilter := bson.M{
		"type":  bson.M{"$in": []constants.TxnType{constants.TXN_VOUCHER_USED}},
		"trash": bson.M{"$ne": true},
	}
	totalRedemptions, _ := transactionColl.CountDocuments(s.ctx, redemptionFilter)

	// Get transactions by service
	transactionsByService, _ := s.getTransactionsByService(transactionColl)

	// Get points per day
	pointsPerDay, _ := s.getPointsTransferredPerDay(transactionColl)

	// Get pending points stats
	pendingPointsCount, totalPendingPoints, _ := s.getPendingPointsStats()

	return &models.TransactionAnalytics{
		TotalRedemptions:      totalRedemptions,
		TransactionsByService: transactionsByService,
		PointsPerDay:          pointsPerDay,
		PendingPointsCount:    pendingPointsCount,
		TotalPendingPoints:    totalPendingPoints,
	}, nil
}

// GetSurveyAnalytics returns survey analytics with optional filters
func (s *analyticsSvc) GetSurveyAnalytics(filters map[string]interface{}) (*models.SurveyAnalytics, error) {
	// Build match filter based on provided filters
	matchFilter := bson.M{"trash": bson.M{"$ne": true}}
	if filters != nil {
		if startDate, ok := filters["start_date"].(time.Time); ok {
			matchFilter["createdAt"] = bson.M{"$gte": startDate}
		}
		if endDate, ok := filters["end_date"].(time.Time); ok {
			if existingFilter, exists := matchFilter["createdAt"]; exists {
				matchFilter["createdAt"] = bson.M{"$gte": existingFilter.(bson.M)["$gte"], "$lte": endDate}
			} else {
				matchFilter["createdAt"] = bson.M{"$lte": endDate}
			}
		}
		if surveyID, ok := filters["survey_id"].(string); ok {
			matchFilter["survey_id"] = surveyID
		}
		if category, ok := filters["category"].(string); ok {
			matchFilter["category"] = category
		}
		if minRating, ok := filters["min_rating"].(float64); ok {
			matchFilter["rating"] = bson.M{"$gte": minRating}
		}
		if maxRating, ok := filters["max_rating"].(float64); ok {
			if existingFilter, exists := matchFilter["rating"]; exists {
				matchFilter["rating"] = bson.M{"$gte": existingFilter.(bson.M)["$gte"], "$lte": maxRating}
			} else {
				matchFilter["rating"] = bson.M{"$lte": maxRating}
			}
		}
	}

	// Get surveys by category with filters
	surveysByCategory, err := s.getSurveysByCategory(matchFilter)
	if err != nil {
		surveysByCategory = make(map[string]models.SurveyCategoryStats)
	}

	return &models.SurveyAnalytics{
		SurveysByCategory: surveysByCategory,
	}, nil
}

// GetTransactionsByService returns transactions grouped by service with filters
func (s *analyticsSvc) GetTransactionsByService(filters map[string]interface{}) ([]models.TransactionServiceStats, error) {
	// Build match filter
	matchFilter := bson.M{"trash": bson.M{"$ne": true}}
	if filters != nil {
		if startDate, ok := filters["start_date"].(time.Time); ok {
			matchFilter["createdAt"] = bson.M{"$gte": startDate}
		}
		if endDate, ok := filters["end_date"].(time.Time); ok {
			if existingFilter, exists := matchFilter["createdAt"]; exists {
				matchFilter["createdAt"] = bson.M{"$gte": existingFilter.(bson.M)["$gte"], "$lte": endDate}
			} else {
				matchFilter["createdAt"] = bson.M{"$lte": endDate}
			}
		}
		if serviceID, ok := filters["service_id"].(string); ok {
			matchFilter["service_id"] = serviceID
		}
	}

	transactionColl := database.Mdb.Database(s.db).Collection("loyaltyTransactions")
	transactionsByServiceMap, err := s.getTransactionsByService(transactionColl)
	if err != nil {
		return nil, err
	}

	// Convert map to slice
	var result []models.TransactionServiceStats
	for _, stats := range transactionsByServiceMap {
		result = append(result, stats)
	}

	return result, nil
}

// GetPointsPerDay returns points transferred per day with filters
func (s *analyticsSvc) GetPointsPerDay(filters map[string]interface{}) ([]models.DailyPointsStats, error) {
	transactionColl := database.Mdb.Database(s.db).Collection("loyaltyTransactions")
	pointsPerDayMap, err := s.getPointsTransferredPerDay(transactionColl)
	if err != nil {
		return nil, err
	}

	// Convert map to slice
	var result []models.DailyPointsStats
	for date, points := range pointsPerDayMap {
		result = append(result, models.DailyPointsStats{
			Date:        date,
			TotalPoints: points,
			Count:       1, // This would need to be calculated separately if needed
		})
	}

	return result, nil
}

// Helper methods for filtering
func (s *analyticsSvc) getUsersByMonth(filters map[string]interface{}) (map[string]int64, error) {
	// Build URL for users service
	usersServiceURL := os.Getenv("USERS_URL")
	if usersServiceURL == "" {
		usersServiceURL = "http://localhost:3201"
	}

	url := fmt.Sprintf("%s/users/analytics/by-month", usersServiceURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return make(map[string]int64), err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-client", "loyalty-service")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return make(map[string]int64), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return make(map[string]int64), nil // Return empty map on error
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return make(map[string]int64), err
	}

	var response struct {
		Success bool             `json:"success"`
		Data    map[string]int64 `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return make(map[string]int64), err
	}

	if !response.Success {
		return make(map[string]int64), nil
	}

	return response.Data, nil
}

// getUsersByPeriod returns users grouped by the specified period from users service
func (s *analyticsSvc) getUsersByPeriod(client string, filters map[string]interface{}, period string) (map[string]int64, error) {
	// Build URL for users service
	usersServiceURL := os.Getenv("USERS_URL")
	if usersServiceURL == "" {
		usersServiceURL = "http://localhost:3201"
	}

	// Build query parameters for filtering
	params := url.Values{}
	if filters != nil {
		if startDate, ok := filters["start_date"].(time.Time); ok {
			params.Add("start_date", startDate.Format("2006-01-02"))
		}
		if endDate, ok := filters["end_date"].(time.Time); ok {
			params.Add("end_date", endDate.Format("2006-01-02"))
		}
		if year, ok := filters["year"].(int); ok {
			params.Add("year", fmt.Sprintf("%d", year))
		}
		if month, ok := filters["month"].(int); ok {
			params.Add("month", fmt.Sprintf("%d", month))
		}
		if days, ok := filters["days"].(int); ok {
			params.Add("days", fmt.Sprintf("%d", days))
		}
	}

	// Make request to users service to get all users
	url := fmt.Sprintf("%s/users?%s", usersServiceURL, params.Encode())
	fmt.Printf("DEBUG: Calling users service for period grouping: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return make(map[string]int64), err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-client", client)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return make(map[string]int64), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("DEBUG: Users service error: %d - %s\n", resp.StatusCode, string(body))

		// If it's a subscription error, fallback to local wallet data
		if resp.StatusCode == 500 && strings.Contains(string(body), "subscription") {
			fmt.Printf("DEBUG: Users service subscription issue in getUsersByPeriod, falling back to local wallet data\n")
			return s.getUsersByPeriodFromWallets(client, filters, period)
		}

		return make(map[string]int64), nil // Return empty map instead of error
	}

	// Parse users response
	var usersResponse struct {
		Users []struct {
			ID        string    `json:"_id"`
			CreatedAt time.Time `json:"createdAt"`
		} `json:"users"`
		Pagination struct {
			TotalPages int   `json:"totalPages"`
			PerPage    int   `json:"perPage"`
			TotalCount int64 `json:"totalCount"`
		} `json:"pagination"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&usersResponse); err != nil {
		fmt.Printf("DEBUG: Error parsing users response: %v\n", err)
		return make(map[string]int64), nil
	}

	// Group users by period based on createdAt
	result := make(map[string]int64)
	for _, user := range usersResponse.Users {
		var key string
		switch period {
		case "year":
			key = user.CreatedAt.Format("2006")
		case "day":
			key = user.CreatedAt.Format("2006-01-02")
		default: // month
			key = user.CreatedAt.Format("2006-01")
		}
		result[key]++
	}

	fmt.Printf("DEBUG: Users by period (%s): %+v\n", period, result)
	return result, nil
}

func (s *analyticsSvc) getNewUsersByDay(client string, filters map[string]interface{}) (map[string]int64, error) {
	// Use the same approach as getUsersByPeriod but force period to "day"
	return s.getUsersByPeriod(client, filters, "day")
}

// getSurveysByCategory gets surveys grouped by title with optional filters
func (s *analyticsSvc) getSurveysByCategory(matchFilter bson.M) (map[string]models.SurveyCategoryStats, error) {
	result := make(map[string]models.SurveyCategoryStats)

	// Access the loyaltySurveys collection
	surveyColl := database.Mdb.Database(s.db).Collection("loyaltySurveys")

	// Add case-insensitive filter for specific titles if needed
	// You can filter by specific titles like "facility", "dentist", "customer service team"
	titleFilter := bson.M{
		"$or": []bson.M{
			{"title.en": bson.M{"$regex": "facility", "$options": "i"}},
			{"title.en": bson.M{"$regex": "dentist", "$options": "i"}},
			{"title.en": bson.M{"$regex": "customer service team", "$options": "i"}},
			{"title.ar": bson.M{"$regex": "facility", "$options": "i"}},
			{"title.ar": bson.M{"$regex": "dentist", "$options": "i"}},
			{"title.ar": bson.M{"$regex": "customer service team", "$options": "i"}},
		},
	}

	// Combine existing match filter with title filter
	combinedFilter := bson.M{
		"$and": []bson.M{matchFilter, titleFilter},
	}

	// Build aggregation pipeline
	pipeline := []bson.M{
		{"$match": combinedFilter},
		{"$group": bson.M{
			"_id":   "$title.en", // Group by English title
			"count": bson.M{"$sum": 1},
			// "average_rating": bson.M{"$avg": 4.0}, // Default rating since surveys don't have ratings
		}},
		{"$sort": bson.M{"count": -1}},
	}

	cursor, err := surveyColl.Aggregate(s.ctx, pipeline)
	if err != nil {
		return result, err
	}
	defer cursor.Close(s.ctx)

	// Process aggregation results
	for cursor.Next(s.ctx) {
		var doc struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
			// AverageRating float64 `bson:"average_rating"`
		}

		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		// Handle case where title might be null/empty
		title := doc.ID
		if title == "" {
			title = "untitled"
		}

		result[title] = models.SurveyCategoryStats{
			Count: doc.Count,
			// AverageRating: doc.AverageRating,
		}
	}

	if err := cursor.Err(); err != nil {
		return result, err
	}

	return result, nil
}

// GetEnhancedTransactionAnalytics returns the enhanced transaction analytics with optional filters
func (s *analyticsSvc) GetEnhancedTransactionAnalytics(filters map[string]interface{}) (*models.EnhancedTransactionAnalytics, error) {
	// Debug: Log the received filters
	// Get transactions by service and category
	transactionsByServiceCategory, err := s.getTransactionsByServiceCategory(filters)
	if err != nil {
		transactionsByServiceCategory = []models.TransactionServiceCategoryStats{}
	}
	if transactionsByServiceCategory == nil {
		transactionsByServiceCategory = []models.TransactionServiceCategoryStats{}
	}

	// Get points transferred per day
	pointsTransferredPerDay, err := s.getPointsTransferredPerDayDetailed(filters)
	if err != nil {
		pointsTransferredPerDay = []models.DailyPointsTransferStats{}
	}
	if pointsTransferredPerDay == nil {
		pointsTransferredPerDay = []models.DailyPointsTransferStats{}
	}

	// Get pending points transactions per day
	pendingPointsTransactionsPerDay, err := s.getPendingPointsTransactionsPerDay(filters)
	if err != nil {
		pendingPointsTransactionsPerDay = []models.DailyPendingPointsStats{}
	}
	if pendingPointsTransactionsPerDay == nil {
		pendingPointsTransactionsPerDay = []models.DailyPendingPointsStats{}
	}

	return &models.EnhancedTransactionAnalytics{
		TransactionsByServiceCategory:   transactionsByServiceCategory,
		PointsTransferredPerDay:         pointsTransferredPerDay,
		PendingPointsTransactionsPerDay: pendingPointsTransactionsPerDay,
	}, nil
}

// getTransactionsByServiceCategory gets transactions grouped by service and category (type_name)
func (s *analyticsSvc) getTransactionsByServiceCategory(filters map[string]interface{}) ([]models.TransactionServiceCategoryStats, error) {
	transactionColl := database.Mdb.Database(s.db).Collection("loyaltyTransactions")

	// Build match filter
	matchFilter := bson.M{"trash": bson.M{"$ne": true}}
	s.applyFiltersToMatch(matchFilter, filters)

	// Pipeline with lookup for voucher and service details (similar to transaction service)
	pipeline := []bson.M{
		{"$match": matchFilter},
		// Lookup voucher details
		{
			"$lookup": bson.M{
				"from":         "loyaltyVouchers",
				"localField":   "ref_id",
				"foreignField": "_id",
				"as":           "voucher_info",
			},
		},
		// Lookup service details from voucher
		{
			"$lookup": bson.M{
				"from": "loyaltyServices",
				"let":  bson.M{"serviceId": bson.M{"$arrayElemAt": []interface{}{"$voucher_info.service_id", 0}}},
				"pipeline": []bson.M{
					{
						"$match": bson.M{
							"$expr": bson.M{
								"$eq": []interface{}{"$_id", "$$serviceId"},
							},
						},
					},
				},
				"as": "service_info",
			},
		},
		// Update type_name field for voucher transactions
		{
			"$addFields": bson.M{
				"dynamic_type_name": bson.M{
					"$cond": bson.M{
						"if": bson.M{
							"$and": []bson.M{
								{"$in": []interface{}{"$type", []int{int(constants.TXN_VOUCHER_USED), int(constants.TXN_VOUCHER_EXPIRED)}}},
								{"$gt": []interface{}{bson.M{"$size": "$voucher_info"}, 0}},
								{"$gt": []interface{}{bson.M{"$size": "$service_info"}, 0}},
							},
						},
						"then": bson.M{"$arrayElemAt": []interface{}{"$service_info.name.en", 0}},
						"else": "$type_name",
					},
				},
				"service_name": bson.M{
					"$cond": bson.M{
						"if": bson.M{
							"$and": []bson.M{
								{"$gt": []interface{}{bson.M{"$size": "$service_info"}, 0}},
							},
						},
						"then": bson.M{"$arrayElemAt": []interface{}{"$service_info.name.en", 0}},
						"else": "Other",
					},
				},
			},
		},
		// Group by service name and type name
		{
			"$group": bson.M{
				"_id": bson.M{
					"service_name": "$service_name",
					"type_name":    "$dynamic_type_name",
				},
				"count":        bson.M{"$sum": 1},
				"total_amount": bson.M{"$sum": "$amount"},
			},
		},
		{"$sort": bson.M{"count": -1}},
	}

	cursor, err := transactionColl.Aggregate(s.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	var result []models.TransactionServiceCategoryStats
	for cursor.Next(s.ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		// Safe extraction of _id fields
		serviceName := "Unknown"
		typeName := "Unknown"

		if idVal := doc["_id"]; idVal != nil {
			if idDoc, ok := idVal.(bson.M); ok {
				if sName, ok := idDoc["service_name"].(string); ok && sName != "" {
					serviceName = sName
				}
				if tName, ok := idDoc["type_name"].(string); ok && tName != "" {
					typeName = tName
				}
			}
		}

		count := int64(0)
		if countVal := doc["count"]; countVal != nil {
			if c, ok := countVal.(int32); ok {
				count = int64(c)
			} else if c, ok := countVal.(int64); ok {
				count = c
			}
		}

		totalAmount := float64(0)
		if totalAmountVal := doc["total_amount"]; totalAmountVal != nil {
			if amt, ok := totalAmountVal.(float64); ok {
				totalAmount = amt
			} else if amt, ok := totalAmountVal.(int32); ok {
				totalAmount = float64(amt)
			} else if amt, ok := totalAmountVal.(int64); ok {
				totalAmount = float64(amt)
			}
		}

		result = append(result, models.TransactionServiceCategoryStats{
			ServiceName: serviceName,
			TypeName:    typeName,
			Count:       count,
			TotalAmount: totalAmount,
		})
	}

	// Ensure we always return a non-nil slice
	if result == nil {
		result = []models.TransactionServiceCategoryStats{}
	}

	return result, nil
}

// getPointsTransferredPerDayDetailed gets detailed daily points transfer statistics
func (s *analyticsSvc) getPointsTransferredPerDayDetailed(filters map[string]interface{}) ([]models.DailyPointsTransferStats, error) {
	transactionColl := database.Mdb.Database(s.db).Collection("loyaltyTransactions")

	// Build match filter for point transfer transactions
	matchFilter := bson.M{
		"trash": bson.M{"$ne": true},
		"type": bson.M{"$in": []constants.TxnType{
			constants.TXN_POINTS_ADDED_TO_WALLET,
			constants.TXN_VOUCHER_USED,
		}},
	}
	s.applyFiltersToMatch(matchFilter, filters)

	pipeline := []bson.M{
		{"$match": matchFilter},
		{
			"$group": bson.M{
				"_id": bson.M{
					"year":  bson.M{"$year": "$createdAt"},
					"month": bson.M{"$month": "$createdAt"},
					"day":   bson.M{"$dayOfMonth": "$createdAt"},
				},
				"total_points": bson.M{"$sum": "$amount"},
				"count":        bson.M{"$sum": 1},
			},
		},
		{"$sort": bson.M{"_id.year": 1, "_id.month": 1, "_id.day": 1}},
	}

	cursor, err := transactionColl.Aggregate(s.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	var result []models.DailyPointsTransferStats
	for cursor.Next(s.ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		// Safe extraction of date fields
		year := int32(2024) // Default values
		month := int32(1)
		day := int32(1)

		if idVal := doc["_id"]; idVal != nil {
			if idDoc, ok := idVal.(bson.M); ok {
				if y, ok := idDoc["year"].(int32); ok {
					year = y
				}
				if m, ok := idDoc["month"].(int32); ok {
					month = m
				}
				if d, ok := idDoc["day"].(int32); ok {
					day = d
				}
			}
		}

		date := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC).Format("2006-01-02")

		totalPoints := float64(0)
		if totalPointsVal := doc["total_points"]; totalPointsVal != nil {
			if points, ok := totalPointsVal.(float64); ok {
				totalPoints = points
			} else if points, ok := totalPointsVal.(int32); ok {
				totalPoints = float64(points)
			} else if points, ok := totalPointsVal.(int64); ok {
				totalPoints = float64(points)
			}
		}

		count := int64(0)
		if countVal := doc["count"]; countVal != nil {
			if c, ok := countVal.(int32); ok {
				count = int64(c)
			} else if c, ok := countVal.(int64); ok {
				count = c
			}
		}

		result = append(result, models.DailyPointsTransferStats{
			Date:        date,
			TotalPoints: totalPoints,
			Count:       count,
		})
	}

	// Ensure we always return a non-nil slice
	if result == nil {
		result = []models.DailyPointsTransferStats{}
	}

	return result, nil
}

// getPendingPointsTransactionsPerDay gets daily pending points transaction counts
func (s *analyticsSvc) getPendingPointsTransactionsPerDay(filters map[string]interface{}) ([]models.DailyPendingPointsStats, error) {
	transactionColl := database.Mdb.Database(s.db).Collection("loyaltyTransactions")

	// Build match filter for pending transactions
	matchFilter := bson.M{
		"trash": bson.M{"$ne": true},
		"type": bson.M{"$in": []constants.TxnType{
			constants.TXN_POINTS_ADDED_TO_WALLET,
		}},
		"amount": bson.M{"$gt": 0}, // Only positive amounts for pending points
	}
	s.applyFiltersToMatch(matchFilter, filters)

	pipeline := []bson.M{
		{"$match": matchFilter},
		{
			"$group": bson.M{
				"_id": bson.M{
					"year":  bson.M{"$year": "$createdAt"},
					"month": bson.M{"$month": "$createdAt"},
					"day":   bson.M{"$dayOfMonth": "$createdAt"},
				},
				"count": bson.M{"$sum": 1},
			},
		},
		{"$sort": bson.M{"_id.year": 1, "_id.month": 1, "_id.day": 1}},
	}

	cursor, err := transactionColl.Aggregate(s.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	var result []models.DailyPendingPointsStats
	for cursor.Next(s.ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		// Safe extraction of date fields
		year := int32(2024) // Default values
		month := int32(1)
		day := int32(1)

		if idVal := doc["_id"]; idVal != nil {
			if idDoc, ok := idVal.(bson.M); ok {
				if y, ok := idDoc["year"].(int32); ok {
					year = y
				}
				if m, ok := idDoc["month"].(int32); ok {
					month = m
				}
				if d, ok := idDoc["day"].(int32); ok {
					day = d
				}
			}
		}

		date := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC).Format("2006-01-02")

		count := int64(0)
		if countVal := doc["count"]; countVal != nil {
			if c, ok := countVal.(int32); ok {
				count = int64(c)
			} else if c, ok := countVal.(int64); ok {
				count = c
			}
		}

		result = append(result, models.DailyPendingPointsStats{
			Date:  date,
			Count: count,
		})
	}

	// Ensure we always return a non-nil slice
	if result == nil {
		result = []models.DailyPendingPointsStats{}
	}

	return result, nil
}

// Helper function to apply filters to match stage
func (s *analyticsSvc) applyFiltersToMatch(matchFilter bson.M, filters map[string]interface{}) {
	if filters == nil {
		return
	}

	if startDate, ok := filters["start_date"].(time.Time); ok {
		matchFilter["createdAt"] = bson.M{"$gte": startDate}
	}
	if endDate, ok := filters["end_date"].(time.Time); ok {
		if existingFilter, exists := matchFilter["createdAt"]; exists {
			matchFilter["createdAt"] = bson.M{"$gte": existingFilter.(bson.M)["$gte"], "$lt": endDate}
		} else {
			matchFilter["createdAt"] = bson.M{"$lt": endDate}
		}
	}
	if serviceID, ok := filters["service_id"].(string); ok {
		matchFilter["service_id"] = serviceID
	}
	if txnType, ok := filters["type"].(string); ok {
		matchFilter["type"] = txnType
	}
	if userID, ok := filters["user_id"].(string); ok {
		matchFilter["user_id"] = userID
	}
	if minAmount, ok := filters["min_amount"].(float64); ok {
		matchFilter["amount"] = bson.M{"$gte": minAmount}
	}
	if maxAmount, ok := filters["max_amount"].(float64); ok {
		if existingFilter, exists := matchFilter["amount"]; exists {
			matchFilter["amount"] = bson.M{"$gte": existingFilter.(bson.M)["$gte"], "$lte": maxAmount}
		} else {
			matchFilter["amount"] = bson.M{"$lte": maxAmount}
		}
	}
}

// Helper function to divide date range into 10 equal intervals (simple approach)
func (s *analyticsSvc) generateTimeIntervals(fromDate, toDate time.Time) []models.TimeSeriesPeriod {
	totalDays := int(toDate.Sub(fromDate).Hours() / 24)
	daysPerPeriod := totalDays / 10
	remainingDays := totalDays % 10

	var periods []models.TimeSeriesPeriod
	currentDate := fromDate

	for i := 0; i < 10; i++ {
		start := currentDate

		// Calculate days for this period (distribute remaining days to first few periods)
		periodDays := daysPerPeriod
		if i < remainingDays {
			periodDays++
		}

		var end time.Time
		if i == 9 {
			// Last period always ends on the exact end date
			end = toDate
		} else {
			// End date is start + period days
			end = start.AddDate(0, 0, periodDays)
		}

		periods = append(periods, models.TimeSeriesPeriod{
			FromDate:    start.Format("2006-01-02"),
			ToDate:      end.Format("2006-01-02"),
			PeriodIndex: i + 1,
			Label:       fmt.Sprintf("Period %d (%s to %s)", i+1, start.Format("Jan 02"), end.Format("Jan 02")),
		})

		// Next period starts the day after this period ends
		currentDate = end.AddDate(0, 0, 1)
	}

	return periods
}

// Helper function to generate time periods based on actual data existence
func (s *analyticsSvc) generateDataDrivenTimeIntervals(xclient, accessToken string, fromDate, toDate time.Time) []models.TimeSeriesPeriod {
	// First, try to find periods with data by sampling different time ranges
	var periods []models.TimeSeriesPeriod

	totalDuration := toDate.Sub(fromDate)

	// Determine appropriate interval size based on date range
	var intervalDuration time.Duration
	if totalDuration <= 30*24*time.Hour { // Less than 30 days - use daily periods
		intervalDuration = 24 * time.Hour
	} else if totalDuration <= 365*24*time.Hour { // Less than a year - use weekly periods
		intervalDuration = 7 * 24 * time.Hour
	} else { // More than a year - use monthly periods
		intervalDuration = 30 * 24 * time.Hour
	}

	// Generate periods and check if they have data
	periodIndex := 1
	for current := fromDate; current.Before(toDate) && len(periods) < 10; current = current.Add(intervalDuration) {
		start := current
		end := current.Add(intervalDuration)
		if end.After(toDate) {
			end = toDate
		}

		// Quick check if this period has any data
		query := s.buildQueryWithDateRange(start, end)
		hasData := false

		// Check each service for data in this period
		if bookingResponse, err := s.GetTotalBookings(xclient, accessToken, query); err == nil {
			if s.extractCountFromResponse(bookingResponse, "total_bookings") > 0 {
				hasData = true
			}
		}

		if !hasData {
			if consultationResponse, err := s.GetTotalOnlineConsultations(xclient, accessToken, query); err == nil {
				if s.extractCountFromResponse(consultationResponse, "total_online_consultations") > 0 {
					hasData = true
				}
			}
		}

		if !hasData {
			if testDriveResponse, err := s.GetTotalSmileTestDrives(xclient, accessToken, query); err == nil {
				if s.extractCountFromResponse(testDriveResponse, "total_smile_test_drives") > 0 {
					hasData = true
				}
			}
		}

		// Only add periods that have data
		if hasData {
			var label string
			if intervalDuration == 24*time.Hour {
				label = fmt.Sprintf("Day %s", start.Format("Jan 02"))
			} else if intervalDuration == 7*24*time.Hour {
				label = fmt.Sprintf("Week of %s", start.Format("Jan 02"))
			} else {
				label = fmt.Sprintf("Month of %s", start.Format("Jan 2006"))
			}

			periods = append(periods, models.TimeSeriesPeriod{
				FromDate:    start.Format("2006-01-02"),
				ToDate:      end.Format("2006-01-02"),
				PeriodIndex: periodIndex,
				Label:       label,
			})
			periodIndex++

			fmt.Printf("DEBUG: Found data in period %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
		}
	}

	// If no periods have data, return a few recent periods anyway
	if len(periods) == 0 {
		fmt.Printf("DEBUG: No data found, returning recent periods\n")
		// Return last 3 periods as fallback
		for i := 0; i < 3; i++ {
			start := toDate.AddDate(0, 0, -(i+1)*int(intervalDuration.Hours()/24))
			end := toDate.AddDate(0, 0, -i*int(intervalDuration.Hours()/24))

			periods = append([]models.TimeSeriesPeriod{{
				FromDate:    start.Format("2006-01-02"),
				ToDate:      end.Format("2006-01-02"),
				PeriodIndex: i + 1,
				Label:       fmt.Sprintf("Recent Period %d", i+1),
			}}, periods...)
		}
	}

	return periods
}

// Helper function to generate time periods based on booking data existence
func (s *analyticsSvc) generateDataDrivenBookingIntervals(xclient, accessToken string, fromDate, toDate time.Time) []models.TimeSeriesPeriod {
	var periods []models.TimeSeriesPeriod

	totalDuration := toDate.Sub(fromDate)

	// Determine appropriate interval size
	var intervalDuration time.Duration
	if totalDuration <= 30*24*time.Hour {
		intervalDuration = 24 * time.Hour
	} else if totalDuration <= 365*24*time.Hour {
		intervalDuration = 7 * 24 * time.Hour
	} else {
		intervalDuration = 30 * 24 * time.Hour
	}

	periodIndex := 1
	for current := fromDate; current.Before(toDate) && len(periods) < 10; current = current.Add(intervalDuration) {
		start := current
		end := current.Add(intervalDuration)
		if end.After(toDate) {
			end = toDate
		}

		// Check if this period has booking data
		query := s.buildQueryWithDateRange(start, end)
		hasData := false

		if bookingResponse, err := s.GetTotalBookings(xclient, accessToken, query); err == nil {
			if s.extractCountFromResponse(bookingResponse, "total_bookings") > 0 {
				hasData = true
			}
		}

		if hasData {
			var label string
			if intervalDuration == 24*time.Hour {
				label = fmt.Sprintf("Day %s", start.Format("Jan 02"))
			} else if intervalDuration == 7*24*time.Hour {
				label = fmt.Sprintf("Week of %s", start.Format("Jan 02"))
			} else {
				label = fmt.Sprintf("Month of %s", start.Format("Jan 2006"))
			}

			periods = append(periods, models.TimeSeriesPeriod{
				FromDate:    start.Format("2006-01-02"),
				ToDate:      end.Format("2006-01-02"),
				PeriodIndex: periodIndex,
				Label:       label,
			})
			periodIndex++
		}
	}

	return periods
}

// Helper function to generate time periods based on consultation data existence
func (s *analyticsSvc) generateDataDrivenConsultationIntervals(xclient, accessToken string, fromDate, toDate time.Time) []models.TimeSeriesPeriod {
	var periods []models.TimeSeriesPeriod

	totalDuration := toDate.Sub(fromDate)

	var intervalDuration time.Duration
	if totalDuration <= 30*24*time.Hour {
		intervalDuration = 24 * time.Hour
	} else if totalDuration <= 365*24*time.Hour {
		intervalDuration = 7 * 24 * time.Hour
	} else {
		intervalDuration = 30 * 24 * time.Hour
	}

	periodIndex := 1
	for current := fromDate; current.Before(toDate) && len(periods) < 10; current = current.Add(intervalDuration) {
		start := current
		end := current.Add(intervalDuration)
		if end.After(toDate) {
			end = toDate
		}

		query := s.buildQueryWithDateRange(start, end)
		hasData := false

		if consultationResponse, err := s.GetTotalOnlineConsultations(xclient, accessToken, query); err == nil {
			if s.extractCountFromResponse(consultationResponse, "total_online_consultations") > 0 {
				hasData = true
			}
		}

		if hasData {
			var label string
			if intervalDuration == 24*time.Hour {
				label = fmt.Sprintf("Day %s", start.Format("Jan 02"))
			} else if intervalDuration == 7*24*time.Hour {
				label = fmt.Sprintf("Week of %s", start.Format("Jan 02"))
			} else {
				label = fmt.Sprintf("Month of %s", start.Format("Jan 2006"))
			}

			periods = append(periods, models.TimeSeriesPeriod{
				FromDate:    start.Format("2006-01-02"),
				ToDate:      end.Format("2006-01-02"),
				PeriodIndex: periodIndex,
				Label:       label,
			})
			periodIndex++
		}
	}

	return periods
}

// Helper function to generate time periods based on smile test drive data existence
func (s *analyticsSvc) generateDataDrivenSmileTestDriveIntervals(xclient, accessToken string, fromDate, toDate time.Time) []models.TimeSeriesPeriod {
	var periods []models.TimeSeriesPeriod

	totalDuration := toDate.Sub(fromDate)

	var intervalDuration time.Duration
	if totalDuration <= 30*24*time.Hour {
		intervalDuration = 24 * time.Hour
	} else if totalDuration <= 365*24*time.Hour {
		intervalDuration = 7 * 24 * time.Hour
	} else {
		intervalDuration = 30 * 24 * time.Hour
	}

	periodIndex := 1
	for current := fromDate; current.Before(toDate) && len(periods) < 10; current = current.Add(intervalDuration) {
		start := current
		end := current.Add(intervalDuration)
		if end.After(toDate) {
			end = toDate
		}

		query := s.buildQueryWithDateRange(start, end)
		hasData := false

		if testDriveResponse, err := s.GetTotalSmileTestDrives(xclient, accessToken, query); err == nil {
			if s.extractCountFromResponse(testDriveResponse, "total_smile_test_drives") > 0 {
				hasData = true
			}
		}

		if hasData {
			var label string
			if intervalDuration == 24*time.Hour {
				label = fmt.Sprintf("Day %s", start.Format("Jan 02"))
			} else if intervalDuration == 7*24*time.Hour {
				label = fmt.Sprintf("Week of %s", start.Format("Jan 02"))
			} else {
				label = fmt.Sprintf("Month of %s", start.Format("Jan 2006"))
			}

			periods = append(periods, models.TimeSeriesPeriod{
				FromDate:    start.Format("2006-01-02"),
				ToDate:      end.Format("2006-01-02"),
				PeriodIndex: periodIndex,
				Label:       label,
			})
			periodIndex++
		}
	}

	return periods
}

// Helper function to generate time periods based on internal transaction data existence
func (s *analyticsSvc) generateDataDrivenTransactionIntervals(fromDate, toDate time.Time) []models.TimeSeriesPeriod {
	var periods []models.TimeSeriesPeriod

	totalDuration := toDate.Sub(fromDate)

	// Determine appropriate interval size
	var intervalDuration time.Duration
	if totalDuration <= 30*24*time.Hour {
		intervalDuration = 24 * time.Hour
	} else if totalDuration <= 365*24*time.Hour {
		intervalDuration = 7 * 24 * time.Hour
	} else {
		intervalDuration = 30 * 24 * time.Hour
	}

	transactionColl := database.Mdb.Database(s.db).Collection("loyaltyTransactions")

	periodIndex := 1
	for current := fromDate; current.Before(toDate) && len(periods) < 10; current = current.Add(intervalDuration) {
		start := current
		end := current.Add(intervalDuration)
		if end.After(toDate) {
			end = toDate
		}

		// Check if this period has transaction data
		endInclusive := end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		matchFilter := bson.M{
			"trash": bson.M{"$ne": true},
			"createdAt": bson.M{
				"$gte": start,
				"$lte": endInclusive,
			},
		}

		count, err := transactionColl.CountDocuments(s.ctx, matchFilter)
		if err == nil && count > 0 {
			var label string
			if intervalDuration == 24*time.Hour {
				label = fmt.Sprintf("Day %s", start.Format("Jan 02"))
			} else if intervalDuration == 7*24*time.Hour {
				label = fmt.Sprintf("Week of %s", start.Format("Jan 02"))
			} else {
				label = fmt.Sprintf("Month of %s", start.Format("Jan 2006"))
			}

			periods = append(periods, models.TimeSeriesPeriod{
				FromDate:    start.Format("2006-01-02"),
				ToDate:      end.Format("2006-01-02"),
				PeriodIndex: periodIndex,
				Label:       label,
			})
			periodIndex++

			fmt.Printf("DEBUG: Found transaction data in period %s to %s (%d transactions)\n",
				start.Format("2006-01-02"), end.Format("2006-01-02"), count)
		}
	}

	return periods
}

// Helper function to generate time periods based on user data existence from users service
func (s *analyticsSvc) generateDataDrivenUserIntervals(client string, fromDate, toDate time.Time) []models.TimeSeriesPeriod {
	var periods []models.TimeSeriesPeriod

	// First, try to get overall user count to see if users service is working
	usersServiceURL := os.Getenv("USERS_URL")
	if usersServiceURL == "" {
		usersServiceURL = "http://localhost:3201"
	}

	// Test if users service is accessible
	testURL := fmt.Sprintf("%s/users", usersServiceURL)
	testReq, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		fmt.Printf("DEBUG: Error creating test request: %v\n", err)
		return s.generateFallbackPeriods(fromDate, toDate)
	}

	testReq.Header.Set("Content-Type", "application/json")
	testReq.Header.Set("x-client", client)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	testResp, err := httpClient.Do(testReq)
	if err != nil {
		fmt.Printf("DEBUG: Users service not accessible: %v\n", err)
		return s.generateFallbackPeriods(fromDate, toDate)
	}
	defer testResp.Body.Close()

	if testResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(testResp.Body)
		fmt.Printf("DEBUG: Users service returned status %d: %s\n", testResp.StatusCode, string(body))

		// If it's a subscription error, use local wallet data instead
		if testResp.StatusCode == 500 && strings.Contains(string(body), "subscription") {
			fmt.Printf("DEBUG: Users service subscription issue, falling back to local wallet data\n")
			return s.generateDataDrivenUserIntervalsFromWallets(fromDate, toDate)
		}

		return s.generateFallbackPeriods(fromDate, toDate)
	}

	// Check if we have any users at all
	var testResponse struct {
		Pagination struct {
			TotalCount int64 `json:"totalCount"`
		} `json:"pagination"`
	}

	if err := json.NewDecoder(testResp.Body).Decode(&testResponse); err != nil {
		fmt.Printf("DEBUG: Error parsing test response: %v\n", err)
		return s.generateFallbackPeriods(fromDate, toDate)
	}

	if testResponse.Pagination.TotalCount == 0 {
		fmt.Printf("DEBUG: No users found in users service\n")
		return []models.TimeSeriesPeriod{} // Return empty if no users exist
	}

	fmt.Printf("DEBUG: Users service working, total users: %d\n", testResponse.Pagination.TotalCount)

	// Now proceed with normal period generation
	totalDuration := toDate.Sub(fromDate)

	var intervalDuration time.Duration
	if totalDuration <= 30*24*time.Hour {
		intervalDuration = 24 * time.Hour
	} else if totalDuration <= 365*24*time.Hour {
		intervalDuration = 7 * 24 * time.Hour
	} else {
		intervalDuration = 30 * 24 * time.Hour
	}

	periodIndex := 1
	for current := fromDate; current.Before(toDate) && len(periods) < 10; current = current.Add(intervalDuration) {
		start := current
		end := current.Add(intervalDuration)
		if end.After(toDate) {
			end = toDate
		}

		// Check if this period has user data by calling users service
		params := url.Values{}
		params.Add("start_date", start.Format("2006-01-02"))
		params.Add("end_date", end.Format("2006-01-02"))

		periodURL := fmt.Sprintf("%s/users?%s", usersServiceURL, params.Encode())
		fmt.Printf("DEBUG: Checking period %s to %s: %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"), periodURL)

		req, err := http.NewRequest("GET", periodURL, nil)
		if err != nil {
			fmt.Printf("DEBUG: Error creating period request: %v\n", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-client", client)

		resp, err := httpClient.Do(req)
		if err != nil {
			fmt.Printf("DEBUG: Error calling users service for period: %v\n", err)
			continue
		}

		var usersResponse struct {
			Pagination struct {
				TotalCount int64 `json:"totalCount"`
			} `json:"pagination"`
		}

		hasData := false
		if resp.StatusCode == http.StatusOK {
			if err := json.NewDecoder(resp.Body).Decode(&usersResponse); err == nil {
				if usersResponse.Pagination.TotalCount > 0 {
					hasData = true
					fmt.Printf("DEBUG: Period %s to %s has %d users\n",
						start.Format("2006-01-02"), end.Format("2006-01-02"), usersResponse.Pagination.TotalCount)
				} else {
					fmt.Printf("DEBUG: Period %s to %s has 0 users\n",
						start.Format("2006-01-02"), end.Format("2006-01-02"))
				}
			} else {
				fmt.Printf("DEBUG: Error parsing period response: %v\n", err)
			}
		} else {
			fmt.Printf("DEBUG: Period request failed with status: %d\n", resp.StatusCode)
		}
		resp.Body.Close()

		// Add periods that have data
		if hasData {
			var label string
			if intervalDuration == 24*time.Hour {
				label = fmt.Sprintf("Day %s", start.Format("Jan 02"))
			} else if intervalDuration == 7*24*time.Hour {
				label = fmt.Sprintf("Week of %s", start.Format("Jan 02"))
			} else {
				label = fmt.Sprintf("Month of %s", start.Format("Jan 2006"))
			}

			periods = append(periods, models.TimeSeriesPeriod{
				FromDate:    start.Format("2006-01-02"),
				ToDate:      end.Format("2006-01-02"),
				PeriodIndex: periodIndex,
				Label:       label,
			})
			periodIndex++
		}
	}

	fmt.Printf("DEBUG: Found %d periods with user data\n", len(periods))

	// If no periods found but we know users exist, return recent periods as fallback
	if len(periods) == 0 && testResponse.Pagination.TotalCount > 0 {
		fmt.Printf("DEBUG: No periods found with data, using fallback\n")
		return s.generateFallbackPeriods(fromDate, toDate)
	}

	return periods
}

// Fallback method to generate periods when users service is not accessible
func (s *analyticsSvc) generateFallbackPeriods(fromDate, toDate time.Time) []models.TimeSeriesPeriod {
	fmt.Printf("DEBUG: Using fallback period generation\n")

	// Generate recent periods around current time where data is more likely to exist
	now := time.Now()

	// If the date range includes recent time, use that; otherwise use the end of the range
	var baseDate time.Time
	if toDate.After(now.AddDate(0, -6, 0)) { // If within last 6 months
		baseDate = now
	} else {
		baseDate = toDate
	}

	var periods []models.TimeSeriesPeriod

	// Generate 3-5 recent monthly periods
	for i := 0; i < 5; i++ {
		start := baseDate.AddDate(0, -(i + 1), 0)
		end := baseDate.AddDate(0, -i, 0)

		// Ensure periods are within the requested range
		if start.Before(fromDate) {
			start = fromDate
		}
		if end.After(toDate) {
			end = toDate
		}

		if start.Before(end) {
			periods = append([]models.TimeSeriesPeriod{{
				FromDate:    start.Format("2006-01-02"),
				ToDate:      end.Format("2006-01-02"),
				PeriodIndex: i + 1,
				Label:       fmt.Sprintf("Month of %s", start.Format("Jan 2006")),
			}}, periods...)
		}
	}

	return periods
}

// Helper function to generate time periods based on survey data existence
func (s *analyticsSvc) generateDataDrivenSurveyIntervals(fromDate, toDate time.Time) []models.TimeSeriesPeriod {
	var periods []models.TimeSeriesPeriod

	totalDuration := toDate.Sub(fromDate)

	var intervalDuration time.Duration
	if totalDuration <= 30*24*time.Hour {
		intervalDuration = 24 * time.Hour
	} else if totalDuration <= 365*24*time.Hour {
		intervalDuration = 7 * 24 * time.Hour
	} else {
		intervalDuration = 30 * 24 * time.Hour
	}

	surveyAnswerColl := database.Mdb.Database(s.db).Collection("loyaltySurveyAnswers")

	periodIndex := 1
	for current := fromDate; current.Before(toDate) && len(periods) < 10; current = current.Add(intervalDuration) {
		start := current
		end := current.Add(intervalDuration)
		if end.After(toDate) {
			end = toDate
		}

		// Check if this period has survey data
		endInclusive := end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		matchFilter := bson.M{
			"trash": bson.M{"$ne": true},
			"createdAt": bson.M{
				"$gte": start,
				"$lte": endInclusive,
			},
		}

		count, err := surveyAnswerColl.CountDocuments(s.ctx, matchFilter)
		if err == nil && count > 0 {
			var label string
			if intervalDuration == 24*time.Hour {
				label = fmt.Sprintf("Day %s", start.Format("Jan 02"))
			} else if intervalDuration == 7*24*time.Hour {
				label = fmt.Sprintf("Week of %s", start.Format("Jan 02"))
			} else {
				label = fmt.Sprintf("Month of %s", start.Format("Jan 2006"))
			}

			periods = append(periods, models.TimeSeriesPeriod{
				FromDate:    start.Format("2006-01-02"),
				ToDate:      end.Format("2006-01-02"),
				PeriodIndex: periodIndex,
				Label:       label,
			})
			periodIndex++

			fmt.Printf("DEBUG: Found survey data in period %s to %s (%d answers)\n",
				start.Format("2006-01-02"), end.Format("2006-01-02"), count)
		}
	}

	return periods
}

// Helper function to check if filters contain date-related parameters
func hasDateFilters(filters map[string]interface{}) bool {
	if filters == nil {
		return false
	}
	_, hasStart := filters["start_date"]
	_, hasEnd := filters["end_date"]
	_, hasYear := filters["year"]
	_, hasMonth := filters["month"]
	_, hasDays := filters["days"]
	return hasStart || hasEnd || hasYear || hasMonth || hasDays
}

// Helper function to get filtered users count from users service based on date range
func (s *analyticsSvc) getFilteredUsersCount(client string, filters map[string]interface{}) (int64, error) {
	// Build URL for users service
	usersServiceURL := os.Getenv("USERS_URL")
	if usersServiceURL == "" {
		usersServiceURL = "http://localhost:3201"
	}

	// Build query parameters for filtering
	params := url.Values{}
	if filters != nil {
		if startDate, ok := filters["start_date"].(time.Time); ok {
			params.Add("start_date", startDate.Format("2006-01-02"))
		}
		if endDate, ok := filters["end_date"].(time.Time); ok {
			params.Add("end_date", endDate.Format("2006-01-02"))
		}
		if year, ok := filters["year"].(int); ok {
			params.Add("year", fmt.Sprintf("%d", year))
		}
		if month, ok := filters["month"].(int); ok {
			params.Add("month", fmt.Sprintf("%d", month))
		}
		if days, ok := filters["days"].(int); ok {
			params.Add("days", fmt.Sprintf("%d", days))
		}
	}

	// Make request to users service
	url := fmt.Sprintf("%s/users?%s", usersServiceURL, params.Encode())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-client", client)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		// If it's a subscription error, fallback to local wallet data
		if resp.StatusCode == 500 && strings.Contains(string(body), "subscription") {
			return s.countUsersFromWalletsWithFilters(filters)
		}

		return 0, fmt.Errorf("users service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse users response
	var usersResponse struct {
		Users []struct {
			ID        string    `json:"_id"`
			CreatedAt time.Time `json:"createdAt"`
		} `json:"users"`
		Pagination struct {
			TotalPages int   `json:"totalPages"`
			PerPage    int   `json:"perPage"`
			TotalCount int64 `json:"totalCount"`
		} `json:"pagination"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&usersResponse); err != nil {
		return 0, err
	}

	return usersResponse.Pagination.TotalCount, nil
}

// Helper function to get filtered app shares count based on date range
func (s *analyticsSvc) getFilteredAppSharesCount(filters map[string]interface{}) (int64, error) {
	transactionColl := database.Mdb.Database(s.db).Collection("loyaltyTransactions")

	// Build match filter based on provided filters
	matchFilter := bson.M{
		"type":  constants.TXN_APP_SHARED,
		"trash": bson.M{"$ne": true},
	}

	if filters != nil {
		if startDate, ok := filters["start_date"].(time.Time); ok {
			matchFilter["createdAt"] = bson.M{"$gte": startDate}
		}
		if endDate, ok := filters["end_date"].(time.Time); ok {
			if existingFilter, exists := matchFilter["createdAt"]; exists {
				matchFilter["createdAt"] = bson.M{"$gte": existingFilter.(bson.M)["$gte"], "$lte": endDate}
			} else {
				matchFilter["createdAt"] = bson.M{"$lte": endDate}
			}
		}
		if year, ok := filters["year"].(int); ok {
			startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
			endOfYear := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
			matchFilter["createdAt"] = bson.M{"$gte": startOfYear, "$lt": endOfYear}
		}
		if month, ok := filters["month"].(int); ok {
			// If month is specified without year, use current year
			year := time.Now().Year()
			if y, hasYear := filters["year"].(int); hasYear {
				year = y
			}
			startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
			endOfMonth := startOfMonth.AddDate(0, 1, 0)
			matchFilter["createdAt"] = bson.M{"$gte": startOfMonth, "$lt": endOfMonth}
		}
		if days, ok := filters["days"].(int); ok {
			since := time.Now().AddDate(0, 0, -days)
			matchFilter["createdAt"] = bson.M{"$gte": since}
		}
	}

	count, err := transactionColl.CountDocuments(s.ctx, matchFilter)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetTimeSeriesEnhancedTransactionAnalytics returns enhanced transaction analytics only for periods where data exists
func (s *analyticsSvc) GetTimeSeriesEnhancedTransactionAnalytics(fromDate, toDate time.Time) (*models.TimeSeriesEnhancedTransactionAnalytics, error) {
	fmt.Printf("DEBUG: Time-series enhanced analytics from %v to %v\n", fromDate, toDate)

	periods := s.generateDataDrivenTransactionIntervals(fromDate, toDate)
	var analytics []models.EnhancedTransactionAnalytics
	var periodsWithData []models.TimeSeriesPeriod

	for _, period := range periods {
		startDate, err := time.Parse("2006-01-02", period.FromDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing start date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}
		endDate, err := time.Parse("2006-01-02", period.ToDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing end date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Add time to make end date inclusive (end of day)
		endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		filters := map[string]interface{}{
			"start_date": startDate,
			"end_date":   endDate,
		}

		periodAnalytics, err := s.GetEnhancedTransactionAnalytics(filters)
		if err != nil {
			fmt.Printf("DEBUG: Error getting analytics for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Ensure we have non-nil slices
		if periodAnalytics == nil {
			periodAnalytics = &models.EnhancedTransactionAnalytics{}
		}
		if periodAnalytics.TransactionsByServiceCategory == nil {
			periodAnalytics.TransactionsByServiceCategory = []models.TransactionServiceCategoryStats{}
		}
		if periodAnalytics.PointsTransferredPerDay == nil {
			periodAnalytics.PointsTransferredPerDay = []models.DailyPointsTransferStats{}
		}
		if periodAnalytics.PendingPointsTransactionsPerDay == nil {
			periodAnalytics.PendingPointsTransactionsPerDay = []models.DailyPendingPointsStats{}
		}

		// Check if period has meaningful data
		hasData := len(periodAnalytics.TransactionsByServiceCategory) > 0 ||
			len(periodAnalytics.PointsTransferredPerDay) > 0 ||
			len(periodAnalytics.PendingPointsTransactionsPerDay) > 0

		if hasData {
			analytics = append(analytics, *periodAnalytics)
			periodsWithData = append(periodsWithData, period)
			fmt.Printf("DEBUG: Period %d has data: %d categories, %d points days, %d pending days\n",
				period.PeriodIndex,
				len(periodAnalytics.TransactionsByServiceCategory),
				len(periodAnalytics.PointsTransferredPerDay),
				len(periodAnalytics.PendingPointsTransactionsPerDay))
		} else {
			fmt.Printf("DEBUG: Period %d has no transaction data, skipping\n", period.PeriodIndex)
		}
	}

	return &models.TimeSeriesEnhancedTransactionAnalytics{
		TimePeriods: periodsWithData,
		Analytics:   analytics,
	}, nil
}

// GetTimeSeriesTransactionAnalytics returns transaction analytics only for periods where data exists
func (s *analyticsSvc) GetTimeSeriesTransactionAnalytics(fromDate, toDate time.Time) (*models.TimeSeriesTransactionAnalytics, error) {
	fmt.Printf("DEBUG: Time-series transaction analytics from %v to %v\n", fromDate, toDate)

	periods := s.generateDataDrivenTransactionIntervals(fromDate, toDate)
	var analytics []models.TransactionAnalytics
	var periodsWithData []models.TimeSeriesPeriod

	for _, period := range periods {
		startDate, err := time.Parse("2006-01-02", period.FromDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing start date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		endDate, err := time.Parse("2006-01-02", period.ToDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing end date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Make end date inclusive
		endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		filters := map[string]interface{}{
			"start_date": startDate,
			"end_date":   endDate,
		}

		periodAnalytics, err := s.GetTransactionAnalytics(filters)
		if err != nil {
			fmt.Printf("DEBUG: Error getting transaction analytics for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Check if period has meaningful transaction data
		hasData := periodAnalytics.TotalRedemptions > 0 ||
			len(periodAnalytics.TransactionsByService) > 0 ||
			len(periodAnalytics.PointsPerDay) > 0 ||
			periodAnalytics.PendingPointsCount > 0

		if hasData {
			analytics = append(analytics, *periodAnalytics)
			periodsWithData = append(periodsWithData, period)
			fmt.Printf("DEBUG: Period %d has transaction data: %d redemptions, %d services, %d point days\n",
				period.PeriodIndex,
				periodAnalytics.TotalRedemptions,
				len(periodAnalytics.TransactionsByService),
				len(periodAnalytics.PointsPerDay))
		} else {
			fmt.Printf("DEBUG: Period %d has no transaction data, skipping\n", period.PeriodIndex)
		}
	}

	return &models.TimeSeriesTransactionAnalytics{
		TimePeriods: periodsWithData,
		Analytics:   analytics,
	}, nil
}

// GetTimeSeriesSurveyAnalytics returns survey analytics only for periods where data exists
func (s *analyticsSvc) GetTimeSeriesSurveyAnalytics(fromDate, toDate time.Time) (*models.TimeSeriesSurveyAnalytics, error) {
	fmt.Printf("DEBUG: Time-series survey analytics from %v to %v\n", fromDate, toDate)

	periods := s.generateDataDrivenSurveyIntervals(fromDate, toDate)
	var analytics []models.SurveyAnalytics
	var periodsWithData []models.TimeSeriesPeriod

	for _, period := range periods {
		startDate, err := time.Parse("2006-01-02", period.FromDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing start date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		endDate, err := time.Parse("2006-01-02", period.ToDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing end date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Make end date inclusive
		endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		filters := map[string]interface{}{
			"start_date": startDate,
			"end_date":   endDate,
		}

		periodAnalytics, err := s.GetSurveyAnalytics(filters)
		if err != nil {
			fmt.Printf("DEBUG: Error getting survey analytics for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Check if period has meaningful survey data
		hasData := len(periodAnalytics.SurveysByCategory) > 0

		if hasData {
			analytics = append(analytics, *periodAnalytics)
			periodsWithData = append(periodsWithData, period)
			fmt.Printf("DEBUG: Period %d has survey data: %d categories\n",
				period.PeriodIndex,
				len(periodAnalytics.SurveysByCategory))
		} else {
			fmt.Printf("DEBUG: Period %d has no survey data, skipping\n", period.PeriodIndex)
		}
	}

	return &models.TimeSeriesSurveyAnalytics{
		TimePeriods: periodsWithData,
		Analytics:   analytics,
	}, nil
}

// Helper function to build query string for external services with date range
func (s *analyticsSvc) buildQueryWithDateRange(startDate, endDate time.Time) string {
	// Build query string using createdAt filter format that external services expect
	// Format the dates in ISO format with timezone
	queryObj := map[string]interface{}{
		"createdAt": map[string]string{
			"$gte": startDate.UTC().Format("2006-01-02T15:04:05.000Z"),
			"$lte": endDate.UTC().Format("2006-01-02T15:04:05.000Z"),
		},
	}

	queryJSON, _ := json.Marshal(queryObj)
	query := fmt.Sprintf("query=%s", url.QueryEscape(string(queryJSON)))
	return query
}

// Helper function to extract count from external service response
func (s *analyticsSvc) extractCountFromResponse(response interface{}, key string) int {
	if responseMap, ok := response.(map[string]interface{}); ok {
		if count, exists := responseMap[key]; exists {
			if countFloat, ok := count.(float64); ok {
				return int(countFloat)
			}
			if countInt, ok := count.(int); ok {
				return countInt
			}
		}
	}
	return 0
}

// GetTimeSeriesExternalAnalytics returns external analytics only for periods where data exists
func (s *analyticsSvc) GetTimeSeriesExternalAnalytics(xclient, accessToken string, fromDate, toDate time.Time) (*models.TimeSeriesExternalAnalytics, error) {
	fmt.Printf("DEBUG: Time-series external analytics from %v to %v\n", fromDate, toDate)

	// Generate meaningful time periods based on data density
	periods := s.generateDataDrivenTimeIntervals(xclient, accessToken, fromDate, toDate)
	var analytics []models.ExternalAnalytics
	var periodsWithData []models.TimeSeriesPeriod

	for _, period := range periods {
		startDate, err := time.Parse("2006-01-02", period.FromDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing start date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		endDate, err := time.Parse("2006-01-02", period.ToDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing end date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Make end date inclusive (end of day)
		endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		query := s.buildQueryWithDateRange(startDate, endDate)
		fmt.Printf("DEBUG: Period %d dates: %s to %s\n", period.PeriodIndex, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

		// Get all counts for this period
		periodAnalytics := models.ExternalAnalytics{}

		// Get bookings count
		if bookingResponse, err := s.GetTotalBookings(xclient, accessToken, query); err == nil {
			periodAnalytics.TotalBookings = s.extractCountFromResponse(bookingResponse, "total_bookings")
		} else {
			fmt.Printf("DEBUG: Error getting bookings for period %d: %v\n", period.PeriodIndex, err)
		}

		// Get online consultations count
		if consultationResponse, err := s.GetTotalOnlineConsultations(xclient, accessToken, query); err == nil {
			periodAnalytics.TotalOnlineConsultations = s.extractCountFromResponse(consultationResponse, "total_online_consultations")
		} else {
			fmt.Printf("DEBUG: Error getting consultations for period %d: %v\n", period.PeriodIndex, err)
		}

		// Get smile test drives count
		if testDriveResponse, err := s.GetTotalSmileTestDrives(xclient, accessToken, query); err == nil {
			periodAnalytics.TotalSmileTestDrives = s.extractCountFromResponse(testDriveResponse, "total_smile_test_drives")
		} else {
			fmt.Printf("DEBUG: Error getting test drives for period %d: %v\n", period.PeriodIndex, err)
		}

		// Since generateDataDrivenTimeIntervals already filters for periods with data,
		// all periods should have data, but double-check for safety
		totalCount := periodAnalytics.TotalBookings + periodAnalytics.TotalOnlineConsultations + periodAnalytics.TotalSmileTestDrives
		if totalCount > 0 {
			analytics = append(analytics, periodAnalytics)
			periodsWithData = append(periodsWithData, period)
			fmt.Printf("DEBUG: Period %d has data: %d bookings, %d consultations, %d test drives\n",
				period.PeriodIndex,
				periodAnalytics.TotalBookings,
				periodAnalytics.TotalOnlineConsultations,
				periodAnalytics.TotalSmileTestDrives)
		} else {
			fmt.Printf("DEBUG: Period %d has no data, skipping\n", period.PeriodIndex)
		}
	}

	return &models.TimeSeriesExternalAnalytics{
		TimePeriods: periodsWithData,
		Analytics:   analytics,
	}, nil
}

// GetTimeSeriesBookingAnalytics returns booking analytics only for periods where data exists
func (s *analyticsSvc) GetTimeSeriesBookingAnalytics(xclient, accessToken string, fromDate, toDate time.Time) (*models.TimeSeriesBookingAnalytics, error) {
	fmt.Printf("DEBUG: Time-series booking analytics from %v to %v\n", fromDate, toDate)

	periods := s.generateDataDrivenBookingIntervals(xclient, accessToken, fromDate, toDate)
	var analytics []models.BookingAnalytics
	var periodsWithData []models.TimeSeriesPeriod

	for _, period := range periods {
		startDate, err := time.Parse("2006-01-02", period.FromDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing start date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		endDate, err := time.Parse("2006-01-02", period.ToDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing end date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Make end date inclusive
		endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		query := s.buildQueryWithDateRange(startDate, endDate)

		periodAnalytics := models.BookingAnalytics{}

		if bookingResponse, err := s.GetTotalBookings(xclient, accessToken, query); err == nil {
			periodAnalytics.TotalBookings = s.extractCountFromResponse(bookingResponse, "total_bookings")
		} else {
			fmt.Printf("DEBUG: Error getting bookings for period %d: %v\n", period.PeriodIndex, err)
		}

		// Only include periods with actual booking data
		if periodAnalytics.TotalBookings > 0 {
			analytics = append(analytics, periodAnalytics)
			periodsWithData = append(periodsWithData, period)
			fmt.Printf("DEBUG: Period %d has %d bookings\n", period.PeriodIndex, periodAnalytics.TotalBookings)
		} else {
			fmt.Printf("DEBUG: Period %d has no booking data, skipping\n", period.PeriodIndex)
		}
	}

	return &models.TimeSeriesBookingAnalytics{
		TimePeriods: periodsWithData,
		Analytics:   analytics,
	}, nil
}

// GetTimeSeriesOnlineConsultationAnalytics returns online consultation analytics only for periods where data exists
func (s *analyticsSvc) GetTimeSeriesOnlineConsultationAnalytics(xclient, accessToken string, fromDate, toDate time.Time) (*models.TimeSeriesOnlineConsultationAnalytics, error) {
	fmt.Printf("DEBUG: Time-series online consultation analytics from %v to %v\n", fromDate, toDate)

	periods := s.generateDataDrivenConsultationIntervals(xclient, accessToken, fromDate, toDate)
	var analytics []models.OnlineConsultationAnalytics
	var periodsWithData []models.TimeSeriesPeriod

	for _, period := range periods {
		startDate, err := time.Parse("2006-01-02", period.FromDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing start date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		endDate, err := time.Parse("2006-01-02", period.ToDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing end date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Make end date inclusive
		endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		query := s.buildQueryWithDateRange(startDate, endDate)

		periodAnalytics := models.OnlineConsultationAnalytics{}

		if consultationResponse, err := s.GetTotalOnlineConsultations(xclient, accessToken, query); err == nil {
			periodAnalytics.TotalOnlineConsultations = s.extractCountFromResponse(consultationResponse, "total_online_consultations")
		} else {
			fmt.Printf("DEBUG: Error getting consultations for period %d: %v\n", period.PeriodIndex, err)
		}

		// Only include periods with actual consultation data
		if periodAnalytics.TotalOnlineConsultations > 0 {
			analytics = append(analytics, periodAnalytics)
			periodsWithData = append(periodsWithData, period)
			fmt.Printf("DEBUG: Period %d has %d consultations\n", period.PeriodIndex, periodAnalytics.TotalOnlineConsultations)
		} else {
			fmt.Printf("DEBUG: Period %d has no consultation data, skipping\n", period.PeriodIndex)
		}
	}

	return &models.TimeSeriesOnlineConsultationAnalytics{
		TimePeriods: periodsWithData,
		Analytics:   analytics,
	}, nil
}

// GetTimeSeriesSmileTestDriveAnalytics returns smile test drive analytics only for periods where data exists
func (s *analyticsSvc) GetTimeSeriesSmileTestDriveAnalytics(xclient, accessToken string, fromDate, toDate time.Time) (*models.TimeSeriesSmileTestDriveAnalytics, error) {
	fmt.Printf("DEBUG: Time-series smile test drive analytics from %v to %v\n", fromDate, toDate)

	periods := s.generateDataDrivenSmileTestDriveIntervals(xclient, accessToken, fromDate, toDate)
	var analytics []models.SmileTestDriveAnalytics
	var periodsWithData []models.TimeSeriesPeriod

	for _, period := range periods {
		startDate, err := time.Parse("2006-01-02", period.FromDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing start date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		endDate, err := time.Parse("2006-01-02", period.ToDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing end date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Make end date inclusive
		endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		query := s.buildQueryWithDateRange(startDate, endDate)

		periodAnalytics := models.SmileTestDriveAnalytics{}

		if testDriveResponse, err := s.GetTotalSmileTestDrives(xclient, accessToken, query); err == nil {
			periodAnalytics.TotalSmileTestDrives = s.extractCountFromResponse(testDriveResponse, "total_smile_test_drives")
		} else {
			fmt.Printf("DEBUG: Error getting test drives for period %d: %v\n", period.PeriodIndex, err)
		}

		// Only include periods with actual test drive data
		if periodAnalytics.TotalSmileTestDrives > 0 {
			analytics = append(analytics, periodAnalytics)
			periodsWithData = append(periodsWithData, period)
			fmt.Printf("DEBUG: Period %d has %d test drives\n", period.PeriodIndex, periodAnalytics.TotalSmileTestDrives)
		} else {
			fmt.Printf("DEBUG: Period %d has no test drive data, skipping\n", period.PeriodIndex)
		}
	}

	return &models.TimeSeriesSmileTestDriveAnalytics{
		TimePeriods: periodsWithData,
		Analytics:   analytics,
	}, nil
}

// Fallback method to generate periods using local wallet data when users service is unavailable
func (s *analyticsSvc) generateDataDrivenUserIntervalsFromWallets(fromDate, toDate time.Time) []models.TimeSeriesPeriod {
	var periods []models.TimeSeriesPeriod

	totalDuration := toDate.Sub(fromDate)

	var intervalDuration time.Duration
	if totalDuration <= 30*24*time.Hour {
		intervalDuration = 24 * time.Hour
	} else if totalDuration <= 365*24*time.Hour {
		intervalDuration = 7 * 24 * time.Hour
	} else {
		intervalDuration = 30 * 24 * time.Hour
	}

	userWalletColl := database.Mdb.Database(s.db).Collection("loyaltyUserWallets")

	periodIndex := 1
	for current := fromDate; current.Before(toDate) && len(periods) < 10; current = current.Add(intervalDuration) {
		start := current
		end := current.Add(intervalDuration)
		if end.After(toDate) {
			end = toDate
		}

		// Check if this period has user wallet data
		endInclusive := end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		matchFilter := bson.M{
			"trash": bson.M{"$ne": true},
			"createdAt": bson.M{
				"$gte": start,
				"$lte": endInclusive,
			},
		}

		count, err := userWalletColl.CountDocuments(s.ctx, matchFilter)
		fmt.Printf("DEBUG: Local wallet check for period %s to %s: %d wallets (error: %v)\n",
			start.Format("2006-01-02"), end.Format("2006-01-02"), count, err)

		if err == nil && count > 0 {
			var label string
			if intervalDuration == 24*time.Hour {
				label = fmt.Sprintf("Day %s", start.Format("Jan 02"))
			} else if intervalDuration == 7*24*time.Hour {
				label = fmt.Sprintf("Week of %s", start.Format("Jan 02"))
			} else {
				label = fmt.Sprintf("Month of %s", start.Format("Jan 2006"))
			}

			periods = append(periods, models.TimeSeriesPeriod{
				FromDate:    start.Format("2006-01-02"),
				ToDate:      end.Format("2006-01-02"),
				PeriodIndex: periodIndex,
				Label:       label,
			})
			periodIndex++

			fmt.Printf("DEBUG: Found user wallet data in period %s to %s (%d wallets)\n",
				start.Format("2006-01-02"), end.Format("2006-01-02"), count)
		}
	}

	fmt.Printf("DEBUG: Found %d periods with wallet data using local fallback\n", len(periods))
	return periods
}

// Fallback method to get users by period using local wallet data when users service is unavailable
func (s *analyticsSvc) getUsersByPeriodFromWallets(client string, filters map[string]interface{}, period string) (map[string]int64, error) {
	fmt.Printf("DEBUG: Using local wallet data for getUsersByPeriod (client: %s)\n", client)

	userWalletColl := database.Mdb.Database(s.db).Collection("loyaltyUserWallets")

	// Build match filter based on provided filters
	matchFilter := bson.M{"trash": bson.M{"$ne": true}}

	if filters != nil {
		if startDate, ok := filters["start_date"].(time.Time); ok {
			matchFilter["createdAt"] = bson.M{"$gte": startDate}
		}
		if endDate, ok := filters["end_date"].(time.Time); ok {
			if existingFilter, exists := matchFilter["createdAt"]; exists {
				matchFilter["createdAt"] = bson.M{"$gte": existingFilter.(bson.M)["$gte"], "$lte": endDate}
			} else {
				matchFilter["createdAt"] = bson.M{"$lte": endDate}
			}
		}
		if year, ok := filters["year"].(int); ok {
			startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
			endOfYear := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
			matchFilter["createdAt"] = bson.M{"$gte": startOfYear, "$lt": endOfYear}
		}
		if month, ok := filters["month"].(int); ok {
			year := time.Now().Year()
			if y, hasYear := filters["year"].(int); hasYear {
				year = y
			}
			startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
			endOfMonth := startOfMonth.AddDate(0, 1, 0)
			matchFilter["createdAt"] = bson.M{"$gte": startOfMonth, "$lt": endOfMonth}
		}
		if days, ok := filters["days"].(int); ok {
			since := time.Now().AddDate(0, 0, -days)
			matchFilter["createdAt"] = bson.M{"$gte": since}
		}
	}

	// Build grouping based on period
	var groupBy bson.M
	switch period {
	case "year":
		groupBy = bson.M{
			"year": bson.M{"$year": "$createdAt"},
		}
	case "day":
		groupBy = bson.M{
			"year":  bson.M{"$year": "$createdAt"},
			"month": bson.M{"$month": "$createdAt"},
			"day":   bson.M{"$dayOfMonth": "$createdAt"},
		}
	default: // month
		groupBy = bson.M{
			"year":  bson.M{"$year": "$createdAt"},
			"month": bson.M{"$month": "$createdAt"},
		}
	}

	// Aggregation pipeline
	pipeline := []bson.M{
		{"$match": matchFilter},
		{
			"$group": bson.M{
				"_id":   groupBy,
				"count": bson.M{"$sum": 1},
			},
		},
		{"$sort": bson.M{"_id": 1}},
	}

	cursor, err := userWalletColl.Aggregate(s.ctx, pipeline)
	if err != nil {
		return make(map[string]int64), err
	}
	defer cursor.Close(s.ctx)

	result := make(map[string]int64)
	for cursor.Next(s.ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		id := doc["_id"].(bson.M)
		var key string

		switch period {
		case "year":
			if year, ok := id["year"].(int32); ok {
				key = fmt.Sprintf("%d", year)
			}
		case "day":
			if year, okY := id["year"].(int32); okY {
				if month, okM := id["month"].(int32); okM {
					if day, okD := id["day"].(int32); okD {
						key = time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC).Format("2006-01-02")
					}
				}
			}
		default: // month
			if year, okY := id["year"].(int32); okY {
				if month, okM := id["month"].(int32); okM {
					key = time.Date(int(year), time.Month(month), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
				}
			}
		}

		if key != "" {
			if count, ok := doc["count"].(int32); ok {
				result[key] = int64(count)
			} else if count, ok := doc["count"].(int64); ok {
				result[key] = count
			}
		}
	}

	fmt.Printf("DEBUG: Local wallet users by period (%s): %+v\n", period, result)
	return result, nil
}

// Helper function to extract creation date from MongoDB ObjectId
func extractDateFromObjectId(objectId string) (time.Time, error) {
	if len(objectId) != 24 {
		return time.Time{}, fmt.Errorf("invalid ObjectId length")
	}

	// Extract timestamp from first 4 bytes (8 hex characters)
	timestampHex := objectId[:8]

	// Convert hex to bytes
	timestampBytes, err := hex.DecodeString(timestampHex)
	if err != nil {
		return time.Time{}, err
	}

	// Convert to timestamp (big endian)
	timestamp := int64(timestampBytes[0])<<24 | int64(timestampBytes[1])<<16 | int64(timestampBytes[2])<<8 | int64(timestampBytes[3])

	// Convert to time (MongoDB ObjectId timestamp is in seconds since Unix epoch)
	return time.Unix(timestamp, 0), nil
}

// Helper function to extract creation date from user (tries createdAt first, then ObjectId)
func extractUserCreationDate(user struct {
	ID        string      `json:"_id"`
	CreatedAt interface{} `json:"createdAt"`
}) (time.Time, error) {
	var createdAtStr string

	// First try to get date from createdAt field
	if user.CreatedAt != nil {
		switch v := user.CreatedAt.(type) {
		case string:
			createdAtStr = v
		case map[string]interface{}:
			// MongoDB $date format: {"$date": "2022-08-01T10:47:59.394Z"}
			if date, ok := v["$date"].(string); ok {
				createdAtStr = date
			}
		}
	}

	// If we got a createdAt string, try to parse it
	if createdAtStr != "" {
		// Common time formats to try
		timeFormats := []string{
			"2006-01-02T15:04:05Z07:00", // RFC3339 with timezone
			"2006-01-02T15:04:05.000Z",  // ISO with milliseconds
			"2006-01-02T15:04:05Z",      // ISO without timezone
			"2006-01-02T15:04:05",       // ISO without Z
			"2006-01-02 15:04:05",       // Simple format
			"2006-01-02",                // Date only
		}

		for _, format := range timeFormats {
			if createdAt, err := time.Parse(format, createdAtStr); err == nil {
				return createdAt, nil
			}
		}
	}

	// If createdAt parsing failed, try to extract from ObjectId
	if user.ID != "" {
		if createdAt, err := extractDateFromObjectId(user.ID); err == nil {
			return createdAt, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to extract creation date from user")
}

// GetAppSharingAnalytics returns count of users who shared (have code_sent > 0) within the date range
func (s *analyticsSvc) GetAppSharingAnalytics(fromDate, toDate time.Time) (int64, error) {
	userWalletColl := database.Mdb.Database(s.db).Collection("loyaltyUserWallets")

	matchFilter := bson.M{
		"trash":     bson.M{"$ne": true},
		"code_sent": bson.M{"$gt": 0},
		"createdAt": bson.M{
			"$gte": fromDate,
			"$lte": toDate,
		},
	}

	totalUsersWhoShared, err := userWalletColl.CountDocuments(s.ctx, matchFilter)
	if err != nil {
		return 0, err
	}

	return totalUsersWhoShared, nil
}

// GetVoucherRedemptionAnalytics returns count of users who redeemed vouchers within the date range
func (s *analyticsSvc) GetVoucherRedemptionAnalytics(fromDate, toDate time.Time) (int64, error) {
	transactionColl := database.Mdb.Database(s.db).Collection("loyaltyTransactions")

	// Use aggregation to count distinct users who redeemed vouchers in the date range
	// with proper filtering for non-trashed service groups
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"trash": bson.M{"$ne": true},
				"type":  constants.TXN_VOUCHER_USED,
				"createdAt": bson.M{
					"$gte": fromDate,
					"$lte": toDate,
				},
			},
		},
		// Join with vouchers to get service_id
		{
			"$lookup": bson.M{
				"from":         "loyaltyVouchers",
				"localField":   "ref_id",
				"foreignField": "_id",
				"as":           "voucher_info",
			},
		},
		{"$unwind": "$voucher_info"},
		// Join with services to get group information
		{
			"$lookup": bson.M{
				"from":         "loyaltyServices",
				"localField":   "voucher_info.service_id",
				"foreignField": "_id",
				"as":           "service_info",
			},
		},
		{"$unwind": "$service_info"},
		// Filter out trashed services
		{
			"$match": bson.M{
				"$or": []bson.M{
					{"service_info.trash": bson.M{"$ne": true}},
					{"service_info.trash": bson.M{"$exists": false}},
				},
			},
		},
		// Join with service groups to check trash status
		{
			"$lookup": bson.M{
				"from":         "loyaltyServiceGroups",
				"localField":   "service_info.group",
				"foreignField": "_id",
				"as":           "group_info",
			},
		},
		{"$unwind": "$group_info"},
		// Filter out trashed service groups
		{
			"$match": bson.M{
				"$or": []bson.M{
					{"group_info.trash": bson.M{"$ne": true}},
					{"group_info.trash": bson.M{"$exists": false}},
				},
			},
		},
		{
			"$group": bson.M{
				"_id": "$user_id", // Group by user_id to get unique users
			},
		},
		{
			"$count": "total_users_who_redeemed",
		},
	}

	cursor, err := transactionColl.Aggregate(s.ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(s.ctx)

	var result struct {
		TotalUsersWhoRedeemed int64 `bson:"total_users_who_redeemed"`
	}

	if cursor.Next(s.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, err
		}
		return result.TotalUsersWhoRedeemed, nil
	}

	// If no results, return 0
	return 0, nil
}

// GetVoucherRedemptionByServiceType returns hierarchical voucher redemption analytics by discount type, service groups, and services
func (s *analyticsSvc) GetVoucherRedemptionByServiceType(filters map[string]interface{}) (*models.VoucherRedemptionAnalytics, error) {
	transactionColl := database.Mdb.Database(s.db).Collection("loyaltyTransactions")

	// Build match filter for voucher redemption transactions
	matchFilter := bson.M{
		"trash": bson.M{"$ne": true},
		"type":  constants.TXN_VOUCHER_USED,
	}
	s.applyFiltersToMatch(matchFilter, filters)

	// Aggregation pipeline to build hierarchical structure: discount type -> service group -> service
	pipeline := []bson.M{
		{"$match": matchFilter},
		// Join with vouchers to get service_id
		{
			"$lookup": bson.M{
				"from":         "loyaltyVouchers",
				"localField":   "ref_id",
				"foreignField": "_id",
				"as":           "voucher_info",
			},
		},
		{"$unwind": "$voucher_info"},
		// Join with services to get group information and service name
		{
			"$lookup": bson.M{
				"from":         "loyaltyServices",
				"localField":   "voucher_info.service_id",
				"foreignField": "_id",
				"as":           "service_info",
			},
		},
		{"$unwind": "$service_info"},
		// Filter out trashed services
		{
			"$match": bson.M{
				"$or": []bson.M{
					{"service_info.trash": bson.M{"$ne": true}},
					{"service_info.trash": bson.M{"$exists": false}},
				},
			},
		},
		// Join with service groups to get is_special field and group name
		{
			"$lookup": bson.M{
				"from":         "loyaltyServiceGroups",
				"localField":   "service_info.group",
				"foreignField": "_id",
				"as":           "group_info",
			},
		},
		{"$unwind": "$group_info"},
		// Filter out trashed service groups
		{
			"$match": bson.M{
				"$or": []bson.M{
					{"group_info.trash": bson.M{"$ne": true}},
					{"group_info.trash": bson.M{"$exists": false}},
				},
			},
		},
		// Group by discount type, service group, and service
		{
			"$group": bson.M{
				"_id": bson.M{
					"is_special":   "$group_info.is_special",
					"group_name":   "$group_info.name.en",
					"service_name": "$service_info.name.en",
				},
				"unique_users":      bson.M{"$addToSet": "$user_id"}, // Unique users for redemptions
				"transaction_count": bson.M{"$sum": 1},               // Total transaction count for usages
			},
		},
		// Project to get counts
		{
			"$project": bson.M{
				"_id":               1,
				"redemption_count":  bson.M{"$size": "$unique_users"},
				"transaction_count": 1,
			},
		},
	}

	cursor, err := transactionColl.Aggregate(s.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	// Initialize the result structure
	specialOccasionDiscounts := models.VoucherDiscountTypeStats{
		TotalRedemptions: 0,
		TotalUsages:      0,
		ServiceGroups:    make(map[string]models.VoucherServiceGroup),
	}

	redeemableDiscounts := models.VoucherDiscountTypeStats{
		TotalRedemptions: 0,
		TotalUsages:      0,
		ServiceGroups:    make(map[string]models.VoucherServiceGroup),
	}

	// Process the aggregation results
	for cursor.Next(s.ctx) {
		var result struct {
			ID struct {
				IsSpecial   bool   `bson:"is_special"`
				GroupName   string `bson:"group_name"`
				ServiceName string `bson:"service_name"`
			} `bson:"_id"`
			RedemptionCount  int64 `bson:"redemption_count"`
			TransactionCount int64 `bson:"transaction_count"`
		}

		if err := cursor.Decode(&result); err != nil {
			continue
		}

		var discountType *models.VoucherDiscountTypeStats
		if result.ID.IsSpecial {
			discountType = &specialOccasionDiscounts
		} else {
			discountType = &redeemableDiscounts
		}

		// Update totals for discount type
		discountType.TotalRedemptions += result.RedemptionCount
		discountType.TotalUsages += result.TransactionCount

		// Initialize service group if not exists
		if _, exists := discountType.ServiceGroups[result.ID.GroupName]; !exists {
			discountType.ServiceGroups[result.ID.GroupName] = models.VoucherServiceGroup{
				TotalRedemptions: 0,
				TotalUsages:      0,
				Services:         make(map[string]models.VoucherServiceStats),
			}
		}

		// Update service group totals
		serviceGroup := discountType.ServiceGroups[result.ID.GroupName]
		serviceGroup.TotalRedemptions += result.RedemptionCount
		serviceGroup.TotalUsages += result.TransactionCount

		// Add service stats
		serviceGroup.Services[result.ID.ServiceName] = models.VoucherServiceStats{
			Redemptions: result.RedemptionCount,
			Usages:      result.TransactionCount,
		}

		// Update the service group in the map
		discountType.ServiceGroups[result.ID.GroupName] = serviceGroup
	}

	totalRedemptions := specialOccasionDiscounts.TotalRedemptions + redeemableDiscounts.TotalRedemptions

	return &models.VoucherRedemptionAnalytics{
		SpecialOccasionDiscounts: specialOccasionDiscounts,
		RedeemableDiscounts:      redeemableDiscounts,
		TotalRedemptions:         totalRedemptions,
	}, nil
}

// GetTimeSeriesVoucherRedemptionAnalytics returns voucher redemption analytics by service type over time periods
func (s *analyticsSvc) GetTimeSeriesVoucherRedemptionAnalytics(fromDate, toDate time.Time) (*models.TimeSeriesVoucherRedemptionAnalytics, error) {
	fmt.Printf("DEBUG: Time-series voucher redemption analytics from %v to %v\n", fromDate, toDate)

	// Generate time periods based on voucher redemption data
	periods := s.generateDataDrivenVoucherRedemptionIntervals(fromDate, toDate)
	var analytics []models.VoucherRedemptionAnalytics
	var periodsWithData []models.TimeSeriesPeriod

	for _, period := range periods {
		startDate, err := time.Parse("2006-01-02", period.FromDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing start date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		endDate, err := time.Parse("2006-01-02", period.ToDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing end date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Add one day to end date to make it exclusive boundary for next period
		endDate = endDate.AddDate(0, 0, 1)

		filters := map[string]interface{}{
			"start_date": startDate,
			"end_date":   endDate,
		}

		periodAnalytics, err := s.GetVoucherRedemptionByServiceType(filters)
		if err != nil {
			fmt.Printf("DEBUG: Error getting voucher redemption analytics for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Check if period has meaningful voucher redemption data
		hasData := periodAnalytics.TotalRedemptions > 0

		if hasData {
			analytics = append(analytics, *periodAnalytics)
			periodsWithData = append(periodsWithData, period)
			fmt.Printf("DEBUG: Period %d has voucher redemption data: %d special, %d redeemable, %d total\n",
				period.PeriodIndex,
				periodAnalytics.SpecialOccasionDiscounts.TotalRedemptions,
				periodAnalytics.RedeemableDiscounts.TotalRedemptions,
				periodAnalytics.TotalRedemptions)
		} else {
			fmt.Printf("DEBUG: Period %d has no voucher redemption data, skipping\n", period.PeriodIndex)
		}
	}

	return &models.TimeSeriesVoucherRedemptionAnalytics{
		TimePeriods: periodsWithData,
		Analytics:   analytics,
	}, nil
}

// Helper function to generate time periods based on voucher redemption data existence
func (s *analyticsSvc) generateDataDrivenVoucherRedemptionIntervals(fromDate, toDate time.Time) []models.TimeSeriesPeriod {
	var periods []models.TimeSeriesPeriod

	totalDuration := toDate.Sub(fromDate)

	var intervalDuration time.Duration
	if totalDuration <= 30*24*time.Hour {
		intervalDuration = 24 * time.Hour
	} else if totalDuration <= 365*24*time.Hour {
		intervalDuration = 7 * 24 * time.Hour
	} else {
		intervalDuration = 30 * 24 * time.Hour
	}

	transactionColl := database.Mdb.Database(s.db).Collection("loyaltyTransactions")

	periodIndex := 1
	current := fromDate

	for current.Before(toDate) && len(periods) < 10 {
		start := current
		end := current.Add(intervalDuration)
		if end.After(toDate) {
			end = toDate
		}

		// Check if this period has voucher redemption data
		// Use $gte and $lt to avoid overlapping boundaries (start inclusive, end exclusive)
		matchFilter := bson.M{
			"trash": bson.M{"$ne": true},
			"type":  constants.TXN_VOUCHER_USED,
			"createdAt": bson.M{
				"$gte": start,
				"$lt":  end, // Use $lt instead of $lte to prevent overlap
			},
		}

		count, err := transactionColl.CountDocuments(s.ctx, matchFilter)
		if err == nil && count > 0 {
			var label string
			if intervalDuration == 24*time.Hour {
				label = fmt.Sprintf("Day %s", start.Format("Jan 02"))
			} else if intervalDuration == 7*24*time.Hour {
				label = fmt.Sprintf("Week of %s", start.Format("Jan 02"))
			} else {
				label = fmt.Sprintf("Month of %s", start.Format("Jan 2006"))
			}

			// Store end date as one day before the actual end to show non-overlapping periods
			displayEnd := end.AddDate(0, 0, -1)
			if end == toDate {
				displayEnd = end // Keep original end date for the last period
			}

			periods = append(periods, models.TimeSeriesPeriod{
				FromDate:    start.Format("2006-01-02"),
				ToDate:      displayEnd.Format("2006-01-02"),
				PeriodIndex: periodIndex,
				Label:       label,
			})
			periodIndex++

			fmt.Printf("DEBUG: Found voucher redemption data in period %s to %s (%d redemptions)\n",
				start.Format("2006-01-02"), displayEnd.Format("2006-01-02"), count)
		}

		// Move to next period (non-overlapping)
		current = end
	}

	return periods
}

// GetVoucherUsageByServiceType returns hierarchical voucher usage analytics by discount type, service groups, and services
func (s *analyticsSvc) GetVoucherUsageByServiceType(filters map[string]interface{}) (*models.VoucherUsageAnalytics, error) {
	transactionColl := database.Mdb.Database(s.db).Collection("loyaltyTransactions")

	// Build match filter for voucher usage transactions
	matchFilter := bson.M{
		"trash": bson.M{"$ne": true},
		"type":  constants.TXN_VOUCHER_USED,
	}
	s.applyFiltersToMatch(matchFilter, filters)

	// Aggregation pipeline to build hierarchical structure: discount type -> service group -> service
	pipeline := []bson.M{
		{"$match": matchFilter},
		// Join with vouchers to get service_id
		{
			"$lookup": bson.M{
				"from":         "loyaltyVouchers",
				"localField":   "ref_id",
				"foreignField": "_id",
				"as":           "voucher_info",
			},
		},
		{"$unwind": "$voucher_info"},
		// Join with services to get group information and service name
		{
			"$lookup": bson.M{
				"from":         "loyaltyServices",
				"localField":   "voucher_info.service_id",
				"foreignField": "_id",
				"as":           "service_info",
			},
		},
		{"$unwind": "$service_info"},
		// Filter out trashed services
		{
			"$match": bson.M{
				"$or": []bson.M{
					{"service_info.trash": bson.M{"$ne": true}},
					{"service_info.trash": bson.M{"$exists": false}},
				},
			},
		},
		// Join with service groups to get is_special field and group name
		{
			"$lookup": bson.M{
				"from":         "loyaltyServiceGroups",
				"localField":   "service_info.group",
				"foreignField": "_id",
				"as":           "group_info",
			},
		},
		{"$unwind": "$group_info"},
		// Filter out trashed service groups
		{
			"$match": bson.M{
				"$or": []bson.M{
					{"group_info.trash": bson.M{"$ne": true}},
					{"group_info.trash": bson.M{"$exists": false}},
				},
			},
		},
		// Group by discount type, service group, and service
		{
			"$group": bson.M{
				"_id": bson.M{
					"is_special":   "$group_info.is_special",
					"group_name":   "$group_info.name.en",
					"service_name": "$service_info.name.en",
				},
				"unique_users":      bson.M{"$addToSet": "$user_id"}, // Unique users for redemptions
				"transaction_count": bson.M{"$sum": 1},               // Total transaction count for usages
			},
		},
		// Project to get counts
		{
			"$project": bson.M{
				"_id":               1,
				"redemption_count":  bson.M{"$size": "$unique_users"},
				"transaction_count": 1,
			},
		},
	}

	cursor, err := transactionColl.Aggregate(s.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	// Initialize the result structure
	specialOccasionDiscounts := models.VoucherDiscountTypeStats{
		TotalRedemptions: 0,
		TotalUsages:      0,
		ServiceGroups:    make(map[string]models.VoucherServiceGroup),
	}

	redeemableDiscounts := models.VoucherDiscountTypeStats{
		TotalRedemptions: 0,
		TotalUsages:      0,
		ServiceGroups:    make(map[string]models.VoucherServiceGroup),
	}

	// Process the aggregation results
	for cursor.Next(s.ctx) {
		var result struct {
			ID struct {
				IsSpecial   bool   `bson:"is_special"`
				GroupName   string `bson:"group_name"`
				ServiceName string `bson:"service_name"`
			} `bson:"_id"`
			RedemptionCount  int64 `bson:"redemption_count"`
			TransactionCount int64 `bson:"transaction_count"`
		}

		if err := cursor.Decode(&result); err != nil {
			continue
		}

		var discountType *models.VoucherDiscountTypeStats
		if result.ID.IsSpecial {
			discountType = &specialOccasionDiscounts
		} else {
			discountType = &redeemableDiscounts
		}

		// Update totals for discount type
		discountType.TotalRedemptions += result.RedemptionCount
		discountType.TotalUsages += result.TransactionCount

		// Initialize service group if not exists
		if _, exists := discountType.ServiceGroups[result.ID.GroupName]; !exists {
			discountType.ServiceGroups[result.ID.GroupName] = models.VoucherServiceGroup{
				TotalRedemptions: 0,
				TotalUsages:      0,
				Services:         make(map[string]models.VoucherServiceStats),
			}
		}

		// Update service group totals
		serviceGroup := discountType.ServiceGroups[result.ID.GroupName]
		serviceGroup.TotalRedemptions += result.RedemptionCount
		serviceGroup.TotalUsages += result.TransactionCount

		// Add service stats
		serviceGroup.Services[result.ID.ServiceName] = models.VoucherServiceStats{
			Redemptions: result.RedemptionCount,
			Usages:      result.TransactionCount,
		}

		// Update the service group in the map
		discountType.ServiceGroups[result.ID.GroupName] = serviceGroup
	}

	totalUsages := specialOccasionDiscounts.TotalUsages + redeemableDiscounts.TotalUsages

	return &models.VoucherUsageAnalytics{
		SpecialOccasionDiscounts: specialOccasionDiscounts,
		RedeemableDiscounts:      redeemableDiscounts,
		TotalUsages:              totalUsages,
	}, nil
}

// GetTimeSeriesVoucherUsageAnalytics returns voucher usage analytics by service type over time periods
func (s *analyticsSvc) GetTimeSeriesVoucherUsageAnalytics(fromDate, toDate time.Time) (*models.TimeSeriesVoucherUsageAnalytics, error) {
	fmt.Printf("DEBUG: Time-series voucher usage analytics from %v to %v\n", fromDate, toDate)

	// Generate time periods based on voucher redemption data (reuse existing helper)
	periods := s.generateDataDrivenVoucherRedemptionIntervals(fromDate, toDate)
	var analytics []models.VoucherUsageAnalytics
	var periodsWithData []models.TimeSeriesPeriod

	for _, period := range periods {
		startDate, err := time.Parse("2006-01-02", period.FromDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing start date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		endDate, err := time.Parse("2006-01-02", period.ToDate)
		if err != nil {
			fmt.Printf("DEBUG: Error parsing end date for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Add one day to end date to make it exclusive boundary for next period
		endDate = endDate.AddDate(0, 0, 1)

		filters := map[string]interface{}{
			"start_date": startDate,
			"end_date":   endDate,
		}

		periodAnalytics, err := s.GetVoucherUsageByServiceType(filters)
		if err != nil {
			fmt.Printf("DEBUG: Error getting voucher usage analytics for period %d: %v\n", period.PeriodIndex, err)
			continue
		}

		// Check if period has meaningful voucher usage data
		hasData := periodAnalytics.TotalUsages > 0

		if hasData {
			analytics = append(analytics, *periodAnalytics)
			periodsWithData = append(periodsWithData, period)
			fmt.Printf("DEBUG: Period %d has voucher usage data: %d special, %d redeemable, %d total\n",
				period.PeriodIndex,
				periodAnalytics.SpecialOccasionDiscounts.TotalUsages,
				periodAnalytics.RedeemableDiscounts.TotalUsages,
				periodAnalytics.TotalUsages)
		} else {
			fmt.Printf("DEBUG: Period %d has no voucher usage data, skipping\n", period.PeriodIndex)
		}
	}

	return &models.TimeSeriesVoucherUsageAnalytics{
		TimePeriods: periodsWithData,
		Analytics:   analytics,
	}, nil
}
