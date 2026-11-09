package constants

type TxnType int

const (
	TXN_CAMPAIGN_INTERACT TxnType = iota
	TXN_DISCOUNT_APPLY
	TXN_FEEDBACK_SUBMIT
	TXN_LEVEL_INCREASED
	TXN_QUIZ_SUBMIT
	TXN_SURVEY_SUBMIT
	TXN_WALLET_CREATED
	TXN_POINTS_ADDED_TO_WALLET
	TXN_CHALLENGE_SUBMIT
	TXN_CHALLENGE_ANSWER_ACCEPTED
	TXN_SERVICE_REDEEMED
	TXN_VOUCHER_USED
	TXN_POST_ADDED
	TXN_DAILY_TASK_INTERACTED
	TXN_VISIT_PACKAGE_REDEEMED
	TXN_UPDATE_PROFILE
	TXN_INVITE_BY_INVITATION
	TXN_USER_JOIN_BY_YOUR_INVITATION
	TXN_VOUCHER_EXPIRED
	TXN_APP_SHARED
)

func (t TxnType) String() string {
	switch t {
	case TXN_FEEDBACK_SUBMIT:
		return "feedback_submit"
	case TXN_QUIZ_SUBMIT:
		return "quiz_submit"
	case TXN_SURVEY_SUBMIT:
		return "survey_submit"
	case TXN_WALLET_CREATED:
		return "wallet_created"
	case TXN_POINTS_ADDED_TO_WALLET:
		return "points_added_to_wallet"
	case TXN_CHALLENGE_SUBMIT:
		return "challenge_submit"
	case TXN_CHALLENGE_ANSWER_ACCEPTED:
		return "challenge_answer_accepted"
	case TXN_SERVICE_REDEEMED:
		return "service_redeemed"
	case TXN_VOUCHER_USED:
		return "voucher_used"
	case TXN_VOUCHER_EXPIRED:
		return "voucher_expired"
	case TXN_POST_ADDED:
		return "post_added"
	case TXN_DAILY_TASK_INTERACTED:
		return "daily_task_interacted"
	case TXN_VISIT_PACKAGE_REDEEMED:
		return "visit_package_redeemed"
	case TXN_UPDATE_PROFILE:
		return "profile_updated"
	case TXN_INVITE_BY_INVITATION:
		return "INVITE_BY_INVITATION"
	case TXN_USER_JOIN_BY_YOUR_INVITATION:
		return "USER_JOIN_BY_YOUR_INVITATION"
	case TXN_APP_SHARED:
		return "app_shared"
	default:
		return ""
	}
}
