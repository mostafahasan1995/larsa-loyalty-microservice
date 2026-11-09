package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/database"
	"loyalty/pkg/models"
	"loyalty/pkg/util"
	"math"
	"math/big"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserWalletService interface {
	GetOne(id string) (*models.UserWalletResponse, error)
	Get(args *util.ListArgs) (*models.UserWalletWithPagination, error)
	GetAll(query string) ([]*models.UserWallet, error)
	Add(data *models.UserWallet) (*models.UserWalletResponse, error)
	Delete(id string) error
	GetWalletByUserId(id string) (*models.UserWalletResponse, error)
	Update(string, *models.UserWallet) (*models.UserWalletResponse, error)
	BuyPoints(id string, moneyAmount float64) (*models.UserWallet, error)
	VisitUser(id string) error
	PointsPerAED(moneyAmount float64) models.UserWalletPointsPriceResponse
	AddPoints(id string, points int64, experience int64) (*models.UserWallet, error)
	UpdateProfile(Id string) (*models.UserWalletResponse, error)
	VerifyCode(code string, userId string) (string, error)
	CreateInvitation(userId string) (string, error)
}

type userwalletsvcs struct {
	MainService[*models.UserWallet, *models.UserWalletResponse]
}

var walletPipe = []bson.M{
	{"$lookup": bson.M{
		"from": "loyaltyLevels",
		"let":  bson.M{"serviceExperience": "$experience"},
		"pipeline": []bson.M{
			{"$match": bson.M{
				"$expr": bson.M{"$and": bson.A{
					bson.M{"$lte": bson.A{"$experience", "$$serviceExperience"}},
					bson.M{"$ne": bson.A{"$trash", true}},
				}},
			}},
			{"$sort": bson.M{"experience": -1}},
			{"$limit": 1},
		},
		"as": "level",
	}},
	{"$unwind": bson.M{"path": "$level", "preserveNullAndEmptyArrays": true}},

	{"$lookup": bson.M{
		"from":         "loyaltyVouchers",
		"localField":   "user_id",
		"foreignField": "user_id",
		"pipeline": []bson.M{
			{"$match": bson.M{
				"$expr": bson.M{"$and": bson.A{
					bson.M{"$ne": bson.A{"$trash", true}},
				}},
			}},
		},
		"as": "vouchers",
	}},
}

// Characters used in the unique code generation
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateCode generates a unique code with a specified length
func GenerateCode(length int) (string, error) {
	code := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		randomInt, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		code[i] = charset[randomInt.Int64()]
	}
	return string(code), nil
}

func NewUserWalletSvcs(ctx context.Context, db string) UserWalletService {
	return &userwalletsvcs{
		MainService: MainService[*models.UserWallet, *models.UserWalletResponse]{
			ctx:       ctx,
			db:        db,
			collName:  "loyaltyUserWallets",
			MongoPipe: walletPipe,
		},
	}
}

func (w *userwalletsvcs) Get(args *util.ListArgs) (*models.UserWalletWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	pipeline := PaginationPipe("user_wallets", args, walletPipe...)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(w.ctx)
	var result models.UserWalletWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *userwalletsvcs) CreateInvitation(userId string) (string, error) {
	_id, errId := primitive.ObjectIDFromHex(userId)
	if errId != nil {
		return "", fmt.Errorf("invalid id: %w", errId)
	}
	filter := bson.M{"user_id": _id}
	coll := database.Mdb.Database(w.db).Collection(w.collName)
	update := bson.M{
		"$inc": bson.M{"code_sent": 1},
	}
	if _, err := coll.UpdateOne(w.ctx, filter, update); err != nil {
		return "", fmt.Errorf("failed to update wallet code: %w", err)
	}
	return "Invitation created successfully", nil
}

