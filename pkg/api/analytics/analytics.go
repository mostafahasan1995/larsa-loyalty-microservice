package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"loyalty/pkg/services"
	"net/http"
	"strconv"
	"time"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	goccyjson "github.com/goccy/go-json"
)

// GetDashboardAnalytics returns complete dashboard analytics
func GetDashboardAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse date filters if provided
	var filters map[string]interface{}
	if r.URL.Query().Get("from") != "" || r.URL.Query().Get("to") != "" {
		// Parse from and to dates
		fromDate, toDate, err := parseDateRange(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid date parameters: %v", err), http.StatusBadRequest)
			return
		}
		filters = map[string]interface{}{
			"start_date": fromDate,
			"end_date":   toDate,
		}
	}

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	analytics, err := analyticsService.GetDashboardAnalytics(headerParams.Client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If date filters are provided, update voucher analytics with filtered data
	if filters != nil {
		// Get filtered voucher redemption analytics
		voucherRedemptionAnalytics, err := analyticsService.GetVoucherRedemptionByServiceType(filters)
		if err == nil {
			analytics.VoucherRedemptionAnalytics = *voucherRedemptionAnalytics
		}

		// Get filtered voucher usage analytics
		voucherUsageAnalytics, err := analyticsService.GetVoucherUsageByServiceType(filters)
		if err == nil {
			analytics.VoucherUsageAnalytics = *voucherUsageAnalytics
		}
	}

	j, _ := goccyjson.MarshalContext(r.Context(), analytics)
	w.Write(j)
}

// GetUserCount returns count of users from date to date with app sharing analytics
func GetUserCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse from and to dates
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Get x-access-token header from original request
	accessToken := r.Header.Get("x-access-token")
	if accessToken == "" {
		http.Error(w, "x-access-token header is required", http.StatusBadRequest)
		return
	}

	// Get x-client header from original request
	clientHeader := r.Header.Get("x-client")
	if clientHeader == "" {
		http.Error(w, "x-client header is required", http.StatusBadRequest)
		return
	}

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)

	// Get user count data
	userCount, err := analyticsService.GetUserCount(accessToken, clientHeader, fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get app sharing analytics
	totalUsersWhoShared, err := analyticsService.GetAppSharingAnalytics(fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get voucher redemption analytics
	totalUsersWhoRedeemed, err := analyticsService.GetVoucherRedemptionAnalytics(fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Combine the responses
	response := map[string]interface{}{
		"user_count_data":          userCount,
		"total_users_who_shared":   totalUsersWhoShared,
		"total_users_who_redeemed": totalUsersWhoRedeemed,
		"from_date":                fromDate.Format("2006-01-02"),
		"to_date":                  toDate.Format("2006-01-02"),
		"sharing_note":             "App sharing count includes users who shared within the specified date range",
		"redemption_note":          "Voucher redemption count includes users who redeemed vouchers within the specified date range",
	}

	j, _ := goccyjson.MarshalContext(r.Context(), response)
	w.Write(j)
}

// GetTransactionAnalytics returns transaction-specific analytics with optional filters
func GetTransactionAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse query parameters
	filters := parseTransactionFilters(r)

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	transactionAnalytics, err := analyticsService.GetTransactionAnalytics(filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), transactionAnalytics)
	w.Write(j)
}

// GetSurveyAnalytics returns survey-specific analytics with optional filters
func GetSurveyAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse query parameters
	filters := parseSurveyFilters(r)

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	surveyAnalytics, err := analyticsService.GetSurveyAnalytics(filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), surveyAnalytics)
	w.Write(j)
}

// GetTotalBookings handles getting total bookings count
func GetTotalBookings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()
	query := r.URL.RawQuery
	// Get access token from header
	accessToken := r.Header.Get("x-access-token")
	if accessToken == "" {
		http.Error(w, "x-access-token header is required", http.StatusBadRequest)
		return
	}

	// Initialize analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)

	// Call GetTotalBookings
	result, err := analyticsService.GetTotalBookings(headerParams.Client, accessToken, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return result as JSON
	j, _ := goccyjson.MarshalContext(r.Context(), result)
	w.Write(j)
}

