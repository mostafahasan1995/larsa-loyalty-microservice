package models

import "loyalty/pkg/constants"

// User Analytics Models
type UserAnalytics struct {
	TotalUsers     int64            `json:"total_users"`
	UsersByMonth   map[string]int64 `json:"users_by_month"`  // Backward compatibility - now supports different periods
	UsersByPeriod  map[string]int64 `json:"users_by_period"` // New field for explicit period-based data
	NewUsersByDay  map[string]int64 `json:"new_users_by_day"`
	TotalAppShares int64            `json:"total_app_shares"`
	Period         string           `json:"period,omitempty"` // Current period type (year, month, day)
}

// Transaction Analytics Models
type TransactionAnalytics struct {
	TotalRedemptions      int64                              `json:"total_redemptions"`
	TransactionsByService map[string]TransactionServiceStats `json:"transactions_by_service"`
	PointsPerDay          map[string]float64                 `json:"points_per_day"`
	PendingPointsCount    int64                              `json:"pending_points_count"`
	TotalPendingPoints    float64                            `json:"total_pending_points"`
}

type TransactionServiceStats struct {
	Count       int64             `json:"count"`
	TotalAmount float64           `json:"total_amount"`
	Type        constants.TxnType `json:"type"`
}

// Enhanced Transaction Analytics Models
type EnhancedTransactionAnalytics struct {
	TransactionsByServiceCategory   []TransactionServiceCategoryStats `json:"transactions_by_service_category"`
	PointsTransferredPerDay         []DailyPointsTransferStats        `json:"points_transferred_per_day"`
	PendingPointsTransactionsPerDay []DailyPendingPointsStats         `json:"pending_points_transactions_per_day"`
}

type TransactionServiceCategoryStats struct {
	ServiceName string  `json:"service_name"`
	TypeName    string  `json:"type_name"`
	Count       int64   `json:"count"`
	TotalAmount float64 `json:"total_amount"`
}

type DailyPointsTransferStats struct {
	Date        string  `json:"date"`
	TotalPoints float64 `json:"total_points"`
	Count       int64   `json:"count"`
}

type DailyPendingPointsStats struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// Survey Analytics Models
type SurveyAnalytics struct {
	SurveysByCategory map[string]SurveyCategoryStats `json:"surveys_by_category"`
}

type SurveyCategoryStats struct {
	Count int64 `json:"count"`
	// AverageRating float64 `json:"average_rating"`
}

// Voucher Redemption Analytics Models - Updated for hierarchical structure
type VoucherRedemptionAnalytics struct {
	SpecialOccasionDiscounts VoucherDiscountTypeStats `json:"special_occasion_discounts"`
	RedeemableDiscounts      VoucherDiscountTypeStats `json:"redeemable_discounts"`
	TotalRedemptions         int64                    `json:"total_redemptions"`
}

// Voucher Usage Analytics Models - Updated for hierarchical structure
type VoucherUsageAnalytics struct {
	SpecialOccasionDiscounts VoucherDiscountTypeStats `json:"special_occasion_discounts"`
	RedeemableDiscounts      VoucherDiscountTypeStats `json:"redeemable_discounts"`
	TotalUsages              int64                    `json:"total_usages"`
}

// Common structure for both redemption and usage analytics
type VoucherDiscountTypeStats struct {
	TotalRedemptions int64                          `json:"total_redemptions"` // For redemption analytics, this is unique users
	TotalUsages      int64                          `json:"total_usages"`      // For usage analytics, this is total transaction count
	ServiceGroups    map[string]VoucherServiceGroup `json:"service_groups"`
}

type VoucherServiceGroup struct {
	TotalRedemptions int64                          `json:"total_redemptions"` // Unique users for this group
	TotalUsages      int64                          `json:"total_usages"`      // Total transactions for this group
	Services         map[string]VoucherServiceStats `json:"services"`
}

type VoucherServiceStats struct {
	Redemptions int64 `json:"redemptions"` // Unique users for this service
	Usages      int64 `json:"usages"`      // Total transactions for this service
}

// Time-Series Voucher Redemption Analytics Models
type TimeSeriesVoucherRedemptionAnalytics struct {
	TimePeriods []TimeSeriesPeriod           `json:"time_periods"`
	Analytics   []VoucherRedemptionAnalytics `json:"analytics"`
}

