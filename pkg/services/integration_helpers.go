package services

import (
	"context"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/models"
	"loyalty/pkg/util"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NotifyDashboardOnFeedback(clientId, userId, userName, feedbackId string) {
	ctx := context.Background()
	notification := &models.DashboardNotification{
		Title: models.LocalizableString{
			Ar: "اقتراح جديد من المريض",
			En: "New Patient Suggestion",
		},
		Message: models.LocalizableString{
			Ar: fmt.Sprintf("تم استلام اقتراح جديد من %s يتطلب المراجعة", userName),
			En: fmt.Sprintf("New suggestion received from %s requiring review", userName),
		},
		ActionType: models.DashboardNotificationActionFeedback,
		EntityType: models.DashboardNotificationEntityFeedback,
		Priority:   models.DashboardNotificationPriorityMedium,
		TargetRole: models.DashboardNotificationTargetAdmin,
		UserName:   userName,
	}

	if entityId, err := primitive.ObjectIDFromHex(feedbackId); err == nil {
		notification.EntityId = entityId
	}
	if userObjectId, err := primitive.ObjectIDFromHex(userId); err == nil {
		notification.UserId = userObjectId
	}

	go CreateDashboardNotification(ctx, clientId, notification)

	util.LocalizeNotify(
		util.LocalizeNotificationMsg{
			XClient: clientId,
			Header: models.LocalizableString{
				Ar: "اقتراح جديد من المريض",
				En: "New Patient Suggestion",
			},
			Content: models.LocalizableString{
				Ar: fmt.Sprintf("تم استلام اقتراح جديد من %s", userName),
				En: fmt.Sprintf("New suggestion received from %s", userName),
			},
			TopicType: constants.NotificationTopicTypeAdmin,
			Topic:     "admin",
			Type:      "dashboard",
			Data: util.NotificationPayload{
				Id:    feedbackId,
				Model: "feedback",
			},
		},
	)
}

func NotifyDashboardOnSurveyAnswer(clientId, userId, userName, surveyAnswerId string) {
	ctx := context.Background()
	notification := &models.DashboardNotification{
		Title: models.LocalizableString{
			Ar: "إجابة استبيان جديدة",
			En: "New Survey Response",
		},
		Message: models.LocalizableString{
			Ar: fmt.Sprintf("تم استلام إجابة استبيان جديدة من %s", userName),
			En: fmt.Sprintf("New survey response received from %s", userName),
		},
		ActionType: models.DashboardNotificationActionSurveyAnswer,
		EntityType: models.DashboardNotificationEntitySurveyAnswer,
		Priority:   models.DashboardNotificationPriorityLow,
		TargetRole: models.DashboardNotificationTargetAdmin,
		UserName:   userName,
	}

	if entityId, err := primitive.ObjectIDFromHex(surveyAnswerId); err == nil {
		notification.EntityId = entityId
	}
	if userObjectId, err := primitive.ObjectIDFromHex(userId); err == nil {
		notification.UserId = userObjectId
	}

	go CreateDashboardNotification(ctx, clientId, notification)

	util.LocalizeNotify(
		util.LocalizeNotificationMsg{
			XClient: clientId,
			Header: models.LocalizableString{
				Ar: "إجابة استبيان جديدة",
				En: "New Survey Response",
			},
			Content: models.LocalizableString{
				Ar: fmt.Sprintf("تم استلام إجابة استبيان جديدة من %s", userName),
				En: fmt.Sprintf("New survey response received from %s", userName),
			},
			TopicType: constants.NotificationTopicTypeAdmin,
			Topic:     "admin",
			Type:      "dashboard",
			Data: util.NotificationPayload{
				Id:    surveyAnswerId,
				Model: "survey_answer",
			},
		},
	)
}

func NotifyDashboardOnChallengeSubmission(clientId, userId, userName, challengeAnswerId, challengeTitle string) {
	ctx := context.Background()

	// Always default to facebook follow action type
	actionType := models.DashboardNotificationActionVoucherFacebook
	entityType := models.DashboardNotificationEntityVoucher
	priority := models.DashboardNotificationPriorityMedium

	// Use the challenge title as-is for notification
	titleAr := challengeTitle
	titleEn := challengeTitle
	messageAr := fmt.Sprintf("قام %s بإكمال: %s", userName, challengeTitle)
	messageEn := fmt.Sprintf("%s completed: %s", userName, challengeTitle)

	notification := &models.DashboardNotification{
		Title: models.LocalizableString{
			Ar: titleAr,
			En: titleEn,
		},
		Message: models.LocalizableString{
			Ar: messageAr,
			En: messageEn,
		},
		ActionType: actionType,
		EntityType: entityType,
		Priority:   priority,
		TargetRole: models.DashboardNotificationTargetAdmin,
		UserName:   userName,
	}

	if entityId, err := primitive.ObjectIDFromHex(challengeAnswerId); err == nil {
		notification.EntityId = entityId
	}
	if userObjectId, err := primitive.ObjectIDFromHex(userId); err == nil {
		notification.UserId = userObjectId
	}

	go CreateDashboardNotification(ctx, clientId, notification)

	util.LocalizeNotify(
		util.LocalizeNotificationMsg{
			XClient: clientId,
			Header: models.LocalizableString{
				Ar: titleAr,
				En: titleEn,
			},
			Content: models.LocalizableString{
				Ar: messageAr,
				En: messageEn,
			},
			TopicType: constants.NotificationTopicTypeAdmin,
			Topic:     "admin",
			Type:      "dashboard",
			Data: util.NotificationPayload{
				Id:    challengeAnswerId,
				Model: "challenge_answer",
			},
		},
	)
}

func NotifyDashboardOnVoucherAction(clientId, userId, userName, voucherId, actionType string) {
	ctx := context.Background()
	var titleAr, titleEn, messageAr, messageEn string
	var priority string

	switch actionType {
	case models.DashboardNotificationActionVoucherGoogle:
		titleAr = "مراجعة جوجل تحتاج تأكيد"
		titleEn = "Google Review Needs Confirmation"
		messageAr = fmt.Sprintf("المريض %s ادعى كتابة مراجعة جوجل ويحتاج تأكيد لإضافة النقاط", userName)
		messageEn = fmt.Sprintf("Patient %s claimed to write Google review and needs confirmation for points", userName)
		priority = models.DashboardNotificationPriorityHigh
	case models.DashboardNotificationActionVoucherFacebook:
		titleAr = "متابعة فيسبوك تحتاج تأكيد"
		titleEn = "Facebook Follow Needs Confirmation"
		messageAr = fmt.Sprintf("المريض %s ادعى متابعة صفحة فيسبوك ويحتاج تأكيد لإضافة النقاط", userName)
		messageEn = fmt.Sprintf("Patient %s claimed to follow Facebook page and needs confirmation for points", userName)
		priority = models.DashboardNotificationPriorityMedium
	case models.DashboardNotificationActionVoucherInstagram:
		titleAr = "متابعة انستغرام تحتاج تأكيد"
		titleEn = "Instagram Follow Needs Confirmation"
		messageAr = fmt.Sprintf("المريض %s ادعى متابعة حساب انستغرام ويحتاج تأكيد لإضافة النقاط", userName)
		messageEn = fmt.Sprintf("Patient %s claimed to follow Instagram account and needs confirmation for points", userName)
		priority = models.DashboardNotificationPriorityMedium
	default:
		titleAr = "إجراء قسيمة يتطلب الموافقة"
		titleEn = "Voucher Action Requires Approval"
		messageAr = fmt.Sprintf("المريض %s قام بإجراء ويحتاج موافقة لإضافة النقاط", userName)
		messageEn = fmt.Sprintf("Patient %s performed action and needs approval for points", userName)
		priority = models.DashboardNotificationPriorityMedium
	}

	notification := &models.DashboardNotification{
		Title: models.LocalizableString{
			Ar: titleAr,
			En: titleEn,
		},
		Message: models.LocalizableString{
			Ar: messageAr,
			En: messageEn,
		},
		ActionType: actionType,
		EntityType: models.DashboardNotificationEntityVoucher,
		Priority:   priority,
		TargetRole: models.DashboardNotificationTargetReceptionist,
		UserName:   userName,
		Metadata: map[string]interface{}{
			"voucher_action":        actionType,
			"requires_verification": true,
		},
	}

	if entityId, err := primitive.ObjectIDFromHex(voucherId); err == nil {
		notification.EntityId = entityId
	}
	if userObjectId, err := primitive.ObjectIDFromHex(userId); err == nil {
		notification.UserId = userObjectId
	}

	go CreateDashboardNotification(ctx, clientId, notification)

	util.LocalizeNotify(
		util.LocalizeNotificationMsg{
			XClient: clientId,
			Header: models.LocalizableString{
				Ar: titleAr,
				En: titleEn,
			},
			Content: models.LocalizableString{
				Ar: messageAr,
				En: messageEn,
			},
			TopicType: constants.NotificationTopicTypeReceptionist,
			Topic:     "receptionist",
			Type:      "dashboard",
			Data: util.NotificationPayload{
				Id:    voucherId,
				Model: "voucher_action",
			},
		},
	)
}

func CreateSocialMediaVoucher(ctx context.Context, clientId, userId, userName, actionType string) (string, error) {
	voucherSvc := NewVoucherSvc(ctx, clientId)

	var points int

	switch actionType {
	case constants.GOOGLE_MAPS_REVIEW.String():
		points = 100
	case constants.FACEBOOK_FOLLOW.String():
		points = 50
	case constants.INSTAGRAM_FOLLOW.String():
		points = 50
	default:
		return "", fmt.Errorf("unsupported action type: %s", actionType)
	}

	userObjectId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return "", err
	}

	voucher := &models.Voucher{
		Code:      fmt.Sprintf("%s_%d", actionType, time.Now().Unix()),
		Points:    int64(points),
		UserId:    userObjectId,
		ExpiredAt: time.Now().AddDate(0, 0, 7),
		IsUsed:    false,
	}

	voucher.SetCreatedBy(userObjectId)

	voucherResponse, err := voucherSvc.Add(voucher)
	if err != nil {
		return "", err
	}

	// Update voucher with the response data if needed
	if voucherResponse != nil {
		voucher.Id = voucherResponse.Id
	}

	var notificationActionType string
	switch actionType {
	case constants.GOOGLE_MAPS_REVIEW.String():
		notificationActionType = models.DashboardNotificationActionVoucherGoogle
	case constants.FACEBOOK_FOLLOW.String():
		notificationActionType = models.DashboardNotificationActionVoucherFacebook
	case constants.INSTAGRAM_FOLLOW.String():
		notificationActionType = models.DashboardNotificationActionVoucherInstagram
	}

	go NotifyDashboardOnVoucherAction(clientId, userId, userName, voucher.Id.Hex(), notificationActionType)

	return voucher.Code, nil
}