// GetTotalSmileTestDrives handles getting total smile test drives count
func GetTotalSmileTestDrives(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()
	query := r.URL.RawQuery
	// Get access token from header
	accessToken := r.Header.Get("x-access-token")
	if accessToken == "" {
		http.Error(w, "x-access-token header is required", http.StatusBadRequest)
		return
	}

	// Initialize analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)

	// Call GetTotalSmileTestDrives
	result, err := analyticsService.GetTotalSmileTestDrives(headerParams.Client, accessToken, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return result as JSON
	j, _ := goccyjson.MarshalContext(r.Context(), result)
	w.Write(j)
}

// GetTotalOnlineConsultations handles getting total online consultations count
func GetTotalOnlineConsultations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()
	query := r.URL.RawQuery
	// Get access token from header
	accessToken := r.Header.Get("x-access-token")
	if accessToken == "" {
		http.Error(w, "x-access-token header is required", http.StatusBadRequest)
		return
	}

	// Initialize analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)

	// Call GetTotalOnlineConsultations
	result, err := analyticsService.GetTotalOnlineConsultations(headerParams.Client, accessToken, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return result as JSON
	j, _ := goccyjson.MarshalContext(r.Context(), result)
	w.Write(j)
}

// GetTotalCounts handles getting all total counts
func GetTotalCounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()
	query := r.URL.RawQuery
	// Get access token from header
	accessToken := r.Header.Get("x-access-token")
	if accessToken == "" {
		http.Error(w, "x-access-token header is required", http.StatusBadRequest)
		return
	}

	// Initialize analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)

	// Call GetTotalCounts
	result, err := analyticsService.GetTotalCounts(headerParams.Client, accessToken, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return result as JSON
	j, _ := goccyjson.MarshalContext(r.Context(), result)
	w.Write(j)
}

// GetTransactionsByService returns enhanced transactions analytics with service-category breakdown and daily stats
func GetTransactionsByService(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse query parameters
	filters := parseTransactionFilters(r)

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	enhancedAnalytics, err := analyticsService.GetEnhancedTransactionAnalytics(filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), enhancedAnalytics)
	w.Write(j)
}

// GetPointsPerDay returns points transferred per day with optional date range
func GetPointsPerDay(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse query parameters
	filters := parseTimeFilters(r)

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	pointsPerDay, err := analyticsService.GetPointsPerDay(filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), pointsPerDay)
	w.Write(j)
}

// GetEnhancedTransactionAnalytics returns enhanced transaction analytics with detailed breakdown
func GetEnhancedTransactionAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse query parameters
	filters := parseTransactionFilters(r)

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	enhancedAnalytics, err := analyticsService.GetEnhancedTransactionAnalytics(filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), enhancedAnalytics)
	w.Write(j)
}

// Time-Series Analytics Endpoints

// GetTimeSeriesEnhancedTransactionAnalytics returns enhanced transaction analytics over 10 time periods
func GetTimeSeriesEnhancedTransactionAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse from and to dates
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	timeSeriesAnalytics, err := analyticsService.GetTimeSeriesEnhancedTransactionAnalytics(fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), timeSeriesAnalytics)
	w.Write(j)
}

// GetTimeSeriesTransactionAnalytics returns transaction analytics over 10 time periods
func GetTimeSeriesTransactionAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse from and to dates
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	timeSeriesAnalytics, err := analyticsService.GetTimeSeriesTransactionAnalytics(fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), timeSeriesAnalytics)
	w.Write(j)
}

// GetTimeSeriesSurveyAnalytics returns survey analytics over 10 time periods
func GetTimeSeriesSurveyAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse from and to dates
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	timeSeriesAnalytics, err := analyticsService.GetTimeSeriesSurveyAnalytics(fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), timeSeriesAnalytics)
	w.Write(j)
}

// External Time-Series Analytics Endpoints

// GetTimeSeriesExternalAnalytics returns all external analytics (bookings, consultations, test drives) over 10 time periods
func GetTimeSeriesExternalAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse from and to dates
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Get access token from headers
	accessToken := r.Header.Get("x-access-token")
	if accessToken == "" {
		http.Error(w, "Access token is required", http.StatusBadRequest)
		return
	}

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	timeSeriesAnalytics, err := analyticsService.GetTimeSeriesExternalAnalytics(headerParams.Client, accessToken, fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), timeSeriesAnalytics)
	w.Write(j)
}

// GetTimeSeriesBookingAnalytics returns booking analytics over 10 time periods
func GetTimeSeriesBookingAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse from and to dates
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Get access token from headers
	accessToken := r.Header.Get("x-access-token")
	if accessToken == "" {
		http.Error(w, "Access token is required", http.StatusBadRequest)
		return
	}

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	timeSeriesAnalytics, err := analyticsService.GetTimeSeriesBookingAnalytics(headerParams.Client, accessToken, fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), timeSeriesAnalytics)
	w.Write(j)
}