// Time-Series Voucher Usage Analytics Models
type TimeSeriesVoucherUsageAnalytics struct {
	TimePeriods []TimeSeriesPeriod      `json:"time_periods"`
	Analytics   []VoucherUsageAnalytics `json:"analytics"`
}

// App Sharing Analytics Models
type AppSharingAnalytics struct {
	TotalUsersWhoShared   int64              `json:"total_users_who_shared"`
	TotalInvitationsSent  int64              `json:"total_invitations_sent"`
	TotalInvitationsUsed  int64              `json:"total_invitations_used"`
	ConversionRate        float64            `json:"conversion_rate"`
	SharingActivityPerDay map[string]int64   `json:"sharing_activity_per_day"`
	UsageActivityPerDay   map[string]int64   `json:"usage_activity_per_day"`
	TopSharers            []UserSharingStats `json:"top_sharers"`
}

type UserSharingStats struct {
	UserId          string `json:"user_id"`
	UserName        string `json:"user_name"`
	InvitationsSent int64  `json:"invitations_sent"`
	InvitationsUsed int64  `json:"invitations_used"`
}

// Additional Analytics Models for new endpoints
type DailyPointsStats struct {
	Date        string  `json:"date"`
	TotalPoints float64 `json:"total_points"`
	Count       int64   `json:"count"`
}

type PeriodUserStats struct {
	Period string `json:"period"`
	Count  int64  `json:"count"`
}

// Combined Dashboard Analytics
type DashboardAnalytics struct {
	UserAnalytics              UserAnalytics              `json:"user_analytics"`
	TransactionAnalytics       TransactionAnalytics       `json:"transaction_analytics"`
	SurveyAnalytics            SurveyAnalytics            `json:"survey_analytics"`
	VoucherRedemptionAnalytics VoucherRedemptionAnalytics `json:"voucher_redemption_analytics"`
	VoucherUsageAnalytics      VoucherUsageAnalytics      `json:"voucher_usage_analytics"`
}

// Time-Series Analytics Models
type TimeSeriesEnhancedTransactionAnalytics struct {
	TimePeriods []TimeSeriesPeriod             `json:"time_periods"`
	Analytics   []EnhancedTransactionAnalytics `json:"analytics"`
}

type TimeSeriesTransactionAnalytics struct {
	TimePeriods []TimeSeriesPeriod     `json:"time_periods"`
	Analytics   []TransactionAnalytics `json:"analytics"`
}

type TimeSeriesUserAnalytics struct {
	TimePeriods []TimeSeriesPeriod `json:"time_periods"`
	Analytics   []UserAnalytics    `json:"analytics"`
}

type TimeSeriesSurveyAnalytics struct {
	TimePeriods []TimeSeriesPeriod `json:"time_periods"`
	Analytics   []SurveyAnalytics  `json:"analytics"`
}

type TimeSeriesPeriod struct {
	FromDate    string `json:"from_date"`
	ToDate      string `json:"to_date"`
	PeriodIndex int    `json:"period_index"`
	Label       string `json:"label"`
}

// External Time-Series Analytics Models
type TimeSeriesExternalAnalytics struct {
	TimePeriods []TimeSeriesPeriod  `json:"time_periods"`
	Analytics   []ExternalAnalytics `json:"analytics"`
}

type ExternalAnalytics struct {
	TotalBookings            int `json:"total_bookings"`
	TotalOnlineConsultations int `json:"total_online_consultations"`
	TotalSmileTestDrives     int `json:"total_smile_test_drives"`
}

type TimeSeriesBookingAnalytics struct {
	TimePeriods []TimeSeriesPeriod `json:"time_periods"`
	Analytics   []BookingAnalytics `json:"analytics"`
}

type BookingAnalytics struct {
	TotalBookings int `json:"total_bookings"`
}

type TimeSeriesOnlineConsultationAnalytics struct {
	TimePeriods []TimeSeriesPeriod            `json:"time_periods"`
	Analytics   []OnlineConsultationAnalytics `json:"analytics"`
}

type OnlineConsultationAnalytics struct {
	TotalOnlineConsultations int `json:"total_online_consultations"`
}

type TimeSeriesSmileTestDriveAnalytics struct {
	TimePeriods []TimeSeriesPeriod        `json:"time_periods"`
	Analytics   []SmileTestDriveAnalytics `json:"analytics"`
}

type SmileTestDriveAnalytics struct {
	TotalSmileTestDrives int `json:"total_smile_test_drives"`
}
