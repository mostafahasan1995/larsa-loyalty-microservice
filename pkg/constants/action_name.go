package constants

import "fmt"

type ActionName int

//go:generate go run github.com/dmarkham/enumer -type=ActionName -transform=whitespace

const (
	DeleteService ActionName = iota
	RejectVoucher
	AcceptVoucher

	ChallengeAnswersReject
	ChallengeAnswersAccept
	ChallengeAnswersDelete
	ChallengeAnswersUpdate
	ChallengeAnswersCreate
	ChallengeAnswersRead
	ChallengeAnswersReadAll

	SurveyAnswersCreate
	SurveyAnswersUpdate
	SurveyAnswersArchive
	SurveyAnswersDelete
	SurveyAnswersRead
	SurveyAnswersReadAll
	SurveyAnswersReadUser

	ServiceGroupUpdate
	ServiceGroupDelete
	ServiceGroupCreate

	VisitPackageUpdate
	VisitPackageDelete
	VisitPackageCreate
	VisitPackageRedeem

	LevelUpdate
	LevelDelete
	LevelCreate

	CommunityPostsCreate
	CommunityPostsUpdate
	CommunityPostsDelete

	VouchersReject
	VouchersUse
	VouchersPartnerReject
	VouchersPartnerUse
	VouchersUpdate
	VouchersCreate
	VouchersDelete

	ServicesCreate
	ServicesUpdate
	ServicesDelete
	ServicesBulkDelete
	ServicesRead
	ServicesReadAll
	ServicesRedeem
	ServicesReorder

	TransactionsDelete
	TransactionsBulkDelete
	TransactionsCreate
	TransactionsRead
	TransactionsReadAll
	TransactionsReadUser

	UserWalletBuyPoints
	UserWalletsBulkDelete
	UserWalletsRead
	UserWalletsReadAll
	UserWalletsCreate
	UserWalletsDelete
	UserWalletsUpdateProfile
	UserWalletsVerifyCode
	UserWalletsCreateInvitation
	UserWalletsVisitUser

	QuizzesCreate
	QuizzesUpdate
	QuizzesDelete
	QuizAnswersCreate
	QuizAnswersUpdate
	QuizAnswersDelete
	QuizAnswersRead
	QuizAnswersReadAll

	FeedBackCreate
	FeedBackDelete

	SurveysCreate
	SurveysUpdate
	SurveysDelete

	ChallengeCreate
	ChallengeUpdate
	ChallengeDelete

	SettingCreate
	SettingUpdate
	SettingDelete

	DailyTaskUpdate
	DailyTaskDelete
	DailyTaskCreate

	DashboardNotificationsCreate
	DashboardNotificationsUpdate
	DashboardNotificationsDelete
)