// GetTimeSeriesOnlineConsultationAnalytics returns online consultation analytics over 10 time periods
func GetTimeSeriesOnlineConsultationAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse from and to dates
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Get access token from headers
	accessToken := r.Header.Get("x-access-token")
	if accessToken == "" {
		http.Error(w, "Access token is required", http.StatusBadRequest)
		return
	}

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	timeSeriesAnalytics, err := analyticsService.GetTimeSeriesOnlineConsultationAnalytics(headerParams.Client, accessToken, fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), timeSeriesAnalytics)
	w.Write(j)
}

// GetTimeSeriesSmileTestDriveAnalytics returns smile test drive analytics over 10 time periods
func GetTimeSeriesSmileTestDriveAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse from and to dates
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Get access token from headers
	accessToken := r.Header.Get("x-access-token")
	if accessToken == "" {
		http.Error(w, "Access token is required", http.StatusBadRequest)
		return
	}

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	timeSeriesAnalytics, err := analyticsService.GetTimeSeriesSmileTestDriveAnalytics(headerParams.Client, accessToken, fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), timeSeriesAnalytics)
	w.Write(j)
}

// GetAppSharingAnalytics returns count of users who shared the app in the given date range
func GetAppSharingAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse from and to dates (same as user count API)
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	totalUsersWhoShared, err := analyticsService.GetAppSharingAnalytics(fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return simple response with count
	response := map[string]interface{}{
		"total_users_who_shared": totalUsersWhoShared,
		"from_date":              fromDate.Format("2006-01-02"),
		"to_date":                toDate.Format("2006-01-02"),
	}

	j, _ := goccyjson.MarshalContext(r.Context(), response)
	w.Write(j)
}

// GetVoucherRedemptionAnalytics returns count of users who redeemed vouchers within the date range
func GetVoucherRedemptionAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse from and to dates
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	totalUsersWhoRedeemed, err := analyticsService.GetVoucherRedemptionAnalytics(fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"total_users_who_redeemed": totalUsersWhoRedeemed,
		"from_date":                fromDate.Format("2006-01-02"),
		"to_date":                  toDate.Format("2006-01-02"),
		"note":                     "Count includes users who redeemed vouchers within the specified date range",
	}

	j, _ := goccyjson.MarshalContext(r.Context(), response)
	w.Write(j)
}

// GetVoucherRedemptionByServiceType returns count of users who redeemed vouchers by service type (special occasion vs welcoming)
func GetVoucherRedemptionByServiceType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse query parameters for filters
	filters := parseTransactionFilters(r)

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	voucherRedemptionAnalytics, err := analyticsService.GetVoucherRedemptionByServiceType(filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), voucherRedemptionAnalytics)
	w.Write(j)
}

// GetTimeSeriesVoucherRedemptionAnalytics returns voucher redemption analytics by service type over time periods
func GetTimeSeriesVoucherRedemptionAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse from and to dates
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	timeSeriesAnalytics, err := analyticsService.GetTimeSeriesVoucherRedemptionAnalytics(fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), timeSeriesAnalytics)
	w.Write(j)
}

// GetVoucherUsageByServiceType returns total count of voucher usages by service type (including repeated uses)
func GetVoucherUsageByServiceType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse query parameters for filters
	filters := parseTransactionFilters(r)

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	voucherUsageAnalytics, err := analyticsService.GetVoucherUsageByServiceType(filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), voucherUsageAnalytics)
	w.Write(j)
}

// GetTimeSeriesVoucherUsageAnalytics returns voucher usage analytics by service type over time periods
func GetTimeSeriesVoucherUsageAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	headerParams := common.ExtractHeaderParams(r)
	db := headerParams.Client
	ctx := context.Background()

	// Parse from and to dates
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date parameters: %v", err), http.StatusBadRequest)
		return
	}

	// Get analytics service
	analyticsService := services.NewAnalyticsService(ctx, db)
	timeSeriesAnalytics, err := analyticsService.GetTimeSeriesVoucherUsageAnalytics(fromDate, toDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, _ := goccyjson.MarshalContext(r.Context(), timeSeriesAnalytics)
	w.Write(j)
}

// Helper functions for parsing filters
func parseTimeFilters(r *http.Request) map[string]interface{} {
	filters := make(map[string]interface{})

	// Parse start_date
	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			filters["start_date"] = t
		}
	}

	// Parse end_date
	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			filters["end_date"] = t
		}
	}

	// Parse year
	if year := r.URL.Query().Get("year"); year != "" {
		if y, err := strconv.Atoi(year); err == nil {
			filters["year"] = y
		}
	}

	// Parse month
	if month := r.URL.Query().Get("month"); month != "" {
		if m, err := strconv.Atoi(month); err == nil && m >= 1 && m <= 12 {
			filters["month"] = m
		}
	}

	// Parse days (for last N days)
	if days := r.URL.Query().Get("days"); days != "" {
		if d, err := strconv.Atoi(days); err == nil && d > 0 {
			filters["days"] = d
		}
	}

	return filters
}

