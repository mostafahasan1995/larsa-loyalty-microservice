package constants

type ScoreUpdateType int

const (
	SHARE_APP_LINK ScoreUpdateType = iota + 1
	SOCIAL_MEDIA_SHARE
	GOOGLE_MAPS_REVIEW
	FACEBOOK_FOLLOW
	FACEBOOK_REVIEW
	INSTAGRAM_FOLLOW
	QUIZ_ANSWERED
	SUGGESTION_SUBMITION
	NEW_ACHIEVEMENT
	USER_REGISTER
	CLINIC_VISIT
)

func (s ScoreUpdateType) String() string {
	switch s {
	case SHARE_APP_LINK:
		return "share_app_link"
	case SOCIAL_MEDIA_SHARE:
		return "social_media_share"
	case GOOGLE_MAPS_REVIEW:
		return "google_maps_review"
	case FACEBOOK_FOLLOW:
		return "facebook_follow"
	case FACEBOOK_REVIEW:
		return "facebook_review"
	case INSTAGRAM_FOLLOW:
		return "instagram_follow"
	case QUIZ_ANSWERED:
		return "quiz_answer"
	case SUGGESTION_SUBMITION:
		return "suggestion_submition"
	case NEW_ACHIEVEMENT:
		return "new_achievement"
	case USER_REGISTER:
		return "customer_register"
	case CLINIC_VISIT:
		return "clinic_visit"
	}

	return "invalid_option"
}
