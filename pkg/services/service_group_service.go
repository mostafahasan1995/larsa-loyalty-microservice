package services

import (
	"context"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/database"
	"loyalty/pkg/models"
	"loyalty/pkg/util"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"go.mongodb.org/mongo-driver/bson"
)

type ServiceGroup interface {
	GetOne(id string) (*models.ServiceGroup, error)
	Get(args *util.ListArgs) (*models.ServiceGroupWithPagination, error)
	GetAll(query string) ([]*models.ServiceGroup, error)
	Add(data *models.ServiceGroup) (*models.ServiceGroup, error)
	Update(id string, data *models.ServiceGroup) (*models.ServiceGroup, error)
	Delete(id string) error
}

type Servicegroupsvcs struct {
	MainService[*models.ServiceGroup, *models.ServiceGroup]
}

func NewServicegroupsvcs(ctx context.Context, db string) ServiceGroup {
	return &Servicegroupsvcs{MainService: MainService[*models.ServiceGroup, *models.ServiceGroup]{
		ctx:      ctx,
		db:       db,
		collName: "loyaltyServiceGroups",
	}}
}

func serviceGroupLookupPipe(wallet models.UserWalletResponse) []bson.M {
	return []bson.M{
		{"$lookup": bson.M{
			"from": "loyaltyServices",
			"let":  bson.M{"groupId": "$_id"},
			"pipeline": []bson.M{
				{"$match": bson.M{
					"$expr": bson.M{"$and": []bson.M{
						{"$eq": bson.A{"$group", "$$groupId"}},
						{"$lte": bson.A{"$points", wallet.Points}},
						{"$ne": bson.A{"$trash", true}},
						{"$gte": bson.A{"$discount.valid_until", "$$NOW"}},
					}}},
				},
				{"$lookup": bson.M{
					"from": "loyaltyLevels",
					"let":  bson.M{"levelId": "$level"},
					"pipeline": []bson.M{
						{"$match": bson.M{
							"$expr": bson.M{"$and": []bson.M{
								{"$eq": bson.A{"$_id", "$$levelId"}},
								{"$lte": bson.A{"$num", wallet.Level.Num}},
								{"$ne": bson.A{"$trash", true}},
							}},
						}},
					},
					"as": "levelObject",
				}},
				{"$unwind": "$levelObject"},
			},
			"as": "services",
		}},
		{"$addFields": bson.M{
			"services_count": bson.M{"$size": "$services"},
		}},
	}
}

func (w *Servicegroupsvcs) Get(args *util.ListArgs) (*models.ServiceGroupWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	user := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)

	walletSvc := NewUserWalletSvcs(w.ctx, w.db)
	wallet, err := walletSvc.GetWalletByUserId(user.User.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %v", err)
	}
	pipeline := PaginationPipe("service_groups", args, serviceGroupLookupPipe(*wallet)...)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.ServiceGroupWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}