func (w *userwalletsvcs) VerifyCode(code string, userId string) (string, error) {
	_id, errId := primitive.ObjectIDFromHex(userId)
	if errId != nil {
		return "", fmt.Errorf("invalid id: %w", errId)
	}
	if code == "" {
		return "", fmt.Errorf("invalid code")
	}
	filter := bson.M{"code": code}
	coll := database.Mdb.Database(w.db).Collection(w.collName)
	transactionService := NewTransactionSvcs(w.ctx, w.db)
	result := coll.FindOne(w.ctx, filter)
	settingsSvcs := NewSettingsSvcs(w.ctx, w.db)

	var data models.UserWallet

	if err := result.Decode(&data); err != nil {
		return "", fmt.Errorf("invalid code: %s", code)
	}
	receiverAmount := float64(settingsSvcs.GetValueByKeyAsInt(constants.SettingsKeyDefaultJoinWithInvitePoints, 50))
	receiverTransaction := &models.Transaction{
		UserId:   _id,
		Activity: "Joined By Invitation",
		Type:     constants.TXN_INVITE_BY_INVITATION,
		TypeName: constants.TXN_INVITE_BY_INVITATION.String(),
		Amount:   receiverAmount,
		RefId:    &data.Id,
	}
	receiverTransaction.SetCreatedBy(_id)
	_, err := transactionService.Add(receiverTransaction)
	if err != nil {
		return "", err
	}
	receiverWallet, err := w.GetWalletByUserId(userId)
	if err != nil {
		return "", err
	}
	if _, err := w.AddPoints(receiverWallet.Id.Hex(), int64(receiverAmount), int64(receiverAmount)); err != nil {
		return "", fmt.Errorf("failed to update wallet code: %w", err)
	}
	senderAmount := float64(settingsSvcs.GetValueByKeyAsInt(constants.SettingsKeyDefaultJoinWithYourInvitePoints, 100))
	senderTransaction := &models.Transaction{
		UserId:   data.UserId,
		Activity: "User Joined by your Invitation",
		Type:     constants.TXN_USER_JOIN_BY_YOUR_INVITATION,
		TypeName: constants.TXN_USER_JOIN_BY_YOUR_INVITATION.String(),
		Amount:   senderAmount,
		RefId:    &data.Id,
	}
	senderTransaction.SetCreatedBy(_id)
	_, err = transactionService.Add(senderTransaction)
	if err != nil {
		return "", err
	}
	if _, err := w.AddPoints(data.Id.Hex(), int64(senderAmount), int64(senderAmount)); err != nil {
		return "", fmt.Errorf("failed to update wallet code: %w", err)
	}
	update := bson.M{
		"$inc": bson.M{"code_used": 1},
	}
	if _, err := coll.UpdateOne(w.ctx, filter, update); err != nil {
		return "", fmt.Errorf("failed to update wallet code: %w", err)
	}

	return "Verified successfully", nil

}

func (w *userwalletsvcs) GetWalletByUserId(id string) (*models.UserWalletResponse, error) {
	_id, errId := primitive.ObjectIDFromHex(id)
	if errId != nil {
		return nil, fmt.Errorf("invalid id: %w", errId)
	}

	settingsSvcs := NewSettingsSvcs(w.ctx, w.db)

	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filter := bson.M{"user_id": _id}
	result := coll.FindOne(w.ctx, filter)

	wallet_points := settingsSvcs.GetValueByKeyAsInt(constants.SettingsKeyDefaultWalletPoints, 50)
	experience := settingsSvcs.GetValueByKeyAsInt(constants.SettingsKeyDefaultWalletExperience, 50)

	var data models.UserWallet

	if err := result.Decode(&data); err != nil {
		// If user doesn't have a wallet, create a new one
		code, genCodeErr := GenerateCode(6)
		if genCodeErr != nil {
			return nil, genCodeErr
		}
		data = models.UserWallet{
			Id:         primitive.NewObjectID(),
			Points:     wallet_points,
			Experience: experience,
			UserId:     _id,
			VisitCount: 1,
			Code:       code,
			CodeUsed:   0,
			CodeSent:   0,
		}
		data.SetCreatedBy(_id)
		if _, createErr := w.Add(&data); createErr != nil {
			return nil, createErr
		}

		transactionService := NewTransactionSvcs(w.ctx, w.db)

		newTransaction := &models.Transaction{
			UserId:   _id,
			Activity: "Wallet created",
			Type:     constants.TXN_WALLET_CREATED,
			TypeName: constants.TXN_WALLET_CREATED.String(),
			Amount:   float64(wallet_points),
			RefId:    &data.Id,
		}
		newTransaction.SetCreatedBy(_id)

		_, err = transactionService.Add(newTransaction)
		if err != nil {
			return nil, err
		}
	} else {
		if data.Code == "" {
			code, genCodeErr := GenerateCode(6)
			if genCodeErr != nil {
				return nil, genCodeErr
			}
			data.Code = code
			data.CodeUsed = 0
			data.CodeSent = 0
			update := bson.M{
				"$set": bson.M{
					"code":      data.Code,
					"code_used": data.CodeUsed,
					"code_send": data.CodeSent,
				},
			}
			if _, err := coll.UpdateOne(w.ctx, filter, update); err != nil {
				return nil, fmt.Errorf("failed to update wallet code: %w", err)
			}
		}
	}
	return w.GetOne(data.Id.Hex())
}