func parseTransactionFilters(r *http.Request) map[string]interface{} {
	filters := make(map[string]interface{})

	// Check if there's a JSON query parameter (like transactions service)
	if queryParam := r.URL.Query().Get("query"); queryParam != "" {
		var queryMap map[string]interface{}
		if err := json.Unmarshal([]byte(queryParam), &queryMap); err == nil {
			// Parse dates from JSON query
			if startDateStr, ok := queryMap["start_date"].(string); ok && startDateStr != "" {
				if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
					filters["start_date"] = t
				}
			}
			if endDateStr, ok := queryMap["end_date"].(string); ok && endDateStr != "" {
				if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
					filters["end_date"] = t
				}
			}

			// Parse other fields from JSON query
			if serviceID, ok := queryMap["service_id"].(string); ok && serviceID != "" {
				filters["service_id"] = serviceID
			}
			if txnType, ok := queryMap["type"].(string); ok && txnType != "" {
				filters["type"] = txnType
			}
			if userID, ok := queryMap["user_id"].(string); ok && userID != "" {
				filters["user_id"] = userID
			}
			if minAmount, ok := queryMap["min_amount"].(float64); ok {
				filters["min_amount"] = minAmount
			}
			if maxAmount, ok := queryMap["max_amount"].(float64); ok {
				filters["max_amount"] = maxAmount
			}

			return filters
		}
	}

	// Fallback to individual query parameters for backward compatibility
	timeFilters := parseTimeFilters(r)
	for k, v := range timeFilters {
		filters[k] = v
	}

	// Handle 'from' and 'to' parameters (alternative to start_date/end_date)
	if fromDate := r.URL.Query().Get("from"); fromDate != "" {
		if t, err := time.Parse("2006-01-02", fromDate); err == nil {
			filters["start_date"] = t
		}
	}
	if toDate := r.URL.Query().Get("to"); toDate != "" {
		if t, err := time.Parse("2006-01-02", toDate); err == nil {
			filters["end_date"] = t
		}
	}

	// Parse service_id
	if serviceID := r.URL.Query().Get("service_id"); serviceID != "" {
		filters["service_id"] = serviceID
	}

	// Parse transaction type
	if txnType := r.URL.Query().Get("type"); txnType != "" {
		filters["type"] = txnType
	}

	// Parse user_id
	if userID := r.URL.Query().Get("user_id"); userID != "" {
		filters["user_id"] = userID
	}

	// Parse amount range
	if minAmount := r.URL.Query().Get("min_amount"); minAmount != "" {
		if amt, err := strconv.ParseFloat(minAmount, 64); err == nil {
			filters["min_amount"] = amt
		}
	}

	if maxAmount := r.URL.Query().Get("max_amount"); maxAmount != "" {
		if amt, err := strconv.ParseFloat(maxAmount, 64); err == nil {
			filters["max_amount"] = amt
		}
	}

	return filters
}

func parseSurveyFilters(r *http.Request) map[string]interface{} {
	filters := parseTimeFilters(r)

	// Parse survey_id
	if surveyID := r.URL.Query().Get("survey_id"); surveyID != "" {
		filters["survey_id"] = surveyID
	}

	// Parse category
	if category := r.URL.Query().Get("category"); category != "" {
		filters["category"] = category
	}

	// Parse rating range
	if minRating := r.URL.Query().Get("min_rating"); minRating != "" {
		if rating, err := strconv.ParseFloat(minRating, 64); err == nil {
			filters["min_rating"] = rating
		}
	}

	if maxRating := r.URL.Query().Get("max_rating"); maxRating != "" {
		if rating, err := strconv.ParseFloat(maxRating, 64); err == nil {
			filters["max_rating"] = rating
		}
	}

	return filters
}

// Helper function to parse date range from query parameters
func parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if fromStr == "" || toStr == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("both 'from' and 'to' date parameters are required")
	}

	fromDate, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' date format, expected YYYY-MM-DD: %v", err)
	}

	toDate, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid 'to' date format, expected YYYY-MM-DD: %v", err)
	}

	if toDate.Before(fromDate) {
		return time.Time{}, time.Time{}, fmt.Errorf("'to' date must be after 'from' date")
	}

	return fromDate, toDate, nil
}
