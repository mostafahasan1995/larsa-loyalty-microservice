package constants

type CacheKey int

const (
	UNDEFINED CacheKey = iota
	SERVICE
	TRANSACTION
	UserWallet
	QUIZ
	FEEDBACK
	SURVEY
	CAMPAIGN
	DISCOUNT
	LEVEL
	SETTINGS
	DISCOUNT_GROUP
	SERVICE_GROUP
	QUIZ_ANSWER
	SURVEY_ANSWER
	CHALLENGE
	CHALLENGE_ANSWER
	DAILY_TASK
	DAILY_TASK_INTERACTION
	VOUCHER
	COMMUNITY_POST
	VISIT_PACKAGE
	DASHBOARD_NOTIFICATIONS
	USERS_WITH_PENDING_POINTS
	USERS_WITH_PENDING_POINTS_ALL
)

func (i CacheKey) String() string {
	switch i {
	case SERVICE:
		return "service"
	case TRANSACTION:
		return "transaction"
	case UserWallet:
		return "userwallet"
	case QUIZ:
		return "quiz"
	case QUIZ_ANSWER:
		return "quiz_answer"
	case FEEDBACK:
		return "feedback"
	case SURVEY:
		return "survey"
	case SURVEY_ANSWER:
		return "survey_answer"
	case CAMPAIGN:
		return "campaign"
	case DISCOUNT:
		return "discount"
	case LEVEL:
		return "level"
	case SETTINGS:
		return "settings"
	case DISCOUNT_GROUP:
		return "discount_group"
	case SERVICE_GROUP:
		return "service_group"
	case CHALLENGE:
		return "challenge"
	case CHALLENGE_ANSWER:
		return "challenge_answer"
	case DAILY_TASK:
		return "daily_task"
	case DAILY_TASK_INTERACTION:
		return "daily_task_interaction"
	case VOUCHER:
		return "voucher"
	case COMMUNITY_POST:
		return "community_post"
	case VISIT_PACKAGE:
		return "visit_package"
	case DASHBOARD_NOTIFICATIONS:
		return "dashboard_notifications"
	case USERS_WITH_PENDING_POINTS:
		return "users_with_pending_points"
	case USERS_WITH_PENDING_POINTS_ALL:
		return "users_with_pending_points_all"
	default:
		return ""
	}
}