func (w *userwalletsvcs) AddPoints(id string, points, experience int64) (*models.UserWallet, error) {
	wallet, err := w.GetOne(id)
	if err != nil {
		return nil, fmt.Errorf("wallet not found: %w", err)
	}
	_id, errId := primitive.ObjectIDFromHex(id)
	if errId != nil {
		return nil, fmt.Errorf("invalid id: %w", errId)
	}
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filter := bson.M{"_id": _id}
	update := bson.M{"$inc": bson.M{"points": points, "experience": experience}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	res := coll.FindOneAndUpdate(w.ctx, filter, update, opts)

	if res.Err() != nil {
		return nil, res.Err()
	}
	walletAfterUpdate, _ := w.GetOne(id)
	if wallet.Level.Num < walletAfterUpdate.Level.Num {
		// Notify the user
		go NotifyOnLevelUp(w.db, walletAfterUpdate)
	}
	go NotifyOnAddingPointsToWallet(w.db, &walletAfterUpdate.UserWallet)

	return &walletAfterUpdate.UserWallet, nil
}

func (w *userwalletsvcs) BuyPoints(id string, moneyAmount float64) (*models.UserWallet, error) {
	settingsSvcs := NewSettingsSvcs(w.ctx, w.db)
	pointPrice := settingsSvcs.GetValueByKeyAsFloat(constants.SettingsKeyPointPrice, 1)

	pointsAmount := int64(math.Round(moneyAmount / pointPrice))

	// Add points to the wallet
	wallet, err := w.AddPoints(id, pointsAmount, pointsAmount)
	if err != nil {
		return nil, err
	}
	user := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)
	userId, _ := primitive.ObjectIDFromHex(user.User.Id)

	transactionService := NewTransactionSvcs(w.ctx, w.db)
	newTransaction := &models.Transaction{
		UserId:   wallet.UserId,
		Activity: "Points added to wallet",
		Type:     constants.TXN_POINTS_ADDED_TO_WALLET,
		TypeName: constants.TXN_POINTS_ADDED_TO_WALLET.String(),
		Amount:   float64(pointsAmount),
		RefId:    &wallet.Id,
	}
	newTransaction.SetCreatedBy(userId)

	transactionService.Add(newTransaction)
	if err != nil {
		return nil, err
	}
	return wallet, nil
}

func (w *userwalletsvcs) PointsPerAED(moneyAmount float64) models.UserWalletPointsPriceResponse {
	res := models.UserWalletPointsPriceResponse{MoneyAmount: moneyAmount, Points: 0}
	// Get the point price from the settings service
	settingsSvcs := NewSettingsSvcs(w.ctx, w.db)
	pointPrice := settingsSvcs.GetValueByKeyAsFloat(constants.SettingsKeyPointPrice, 1)

	res.Points = int64(math.Round(moneyAmount / pointPrice))
	return res
}

func NotifyOnAddingPointsToWallet(clientId string, wallet *models.UserWallet) {
	util.LocalizeNotify(
		util.LocalizeNotificationMsg{
			XClient: clientId,
			Header: models.LocalizableString{
				Ar: "تحديث فوري! ",
				En: "Instant Update!",
			},
			Content: models.LocalizableString{
				Ar: "لقد تمت إضافة نقاط جديدة إلى حسابك. تابع تقدمك واستمر في كسب المزيد",
				En: "New points have been added to your account. Keep track of your progress and continue earning more.",
			},
			Topic:     wallet.UserId.Hex(),
			TopicType: constants.NotificationTopicTypeUser,
			Type:      "fcm",
			Data: util.NotificationPayload{
				Id:    wallet.Id.Hex(),
				Model: "service",
			},
		},
	)
}