// String returns the string representation of ActionName
func (i ActionName) String() string {
	switch i {
	case DeleteService:
		return "has deleted a service"
	case RejectVoucher:
		return "has rejected a voucher"
	case AcceptVoucher:
		return "has accepted a voucher"
	case ChallengeAnswersReject:
		return "has rejected challenge answer"
	case ChallengeAnswersAccept:
		return "has accepted challenge answer"
	case ChallengeAnswersDelete:
		return "has deleted challenge answer"
	case ChallengeAnswersUpdate:
		return "has updated challenge answer"
	case ChallengeAnswersCreate:
		return "has submitted challenge answer"
	case ChallengeAnswersRead:
		return "has viewed challenge answer"
	case ChallengeAnswersReadAll:
		return "has visited challenge answers page"
	case SurveyAnswersCreate:
		return "has submitted survey answer"
	case SurveyAnswersUpdate:
		return "has updated survey answer"
	case SurveyAnswersArchive:
		return "has archived survey answer"
	case SurveyAnswersDelete:
		return "has deleted survey answer"
	case SurveyAnswersRead:
		return "has viewed survey answer"
	case SurveyAnswersReadAll:
		return "has visited survey answers page"
	case SurveyAnswersReadUser:
		return "has viewed user survey answers"
	case ServiceGroupUpdate:
		return "has updated service group"
	case ServiceGroupDelete:
		return "has deleted service group"
	case ServiceGroupCreate:
		return "has created service group"
	case VisitPackageUpdate:
		return "has updated visit package"
	case VisitPackageDelete:
		return "has deleted visit package"
	case VisitPackageCreate:
		return "has created visit package"
	case VisitPackageRedeem:
		return "has redeemed visit package"
	case LevelUpdate:
		return "has updated level"
	case LevelDelete:
		return "has deleted level"
	case LevelCreate:
		return "has created level"
	case CommunityPostsCreate:
		return "has created community post"
	case CommunityPostsUpdate:
		return "has updated community post"
	case CommunityPostsDelete:
		return "has deleted community post"
	case VouchersReject:
		return "has rejected voucher"
	case VouchersUse:
		return "has used voucher"
	case VouchersPartnerReject:
		return "partner has rejected voucher"
	case VouchersPartnerUse:
		return "partner has used voucher"
	case VouchersUpdate:
		return "has updated voucher"
	case VouchersCreate:
		return "has created voucher"
	case VouchersDelete:
		return "has deleted voucher"
	case ServicesCreate:
		return "has created service"
	case ServicesUpdate:
		return "has updated service"
	case ServicesDelete:
		return "has deleted service"
	case ServicesBulkDelete:
		return "has bulk deleted services"
	case ServicesRead:
		return "has viewed service"
	case ServicesReadAll:
		return "has browsed available services"
	case ServicesRedeem:
		return "has asked for redeem"
	case ServicesReorder:
		return "has reordered services"
	case TransactionsDelete:
		return "has deleted transaction"
	case TransactionsBulkDelete:
		return "has bulk deleted transactions"
	case TransactionsCreate:
		return "has created transaction"
	case TransactionsRead:
		return "has viewed transaction"
	case TransactionsReadAll:
		return "has viewed transaction history"
	case TransactionsReadUser:
		return "has viewed user transactions"
	case UserWalletBuyPoints:
		return "has purchased loyalty points"
	case UserWalletsBulkDelete:
		return "has bulk deleted user wallets"
	case UserWalletsRead:
		return "has checked wallet balance"
	case UserWalletsReadAll:
		return "has viewed all user wallets"
	case UserWalletsCreate:
		return "has created user wallet"
	case UserWalletsDelete:
		return "has deleted user wallet"
	case UserWalletsUpdateProfile:
		return "has updated profile"
	case UserWalletsVerifyCode:
		return "has verified account code"
	case UserWalletsCreateInvitation:
		return "has created invitation"
	case UserWalletsVisitUser:
		return "has visited user profile"
	case QuizzesCreate:
		return "has created quiz"
	case QuizzesUpdate:
		return "has updated quiz"
	case QuizzesDelete:
		return "has deleted quiz"
	case QuizAnswersCreate:
		return "has submitted quiz answer"
	case QuizAnswersUpdate:
		return "has updated quiz answer"
	case QuizAnswersDelete:
		return "has deleted quiz answer"
	case QuizAnswersRead:
		return "has viewed quiz answer"
	case QuizAnswersReadAll:
		return "has visited quiz answers page"
	case FeedBackCreate:
		return "has submitted feedback"
	case FeedBackDelete:
		return "has deleted feedback"
	case SurveysCreate:
		return "has created survey"
	case SurveysUpdate:
		return "has updated survey"
	case SurveysDelete:
		return "has deleted survey"
	case ChallengeCreate:
		return "has created challenge"
	case ChallengeUpdate:
		return "has updated challenge"
	case ChallengeDelete:
		return "has deleted challenge"
	case SettingCreate:
		return "has created setting"
	case SettingUpdate:
		return "has updated setting"
	case SettingDelete:
		return "has deleted setting"
	case DailyTaskUpdate:
		return "has updated daily task"
	case DailyTaskDelete:
		return "has deleted daily task"
	case DailyTaskCreate:
		return "has created daily task"
	case DashboardNotificationsCreate:
		return "has created dashboard notification"
	case DashboardNotificationsUpdate:
		return "has updated dashboard notification"
	case DashboardNotificationsDelete:
		return "has deleted dashboard notification"
	default:
		return fmt.Sprintf("ActionName(%d)", i)
	}
}