func NotifyDashboardOnVoucherRedemption(clientId, userId, userName, voucherId string) {
	ctx := context.Background()
	notification := &models.DashboardNotification{
		Title: models.LocalizableString{
			Ar: "استرداد قسيمة جديدة",
			En: "New Voucher Redemption",
		},
		Message: models.LocalizableString{
			Ar: fmt.Sprintf("المريض %s استرد قسيمة ويحتاج مراجعة", userName),
			En: fmt.Sprintf("Patient %s redeemed voucher and needs review", userName),
		},
		ActionType: models.DashboardNotificationActionVoucherRedemption,
		EntityType: models.DashboardNotificationEntityVoucher,
		Priority:   models.DashboardNotificationPriorityMedium,
		TargetRole: models.DashboardNotificationTargetAdmin,
		UserName:   userName,
	}

	if entityId, err := primitive.ObjectIDFromHex(voucherId); err == nil {
		notification.EntityId = entityId
	}
	if userObjectId, err := primitive.ObjectIDFromHex(userId); err == nil {
		notification.UserId = userObjectId
	}

	go CreateDashboardNotification(ctx, clientId, notification)

	util.LocalizeNotify(
		util.LocalizeNotificationMsg{
			XClient: clientId,
			Header: models.LocalizableString{
				Ar: "استرداد قسيمة جديدة",
				En: "New Voucher Redemption",
			},
			Content: models.LocalizableString{
				Ar: fmt.Sprintf("تم استرداد قسيمة جديدة بواسطة %s", userName),
				En: fmt.Sprintf("New voucher redeemed by %s", userName),
			},
			TopicType: constants.NotificationTopicTypeAdmin,
			Topic:     "admin",
			Type:      "dashboard",
			Data: util.NotificationPayload{
				Id:    voucherId,
				Model: "voucher_redemption",
			},
		},
	)
}
