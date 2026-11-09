package models

import (
	"time"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DashboardNotification struct {
	Id          primitive.ObjectID     `bson:"_id,omitempty" json:"_id,omitempty"`
	Title       LocalizableString      `bson:"title" json:"title"`
	Message     LocalizableString      `bson:"message" json:"message"`
	ActionType  string                 `bson:"action_type" json:"action_type"`
	EntityId    primitive.ObjectID     `bson:"entity_id" json:"entity_id"`
	EntityType  string                 `bson:"entity_type" json:"entity_type"`
	UserId      primitive.ObjectID     `bson:"user_id" json:"user_id"`
	UserName    string                 `bson:"user_name" json:"user_name"`
	Priority    string                 `bson:"priority" json:"priority"`
	Status      string                 `bson:"status" json:"status"`
	TargetRole  string                 `bson:"target_role" json:"target_role"`
	Metadata    map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	ReadAt      *time.Time             `bson:"read_at,omitempty" json:"read_at,omitempty"`
	ResolvedAt  *time.Time             `bson:"resolved_at,omitempty" json:"resolved_at,omitempty"`
	BaseDbModel `bson:",inline" json:",inline"`
}

type DashboardNotificationResponse struct {
	DashboardNotification `bson:",inline" json:",inline"`
	User                  *common.User `bson:"user,omitempty" json:"user,omitempty"`
	Entity                interface{}  `bson:"entity,omitempty" json:"entity,omitempty"`
}

type DashboardNotificationWithPagination struct {
	Notifications []DashboardNotificationResponse `bson:"notifications" json:"notifications"`
	Pagination    Pagination                      `bson:"pagination" json:"pagination"`
}

type DashboardNotificationStats struct {
	TotalPending   int64            `json:"total_pending"`
	HighPriority   int64            `json:"high_priority"`
	MediumPriority int64            `json:"medium_priority"`
	LowPriority    int64            `json:"low_priority"`
	ByType         map[string]int64 `json:"by_type"`
}

const (
	DashboardNotificationStatusPending      = "pending"
	DashboardNotificationStatusAcknowledged = "acknowledged"
	DashboardNotificationStatusResolved     = "resolved"

	DashboardNotificationPriorityHigh   = "high"
	DashboardNotificationPriorityMedium = "medium"
	DashboardNotificationPriorityLow    = "low"

	DashboardNotificationTargetAdmin        = "admin"
	DashboardNotificationTargetReceptionist = "receptionist"
	DashboardNotificationTargetBoth         = "both"

	DashboardNotificationActionBooking             = "booking"
	DashboardNotificationActionConsultation        = "consultation"
	DashboardNotificationActionSmileTestDrive      = "smile_test_drive"
	DashboardNotificationActionSocialMoments       = "social_moments"
	DashboardNotificationActionFeedback            = "feedback"
	DashboardNotificationActionSurveyAnswer        = "survey_answer"
	DashboardNotificationActionVoucherGoogle       = "voucher_google_review"
	DashboardNotificationActionVoucherFacebook     = "voucher_facebook_follow"
	DashboardNotificationActionVoucherInstagram    = "voucher_instagram_follow"
	DashboardNotificationActionPointsAllocation    = "points_allocation"
	DashboardNotificationActionChallengeSubmission = "challenge_submission"
	DashboardNotificationActionVoucherRedemption   = "voucher_redemption"

	DashboardNotificationEntityFeedback        = "feedback"
	DashboardNotificationEntitySurveyAnswer    = "survey_answer"
	DashboardNotificationEntityVoucher         = "voucher"
	DashboardNotificationEntityBooking         = "booking"
	DashboardNotificationEntityConsultation    = "consultation"
	DashboardNotificationEntityCommunityPost   = "community_post"
	DashboardNotificationEntityChallenge       = "challenge"
	DashboardNotificationEntityChallengeAnswer = "challenge_answer"
)
