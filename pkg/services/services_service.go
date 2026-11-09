package services

import (
	"context"
	"errors"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/database"
	"loyalty/pkg/models"
	"loyalty/pkg/util"
	"time"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"github.com/samber/lo"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ServicesService interface {
	GetOne(id string) (*models.ServiceResponse, error)
	Get(args *util.ListArgs) (*models.ServiceWithPagination, error)
	GetAll(query string) ([]models.Service, error)
	Add(data *models.Service) (*models.ServiceResponse, error)
	Update(id string, data *models.Service) (*models.ServiceResponse, error)
	Delete(id string) error
	Redeem(data models.RedeemRequest, userId string, isUsed bool) (*models.UserWalletResponse, error)
	//
	Reorder(serviceId string, data *models.ServiceReorder) error
}

type servicesvcs struct {
	MainService[*models.Service, *models.ServiceResponse]
}

func NewServiceSvcs(ctx context.Context, db string) ServicesService {
	return &servicesvcs{
		MainService: MainService[*models.Service, *models.ServiceResponse]{
			ctx:             ctx,
			db:              db,
			collName:        "loyaltyServices",
			MongoPipe:       serviceLookupPipe,
			AfterCreateHook: NotifyOnCreateService,
		},
	}
}

var serviceLookupPipe = []bson.M{
	{"$lookup": bson.M{
		"from":         "loyaltyLevels",
		"localField":   "level",
		"foreignField": "_id",
		"as":           "level",
	}},
	{"$unwind": bson.M{"path": "$level", "preserveNullAndEmptyArrays": true}},
	{"$lookup": bson.M{
		"from":         "loyaltyServiceGroups",
		"localField":   "group",
		"foreignField": "_id",
		"as":           "group",
	}},
	{"$unwind": bson.M{"path": "$group", "preserveNullAndEmptyArrays": true}},
}

func (b *servicesvcs) Get(args *util.ListArgs) (*models.ServiceWithPagination, error) {
	coll := database.Mdb.Database(b.db).Collection(b.collName)

	pipeline := PaginationPipe("services", args, serviceLookupPipe...)

	cursor, err := coll.Aggregate(b.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.ServiceWithPagination
	if cursor.Next(b.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
	}
	return &result, nil
}

// override Add method on main service to add seq
func (b *servicesvcs) Add(data *models.Service) (*models.ServiceResponse, error) {
	result, err := WithTransaction(b.ctx, func(sessCtx mongo.SessionContext) (*models.ServiceResponse, error) {
		counterSvcs := NewCounterSvcs(sessCtx, b.db)
		seq, err := counterSvcs.GetAndUpdateSourceSeq("loyaltyservice")
		if err != nil {
			return nil, err
		}

		data.Seq = seq

		return b.MainService.Add(data)
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (b *servicesvcs) GetAll(query string) ([]models.Service, error) {
	coll := database.Mdb.Database(b.db).Collection(b.collName)

	filters, errQuery := getBsonFromString(query)
	if errQuery != nil {
		return nil, errQuery
	}

	cur, err := coll.Find(b.ctx, filters)

	if err != nil {
		return nil, err
	}

	var result []models.Service
	if err := cur.All(b.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (b *servicesvcs) Delete(id string) error {
	_id, errId := primitive.ObjectIDFromHex(id)
	if errId != nil {
		return errId
	}

	coll := database.Mdb.Database(b.db).Collection(b.collName)

	filter := bson.M{"_id": _id}
	update := bson.M{"$set": bson.M{"trash": true}}

	upsert := false
	after := options.After
	opt := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
		Upsert:         &upsert,
	}
	result := coll.FindOneAndUpdate(
		b.ctx,
		filter,
		update,
		&opt,
	)

	if result.Err() != nil {
		return result.Err()
	}

	return nil
}

func (b *servicesvcs) Redeem(data models.RedeemRequest, userId string, isUsed bool) (*models.UserWalletResponse, error) {
	_id, errId := primitive.ObjectIDFromHex(data.ServiceId)
	if errId != nil {
		return nil, errId
	}
	result, err := WithTransaction(b.ctx, func(sessCtx mongo.SessionContext) (*models.UserWalletResponse, error) {
		// check if service is exists
		serviceSvc := NewServiceSvcs(sessCtx, b.db)
		service, err := serviceSvc.GetOne(data.ServiceId)
		if err != nil {
			return nil, fmt.Errorf("service not found: %w", err)
		}
		// get user wallet
		walletService := NewUserWalletSvcs(sessCtx, b.db)
		wallet, err := walletService.GetWalletByUserId(userId)
		if err != nil {
			return nil, fmt.Errorf("wallet not found: %w", err)
		}
		voucher := models.Voucher{
			ServiceId: _id,
			UserId:    wallet.UserId,
			IsUsed:    isUsed,
			PartnerId: service.PartnerId,
			ExpiredAt: time.Now().AddDate(0, 1, 0),
		}
		if service.Level.Experience > wallet.Experience {
			return nil, fmt.Errorf("user level is not enough")
		}
		if service.Discount != nil {
			currentDate := time.Now()
			if currentDate.After(service.Discount.ValidUntil) {
				return nil, fmt.Errorf("discount has expired")
			}
			voucher.Discount = service.Discount
		}
		if wallet.Points < service.Points {
			return nil, fmt.Errorf("no enough points to redeem the service")
		}
		voucher.Points = service.Points
		wallet.Points -= service.Points

		createdByUser, _ := b.ctx.Value(constants.CtxAuthKey).(common.Credentials)
		createdById, _ := primitive.ObjectIDFromHex(createdByUser.User.Id)
		voucher.SetCreatedBy(createdById)
		wallet.SetUpdatedBy(createdById)
		voucherService := NewVoucherSvc(sessCtx, b.db)
		_, err = walletService.Update(wallet.Id.Hex(), &wallet.UserWallet)
		if err != nil {
			return nil, err
		}
		_, err = voucherService.Add(&voucher)
		if err != nil {
			return nil, err
		}

		// Create notification for voucher redemption
		user := b.ctx.Value(constants.CtxAuthKey).(common.Credentials)
		userName := user.User.FirstName
		if userName == "" {
			userName = "User"
		}
		go NotifyDashboardOnVoucherRedemption(b.db, user.User.Id, userName, voucher.Id.Hex())

		wallet.Vouchers = append(wallet.Vouchers, voucher)
		if voucher.IsUsed {
			user := b.ctx.Value(constants.CtxAuthKey).(common.Credentials)
			userId, _ := primitive.ObjectIDFromHex(user.User.Id)
			transactionService := NewTransactionSvcs(sessCtx, b.db)
			newTransaction := &models.Transaction{
				UserId:   voucher.UserId,
				Activity: "Voucher Used",
				Type:     constants.TXN_VOUCHER_USED,
				TypeName: constants.TXN_VOUCHER_USED.String(),
				Amount:   -float64(service.Points),
				RefId:    &voucher.Id,
			}
			newTransaction.SetCreatedBy(userId)
			_, err = transactionService.Add(newTransaction)
			if err != nil {
				return nil, err
			}
		}
		return wallet, nil
	})
	return result, err
}

func NotifyOnCreateService(clientId string, service *models.ServiceResponse) {
	fmt.Println(service.Group)
	model := ""
	id := ""
	if service.Group != nil {
		id = service.Group.Id.Hex()
		model = lo.Ternary(service.Group.IsSpecial, "service/special", "service/redeem")
	}
	util.LocalizeNotify(
		util.LocalizeNotificationMsg{
			XClient: clientId,
			Header: models.LocalizableString{
				Ar: "الخصومات والعروض",
				En: "Discounts and offers!",
			},
			Content: models.LocalizableString{
				Ar: "عرض خاص! استفد من الخصومات والعروض الجديدة اآلن. ال تفوت الفرصة",
				En: "Special Offer! Take advantage of the new discounts and offers now. Don't miss out",
			},
			TopicType: constants.NotificationTopicTypeLevel,
			Topic:     service.Level.Id.Hex(),
			Type:      "fcm",
			Data: util.NotificationPayload{
				Id:    id,
				Model: model,
			},
		},
	)
}

func (b *servicesvcs) Reorder(serviceId string, data *models.ServiceReorder) error {

	svcsid, err := primitive.ObjectIDFromHex(serviceId)
	if err != nil {
		return err
	}

	_, errUp := WithTransaction(b.ctx, func(sessCtx mongo.SessionContext) (bool, error) {

		coll := database.Mdb.Database(b.db).Collection(b.collName)

		var service models.Service
		if err := coll.FindOne(sessCtx, bson.M{"_id": svcsid}).Decode(&service); err != nil {
			return false, errors.New("error get related service")
		}

		if service.Seq != data.OldSeq {
			return false, errors.New("service seq must equal old seq")
		}

		var filter, update bson.M

		if data.NewSeq > data.OldSeq {

			filter = bson.M{"$and": bson.A{
				bson.M{"seq": bson.M{"$gt": data.OldSeq}},
				bson.M{"seq": bson.M{"$lte": data.NewSeq}},
			}}

			update = bson.M{"$inc": bson.M{"seq": -1}}

		} else if data.NewSeq < data.OldSeq {

			filter = bson.M{"$and": bson.A{
				bson.M{"seq": bson.M{"$gte": data.NewSeq}},
				bson.M{"seq": bson.M{"$lt": data.OldSeq}},
			}}

			update = bson.M{"$inc": bson.M{"seq": 1}}

		} else {
			return false, errors.New("error sorting data")
		}

		_, errUp := coll.UpdateMany(sessCtx, filter, update)
		if errUp != nil {
			return false, errUp
		}

		_, errUpdSvcs := coll.UpdateOne(sessCtx, bson.M{"_id": svcsid}, bson.M{"$set": bson.M{"seq": data.NewSeq}})
		if errUpdSvcs != nil {
			return false, errUpdSvcs
		}

		return true, nil
	})

	if errUp != nil {
		return errUp
	}

	return nil

}
