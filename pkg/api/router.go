package api

import (
	"loyalty/pkg/api/analytics"
	challengeAnswers "loyalty/pkg/api/challenge_answers"
	"loyalty/pkg/api/challenges"
	communityPosts "loyalty/pkg/api/community_posts"
	dailyTasks "loyalty/pkg/api/daily_tasks"
	dashboardNotifications "loyalty/pkg/api/dashboard_notifications"
	dbManagement "loyalty/pkg/api/db_management"
	"loyalty/pkg/api/feedback"
	"loyalty/pkg/api/levels"
	quizAnswers "loyalty/pkg/api/quiz_answers"
	"loyalty/pkg/api/quizzes"
	Service_group "loyalty/pkg/api/service_group"
	"loyalty/pkg/api/services"
	"loyalty/pkg/api/settings"
	surveyAnswers "loyalty/pkg/api/survey_answers"
	"loyalty/pkg/api/surveys"
	"loyalty/pkg/api/transactions"
	"loyalty/pkg/api/users"
	"loyalty/pkg/api/userwallets"
	visitpackage "loyalty/pkg/api/visit_package"
	"loyalty/pkg/api/vouchers"
	"loyalty/pkg/constants"
	middlewares "loyalty/pkg/middleware"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

func InitRouter() *chi.Mux {
	initCapabilities()

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "x-client", "x-access-token", "x-service", "x-expire", "x-service-token", "x-user-id"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Use(common.ClientSubscriptionInstance.CheckSubscription)
	r.Use(middlewares.AddLangToContext)
	r.Use(middleware.SetHeader("Content-Type", "application/json; charset=utf-8"))

	//services
	r.Route("/services", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ServicesRead, "loyaltyServices")).Get("/{id}", services.GetServiceById)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ServicesReadAll, "loyaltyServices")).Get("/", services.GetServices)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ServicesReadAll, "loyaltyServices")).Get("/all", services.GetAllServices)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.ServicesCreate, "loyaltyServices")).Post("/", services.CreateService)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ServicesUpdate, "loyaltyServices")).Put("/{id}", services.UpdateService)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ServicesReorder, "loyaltyServices")).Patch("/{serviceid}/reorder", services.Reorder)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ServicesDelete, "loyaltyServices")).Delete("/{id}", services.DeleteService)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ServicesBulkDelete, "loyaltyServices")).Delete("/bulk", services.BulkDeleteServices)

		//redeem
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ServicesRedeem, "loyaltyServices")).Post("/redeem", services.RedeemService)
	})

	//transactions
	r.Route("/transactions", func(r chi.Router) {

		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.TransactionsRead, "loyaltyTransactions")).Get("/{id}", transactions.GetTransactionById)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.TransactionsReadAll, "loyaltyTransactions")).Get("/", transactions.GetTransactions)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.TransactionsReadAll, "loyaltyTransactions")).Get("/all", transactions.GetAllTransactions)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.TransactionsCreate, "loyaltyTransactions")).Post("/", transactions.CreateTransaction)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.TransactionsDelete, "loyaltyTransactions")).Delete("/{id}", transactions.DeleteTransaction)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.TransactionsBulkDelete, "loyaltyTransactions")).Delete("/bulk", transactions.BulkDeleteTransactions)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.TransactionsReadUser, "loyaltyTransactions")).Get("/me", transactions.GetUserTransactions)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.TransactionsReadUser, "loyaltyTransactions")).Get("/user/{id}", transactions.GetByUserId)
	})

	//users wallets
	r.Route("/wallets", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.UserWalletsRead, "loyaltyUserWallets")).Get("/{id}", userwallets.GetWalletById)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.UserWalletsReadAll, "loyaltyUserWallets")).Get("/", userwallets.GetUsersWallets)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.UserWalletsReadAll, "loyaltyUserWallets")).Get("/all", userwallets.GetAllUsersWallets)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.UserWalletsCreate, "loyaltyUserWallets")).Post("/", userwallets.CreateUserWallet)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.UserWalletsDelete, "loyaltyUserWallets")).Delete("/{id}", userwallets.DeleteUserWallet)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.UserWalletsBulkDelete, "loyaltyUserWallets")).Delete("/bulk", userwallets.BulkDeleteUserWallets)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.UserWalletsRead, "loyaltyUserWallets")).Get("/me", userwallets.GetUserWallet)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.UserWalletsRead, "loyaltyUserWallets")).Get("/user/{id}", userwallets.GetUserWalletById)
		r.With(middlewares.Auth("authenticate")).Get("/points-aed", userwallets.GetPointsPerAED)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.UserWalletBuyPoints, "loyaltyUserWallets")).Put("/{id}/buy-points", userwallets.BuyPoints)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.UserWalletsUpdateProfile, "loyaltyUserWallets")).Post("/update-profile", userwallets.UpdateProfile)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.UserWalletsVerifyCode, "loyaltyUserWallets")).Post("/verify-code", userwallets.VerifyCode)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.UserWalletsCreateInvitation, "loyaltyUserWallets")).Post("/create-invitation", userwallets.CreateInvitation)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.UserWalletsVisitUser, "loyaltyUserWallets")).Put("/user/{id}/visit", userwallets.VisitUser)
	})

	//users
	r.Route("/users", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/", users.GetUsersWithPendingPoints)
		r.With(middlewares.Auth("authenticate")).Get("/all", users.GetAllUsersWithPendingPoints)
		r.With(middlewares.Auth("authenticate")).Delete("/cache", users.ClearUsersCache)
	})

	//quizzes
	r.Route("/quizzes", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/{id}", quizzes.GetQuizById)
		r.With(middlewares.Auth("authenticate")).Get("/", quizzes.GetQuizzes)
		r.With(middlewares.Auth("authenticate")).Get("/all", quizzes.GetAllQuizzes)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.QuizzesCreate, "loyaltyQuizzes")).Post("/", quizzes.CreateQuiz)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.QuizzesDelete, "loyaltyQuizzes")).Delete("/{id}", quizzes.DeleteQuiz)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.QuizzesUpdate, "loyaltyQuizzes")).Put("/{id}", quizzes.UpdateQuiz)
		r.With(middlewares.Auth("authenticate")).Post("/submit", quizzes.SubmitQuiz)
		r.With(middlewares.Auth("authenticate")).Get("/me", quizzes.GetUserAvailableQuizzes)
		r.With(middlewares.Auth("authenticate")).Get("/me/answered", quizzes.GetUserQuizzes)
		r.With(middlewares.Auth("authenticate")).Get("/user/{id}", quizzes.GetUserQuizzesById)
	})

	//feedback
	r.Route("/feedback", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/{id}", feedback.GetFeedbackById)
		r.With(middlewares.Auth("authenticate")).Get("/", feedback.GetFeedback)
		r.With(middlewares.Auth("authenticate")).Get("/all", feedback.GetAllFeedback)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.FeedBackCreate, "loyaltyFeedbacks")).Post("/submit", feedback.SubmitFeedback)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.FeedBackDelete, "loyaltyFeedbacks")).Delete("/{id}", feedback.DeleteFeedback)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.FeedBackDelete, "loyaltyFeedbacks")).Delete("/bulk", feedback.BulkDeleteFeedback)
	})

	//surveys
	r.Route("/surveys", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/{id}", surveys.GetSurveyById)
		r.With(middlewares.Auth("authenticate")).Get("/", surveys.GetSurvey)
		r.With(middlewares.Auth("authenticate")).Get("/all", surveys.GetAllSurvey)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.SurveysCreate, "loyaltySurveys")).Post("/", surveys.CreateSurvey)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.SurveysUpdate, "loyaltySurveys")).Put("/{id}", surveys.UpdateSurvey)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.SurveysDelete, "loyaltySurveys")).Delete("/{id}", surveys.DeleteSurvey)
		r.With(middlewares.Auth("authenticate")).Post("/submit", surveys.SubmitSurvey)
		r.With(middlewares.Auth("authenticate")).Get("/me", surveys.GetUserAvailableSurveys)
		r.With(middlewares.Auth("authenticate")).Get("/me/answered", surveys.GetUserSurveys)
		r.With(middlewares.Auth("authenticate")).Get("/user/{id}", surveys.GetUserSurveysById)
	})

	//challenges
	r.Route("/challenges", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/{id}", challenges.GetChallengeById)
		r.With(middlewares.Auth("authenticate")).Get("/", challenges.GetChallenge)
		r.With(middlewares.Auth("authenticate")).Get("/all", challenges.GetAllChallenge)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.ChallengeCreate, "loyaltyChallenges")).Post("/", challenges.CreateChallenge)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ChallengeDelete, "loyaltyChallenges")).Delete("/{id}", challenges.DeleteChallenge)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ChallengeUpdate, "loyaltyChallenges")).Put("/{id}", challenges.UpdateChallenge)
		r.With(middlewares.Auth("authenticate")).Post("/submit", challenges.SubmitChallenge)
		r.With(middlewares.Auth("authenticate")).Get("/me", challenges.GetUserAvailableChallenges)
		r.With(middlewares.Auth("authenticate")).Get("/me/answered", challenges.GetUserChallenges)
		r.With(middlewares.Auth("authenticate")).Get("/user/{id}", challenges.GetUserChallengesById)
	})

	//vouchers
	r.Route("/vouchers", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/user/{userId}/{code}", vouchers.GetUserVoucherByCode)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.VouchersUse, "loyaltyVouchers", "code")).Put("/user/{userId}/{code}", vouchers.UseVoucher)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.VouchersReject, "loyaltyVouchers", "code")).Delete("/user/{userId}/{code}", vouchers.RejectVoucher)
		r.With(middlewares.Auth("authenticate")).Get("/user/{userId}", vouchers.GetUserVouchersById)
		r.With(middlewares.Auth("authenticate")).Get("/partner/me/{code}", vouchers.PartnerGetVoucherByCode)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.VouchersPartnerUse, "loyaltyVouchers", "code")).Put("/partner/me/{code}", vouchers.PartnerUseVoucher)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.VouchersPartnerReject, "loyaltyVouchers", "code")).Delete("/partner/me/{code}", vouchers.PartnerRejectVoucher)
		r.With(middlewares.Auth("authenticate")).Get("/partner/me", vouchers.PartnerGetVouchers)
		r.With(middlewares.Auth("authenticate")).Get("/code/{code}", vouchers.GetVoucherByCode)
		r.With(middlewares.Auth("authenticate")).Get("/me", vouchers.GetUserVouchers)
		r.With(middlewares.Auth("authenticate")).Get("/{id}", vouchers.GetVoucherById)
		r.With(middlewares.Auth("authenticate")).Delete("/{id}", vouchers.DeleteVoucher)
		r.With(middlewares.Auth("authenticate")).Put("/{id}", vouchers.UpdateVoucher)
		r.With(middlewares.Auth("authenticate")).Get("/all", vouchers.GetAllVoucher)
		r.With(middlewares.Auth("authenticate")).Post("/", vouchers.CreateVoucher)
		r.With(middlewares.Auth("authenticate")).Get("/", vouchers.GetVoucher)
		r.With(middlewares.Auth("authenticate")).Get("/management", vouchers.GetVouchersForManagement)
		r.With(middlewares.Auth("authenticate")).Post("/management", vouchers.PostVouchersForManagement)
		r.With(middlewares.Auth("authenticate")).Post("/management/v2", vouchers.GetVouchersForManagementV2)
		r.With(middlewares.Auth("authenticate")).Get("/management/all", vouchers.GetAllVouchersForManagement)
		r.With(middlewares.Auth("authenticate")).Put("/{id}/status", vouchers.UpdateVoucherStatus)
		r.With(middlewares.Auth("authenticate")).Get("/management/status-options", vouchers.GetVoucherStatusOptions)
		r.With(middlewares.Auth("authenticate")).Get("/management/stats", vouchers.GetVoucherStats)
		r.With(middlewares.Auth("authenticate")).Post("/filter", vouchers.FilterVouchers)
		r.With(middlewares.Auth("authenticate")).Post("/fix-status-data", vouchers.FixVoucherStatusData)
	})

	//communityPosts
	r.Route("/community-posts", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/{id}", communityPosts.GetCommunityPostById)
		r.With(middlewares.Auth("authenticate")).Get("/", communityPosts.GetCommunityPost)
		r.With(middlewares.Auth("authenticate")).Get("/all", communityPosts.GetAllCommunityPost)

		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.CommunityPostsCreate, "loyaltyCommunityPosts")).Post("/", communityPosts.CreateCommunityPost)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.CommunityPostsDelete, "loyaltyCommunityPosts")).Delete("/{id}", communityPosts.DeleteCommunityPost)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.CommunityPostsUpdate, "loyaltyCommunityPosts")).Put("/{id}", communityPosts.UpdateCommunityPost)
		r.With(middlewares.Auth("authenticate")).Get("/me", communityPosts.GetUserCommunityPosts)
		r.With(middlewares.Auth("authenticate")).Get("/user/{id}", communityPosts.GetUserCommunityPostsById)
		r.With(middlewares.Auth("authenticate")).Put("/react/{id}", communityPosts.React)
		r.With(middlewares.Auth("authenticate")).Put("/un-react/{id}", communityPosts.UnReact)

	})

	//levels
	r.Route("/levels", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/{id}", levels.GetLevelById)
		r.With(middlewares.Auth("authenticate")).Get("/", levels.GetLevels)
		r.With(middlewares.Auth("authenticate")).Get("/all", levels.GetAllLevels)

		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.LevelCreate, "loyaltyLevels")).Post("/", levels.CreateLevel)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.LevelUpdate, "loyaltyLevels")).Put("/{id}", levels.UpdateLevel)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.LevelDelete, "loyaltyLevels")).Delete("/{id}", levels.DeleteLevel)
	})

	//visit packages
	r.Route("/visit-packages", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/{id}", visitpackage.GetVisitPackageById)
		r.With(middlewares.Auth("authenticate")).Get("/", visitpackage.GetVisitPackages)
		r.With(middlewares.Auth("authenticate")).Get("/all", visitpackage.GetAllVisitPackages)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.VisitPackageCreate, "loyaltyVisitPackages")).Post("/", visitpackage.CreateVisitPackage)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.VisitPackageUpdate, "loyaltyVisitPackages")).Put("/{id}", visitpackage.UpdateVisitPackage)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.VisitPackageDelete, "loyaltyVisitPackages")).Delete("/{id}", visitpackage.DeleteVisitPackage)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.VisitPackageRedeem, "loyaltyVisitPackages")).Post("/{id}/user/{userId}/redeem", visitpackage.RedeemPkg)
	})

	//settings
	r.Route("/settings", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/{id}", settings.GetSettingsById)
		r.With(middlewares.Auth("authenticate")).Get("/", settings.GetSettings)
		r.With(middlewares.Auth("authenticate")).Get("/all", settings.GetAllSettings)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.SettingCreate, "loyaltySettings")).Post("/", settings.CreateSettings)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.SettingUpdate, "loyaltySettings")).Put("/{id}", settings.UpdateSettings)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.SettingDelete, "loyaltySettings")).Delete("/{id}", settings.DeleteSettings)
	})

	//service groups
	r.Route("/service-group", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/{id}", Service_group.GetServiceGroupById)
		r.With(middlewares.Auth("authenticate")).Get("/", Service_group.GetServiceGroups)
		r.With(middlewares.Auth("authenticate")).Get("/all", Service_group.GetAllServiceGroups)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.ServiceGroupCreate, "loyaltyServiceGroups")).Post("/", Service_group.CreateServiceGroup)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ServiceGroupUpdate, "loyaltyServiceGroups")).Put("/{id}", Service_group.UpdateServiceGroup)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ServiceGroupDelete, "loyaltyServiceGroups")).Delete("/{id}", Service_group.DeleteServiceGroup)
	})

	//quiz_answers
	r.Route("/quiz-answers", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.QuizAnswersRead, "loyaltyQuizAnswers")).Get("/{id}", quizAnswers.GetQuizAnswerById)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.QuizAnswersReadAll, "loyaltyQuizAnswers")).Get("/", quizAnswers.GetQuizAnswers)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.QuizAnswersReadAll, "loyaltyQuizAnswers")).Get("/all", quizAnswers.GetAllQuizAnswers)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.QuizAnswersCreate, "loyaltyQuizAnswers")).Post("/", quizAnswers.CreateQuizAnswer)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.QuizAnswersUpdate, "loyaltyQuizAnswers")).Put("/{id}", quizAnswers.UpdateQuizAnswer)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.QuizAnswersDelete, "loyaltyQuizAnswers")).Delete("/{id}", quizAnswers.DeleteQuizAnswer)
	})

	//survey_answers
	r.Route("/survey-answers", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.SurveyAnswersRead, "loyaltySurveyAnswers")).Get("/{id}", surveyAnswers.GetSurveyAnswerById)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.SurveyAnswersReadAll, "loyaltySurveyAnswers")).Get("/", surveyAnswers.GetSurveyAnswers)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.SurveyAnswersReadAll, "loyaltySurveyAnswers")).Get("/all", surveyAnswers.GetAllSurveyAnswers)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.SurveyAnswersCreate, "loyaltySurveyAnswers")).Post("/", surveyAnswers.CreateSurveyAnswer)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.SurveyAnswersUpdate, "loyaltySurveyAnswers")).Put("/{id}", surveyAnswers.UpdateSurveyAnswer)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.SurveyAnswersReadUser, "loyaltySurveyAnswers")).Get("/user/{id}", surveyAnswers.GetUserSurveyAnswersById)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.SurveyAnswersDelete, "loyaltySurveyAnswers")).Delete("/{id}", surveyAnswers.DeleteSurveyAnswer)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.SurveyAnswersDelete, "loyaltySurveyAnswers")).Delete("/bulk", surveyAnswers.BulkDeleteSurveyAnswers)
	})

	//challenge_answers
	r.Route("/challenge-answers", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ChallengeAnswersRead, "loyaltyChallengeAnswers")).Get("/{id}", challengeAnswers.GetChallengeAnswerById)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ChallengeAnswersReadAll, "loyaltyChallengeAnswers")).Get("/", challengeAnswers.GetChallengeAnswers)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ChallengeAnswersReadAll, "loyaltyChallengeAnswers")).Get("/all", challengeAnswers.GetAllChallengeAnswers)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.ChallengeAnswersCreate, "loyaltyChallengeAnswers")).Post("/", challengeAnswers.CreateChallengeAnswer)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ChallengeAnswersUpdate, "loyaltyChallengeAnswers")).Put("/{id}", challengeAnswers.UpdateChallengeAnswer)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ChallengeAnswersDelete, "loyaltyChallengeAnswers")).Delete("/{id}", challengeAnswers.DeleteChallengeAnswer)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ChallengeAnswersAccept, "loyaltyChallengeAnswers")).Put("/accept/{id}", challengeAnswers.AcceptChallengeAnswer)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.ChallengeAnswersReject, "loyaltyChallengeAnswers")).Put("/reject/{id}", challengeAnswers.RejectChallengeAnswer)
	})

	//daily-tasks
	r.Route("/daily-tasks", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/{id}", dailyTasks.GetDailyTaskById)
		r.With(middlewares.Auth("authenticate")).Get("/", dailyTasks.GetDailyTask)
		r.With(middlewares.Auth("authenticate")).Get("/monthly", dailyTasks.GetMonthlyTasksData)
		r.With(middlewares.Auth("authenticate")).Get("/yearly", dailyTasks.GetYearlyTasksData)
		r.With(middlewares.Auth("authenticate")).Get("/monthly/user/{userId}", dailyTasks.GetMonthlyTasksDataByUserId)
		r.With(middlewares.Auth("authenticate")).Get("/yearly/user/{userId}", dailyTasks.GetYearlyTasksData)
		r.With(middlewares.Auth("authenticate")).Get("/all", dailyTasks.GetAllDailyTask)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.DailyTaskCreate, "loyaltyDailyTasks")).Post("/", dailyTasks.CreateDailyTask)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.DailyTaskUpdate, "loyaltyDailyTasks")).Put("/{id}", dailyTasks.UpdateDailyTask)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.DailyTaskDelete, "loyaltyDailyTasks")).Delete("/{id}", dailyTasks.DeleteDailyTask)
		r.With(middlewares.Auth("authenticate")).Post("/interact", dailyTasks.InteractDailyTask)
		r.With(middlewares.Auth("authenticate")).Get("/me", dailyTasks.GetUserDailyTasks)
		r.With(middlewares.Auth("authenticate")).Get("/user/{id}", dailyTasks.GetUserDailyTasksById)
	})

	//db-management
	r.Route("/db-management", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Post("/apply-operator", dbManagement.ApplyOperator)
	})

	//dashboard-notifications
	r.Route("/dashboard-notifications", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/{id}", dashboardNotifications.GetDashboardNotificationById)
		r.With(middlewares.Auth("authenticate")).Get("/", dashboardNotifications.GetDashboardNotifications)
		r.With(middlewares.Auth("authenticate")).Get("/pending", dashboardNotifications.GetPendingDashboardNotifications)
		r.With(middlewares.Auth("authenticate")).Get("/stats", dashboardNotifications.GetDashboardNotificationStats)
		r.With(middlewares.Auth("authenticate")).Get("/unread-count", dashboardNotifications.GetUnreadCount)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.DashboardNotificationsCreate, "loyaltyDashboardNotifications")).Post("/", dashboardNotifications.CreateDashboardNotification)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.DashboardNotificationsCreate, "loyaltyDashboardNotifications")).Post("/simple", dashboardNotifications.CreateDashboardNotificationSimple)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.DashboardNotificationsCreate, "loyaltyDashboardNotifications")).Post("/smiletestdrive/{entity_id}", dashboardNotifications.CreateSmileTestDriveNotification)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.DashboardNotificationsCreate, "loyaltyDashboardNotifications")).Post("/booking/{entity_id}", dashboardNotifications.CreateBookingNotification)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.DashboardNotificationsCreate, "loyaltyDashboardNotifications")).Post("/onlineconsultation/{entity_id}", dashboardNotifications.CreateOnlineConsultationNotification)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.DashboardNotificationsCreate, "loyaltyDashboardNotifications")).Post("/socialmoment/{entity_id}", dashboardNotifications.CreateSocialMomentNotification)
		r.With(middlewares.Auth("authenticate"),
			middlewares.AuditForCreate(constants.DashboardNotificationsCreate, "loyaltyDashboardNotifications")).Post("/voucher/{entity_id}", dashboardNotifications.CreateVoucherRedemptionNotification)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.DashboardNotificationsUpdate, "loyaltyDashboardNotifications")).Put("/{id}", dashboardNotifications.UpdateDashboardNotification)
		r.With(middlewares.Auth("authenticate")).Patch("/{id}/acknowledge", dashboardNotifications.MarkAsAcknowledged)
		r.With(middlewares.Auth("authenticate")).Patch("/{id}/resolve", dashboardNotifications.MarkAsResolved)
		r.With(middlewares.Auth("authenticate")).Patch("/bulk-read", dashboardNotifications.BulkMarkAsRead)
		r.With(middlewares.Auth("authenticate"),
			middlewares.Audit(constants.DashboardNotificationsDelete, "loyaltyDashboardNotifications")).Delete("/{id}", dashboardNotifications.DeleteDashboardNotification)
		r.With(middlewares.Auth("authenticate")).Post("/cleanup", dashboardNotifications.CleanupOldNotifications)
	})

	//analytics
	r.Route("/analytics", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate")).Get("/dashboard", analytics.GetDashboardAnalytics)

		// User analytics - simple count by date range
		r.With(middlewares.Auth("authenticate")).Get("/users/count", analytics.GetUserCount)

		// Transaction analytics with filters
		r.With(middlewares.Auth("authenticate")).Get("/transactions", analytics.GetTransactionAnalytics)
		r.With(middlewares.Auth("authenticate")).Get("/transactions/by-service", analytics.GetTransactionsByService)
		r.With(middlewares.Auth("authenticate")).Get("/transactions/enhanced", analytics.GetEnhancedTransactionAnalytics)

		// Survey analytics with filters
		r.With(middlewares.Auth("authenticate")).Get("/surveys", analytics.GetSurveyAnalytics)

		// Time-series analytics endpoints
		r.With(middlewares.Auth("authenticate")).Get("/transactions/enhanced/time-series", analytics.GetTimeSeriesEnhancedTransactionAnalytics)
		r.With(middlewares.Auth("authenticate")).Get("/transactions/time-series", analytics.GetTimeSeriesTransactionAnalytics)
		r.With(middlewares.Auth("authenticate")).Get("/surveys/time-series", analytics.GetTimeSeriesSurveyAnalytics)

		// External time-series analytics endpoints
		r.With(middlewares.Auth("authenticate")).Get("/external/time-series", analytics.GetTimeSeriesExternalAnalytics)
		r.With(middlewares.Auth("authenticate")).Get("/bookings/time-series", analytics.GetTimeSeriesBookingAnalytics)
		r.With(middlewares.Auth("authenticate")).Get("/online-consultations/time-series", analytics.GetTimeSeriesOnlineConsultationAnalytics)
		r.With(middlewares.Auth("authenticate")).Get("/smile-test-drives/time-series", analytics.GetTimeSeriesSmileTestDriveAnalytics)

		// External analytics
		r.With(middlewares.Auth("authenticate")).Get("/total-booking", analytics.GetTotalBookings)
		r.With(middlewares.Auth("authenticate")).Get("/total-smile-test-drives", analytics.GetTotalSmileTestDrives)
		r.With(middlewares.Auth("authenticate")).Get("/total-online-consultations", analytics.GetTotalOnlineConsultations)
		r.With(middlewares.Auth("authenticate")).Get("/total-counts", analytics.GetTotalCounts)

		// Points analytics
		r.With(middlewares.Auth("authenticate")).Get("/points/daily", analytics.GetPointsPerDay)

		// App sharing analytics - TEST API
		r.With(middlewares.Auth("authenticate")).Get("/app-sharing", analytics.GetAppSharingAnalytics)

		// Voucher redemption analytics
		r.With(middlewares.Auth("authenticate")).Get("/voucher-redemption", analytics.GetVoucherRedemptionAnalytics)
		r.With(middlewares.Auth("authenticate")).Get("/voucher-redemption/by-service-type", analytics.GetVoucherRedemptionByServiceType)
		r.With(middlewares.Auth("authenticate")).Get("/voucher-redemption/time-series", analytics.GetTimeSeriesVoucherRedemptionAnalytics)

		// Voucher usage analytics (total usage count, not unique users)
		r.With(middlewares.Auth("authenticate")).Get("/voucher-usage/by-service-type", analytics.GetVoucherUsageByServiceType)
		r.With(middlewares.Auth("authenticate")).Get("/voucher-usage/time-series", analytics.GetTimeSeriesVoucherUsageAnalytics)
	})

	//setup
	r.Route("/setup", func(r chi.Router) {
		r.With(middlewares.Auth("authenticate", "postCapability")).Post("/capabilities", SetupCapabilitiesHandler)
	})

	return r

}