func NotifyOnLevelUp(clientId string, wallet *models.UserWalletResponse) {
	util.LocalizeNotify(
		util.LocalizeNotificationMsg{
			XClient: clientId,
			Header: models.LocalizableString{
				Ar: "ترقية! ",
				En: "Level Up!",
			},
			Content: models.LocalizableString{
				Ar: fmt.Sprintf("تهانينا! اصبحت في المستوى %d.", wallet.Level.Num),
				En: fmt.Sprintf("Congratulations! You have reached level %d.", wallet.Level.Num),
			},
			TopicType: constants.NotificationTopicTypeUser,
			Topic:     wallet.UserId.Hex(),
			Type:      "fcm",
			Data: util.NotificationPayload{
				Id:    wallet.UserId.Hex(),
				Model: "wallet",
			},
		},
	)
}

func (w *userwalletsvcs) UpdateProfile(Id string) (*models.UserWalletResponse, error) {
	settingsSvcs := NewSettingsSvcs(w.ctx, w.db)
	pointPrice := settingsSvcs.GetValueByKeyAsInt(constants.SettingsUpdateProfilePoints, 25)

	walletResponse, errWallet := w.GetWalletByUserId(Id)
	if errWallet != nil {
		return nil, errWallet
	}
	if walletResponse.UpdatedProfile {
		return nil, fmt.Errorf("user already earned update profile points")
	}

	// Add points to the wallet
	wallet, err := w.AddPoints(walletResponse.Id.Hex(), pointPrice, pointPrice)
	if err != nil {
		return nil, err
	}
	wallet.UpdatedProfile = true
	walletResponse, err = w.Update(walletResponse.Id.Hex(), wallet)
	if err != nil {
		return nil, err
	}

	transactionService := NewTransactionSvcs(w.ctx, w.db)
	newTransaction := &models.Transaction{
		UserId:   wallet.UserId,
		Activity: "Profile Updated",
		Type:     constants.TXN_UPDATE_PROFILE,
		TypeName: constants.TXN_UPDATE_PROFILE.String(),
		Amount:   float64(pointPrice),
		RefId:    &wallet.Id,
	}
	userId, _ := primitive.ObjectIDFromHex(Id)
	newTransaction.SetCreatedBy(userId)

	transactionService.Add(newTransaction)
	if err != nil {
		return nil, err
	}
	return walletResponse, nil
}

func (w *userwalletsvcs) VisitUser(userId string) error {
	surveyAnswerService := NewSurveyAnswerSvcs(w.ctx, w.db)

	err := surveyAnswerService.Archive(userId)
	if err != nil {
		return err
	}
	wallet, err := w.GetWalletByUserId(userId)
	if err != nil {
		return err
	}
	wallet.VisitCount += 1

	_, err = w.Update(wallet.Id.Hex(), &wallet.UserWallet)

	if err != nil {
		return err
	}

	go util.LocalizeNotify(
		util.LocalizeNotificationMsg{
			XClient: w.db,
			Header: models.LocalizableString{
				Ar: "مرحبًا بك في مركز الحكمة الطبي!",
				En: "Welcome to Hikma Medical Center!",
			},
			Content: models.LocalizableString{
				Ar: "تم تسجيل وصولك بنجاح. نتمنى لك زيارة مريحة.",
				En: "You have successfully checked in. We wish you a comfortable visit",
			},
			TopicType: constants.NotificationTopicTypeUser,
			Topic:     userId,
			Type:      "fcm",
			Data: util.NotificationPayload{
				Model: "voucher",
			},
		},
	)

	go util.LocalizeNotify(
		util.LocalizeNotificationMsg{
			XClient: w.db,
			Header: models.LocalizableString{
				Ar: "مرحبًا بك في مركز الحكمة الطبي!",
				En: "Welcome to Hikma Medical Center!",
			},
			Content: models.LocalizableString{
				Ar: "الرجاء تعبئة الاستبيان بعد الانتهاء من الزيارة",
				En: "Please fill out the survey after finishing",
			},
			TopicType: constants.NotificationTopicTypeUser,
			Topic:     userId,
			Type:      "fcm",
			Data: util.NotificationPayload{
				Model: "survey",
			},
		},
	)

	return nil
}
