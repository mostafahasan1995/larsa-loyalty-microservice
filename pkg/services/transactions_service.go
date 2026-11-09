package services

import (
	"context"
	"loyalty/pkg/constants"
	"loyalty/pkg/database"
	"loyalty/pkg/models"
	"loyalty/pkg/util"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TransactionsService interface {
	GetOne(id string) (*models.Transaction, error)
	Get(args *util.ListArgs, userId *primitive.ObjectID) (*models.TransactionWithPagination, error)
	GetAll(query string) ([]*models.Transaction, error)
	Add(data *models.Transaction) (*models.Transaction, error)
	Delete(id string) error
}

type transactionsvcs struct {
	MainService[*models.Transaction, *models.Transaction]
}

func NewTransactionSvcs(ctx context.Context, db string) TransactionsService {
	// Define lookup pipeline for voucher and service details
	lookupPipeline := []bson.M{
		// First lookup: Get voucher details
		{
			"$lookup": bson.M{
				"from":         "loyaltyVouchers",
				"localField":   "ref_id",
				"foreignField": "_id",
				"as":           "voucher_info",
			},
		},
		// Second lookup: Get service details from voucher
		{
			"$lookup": bson.M{
				"from": "loyaltyServices",
				"let":  bson.M{"serviceId": bson.M{"$arrayElemAt": []interface{}{"$voucher_info.service_id", 0}}},
				"pipeline": []bson.M{
					{
						"$match": bson.M{
							"$expr": bson.M{
								"$eq": []interface{}{"$_id", "$$serviceId"},
							},
						},
					},
				},
				"as": "service_info",
			},
		},
		// Update type_name field for voucher transactions
		{
			"$addFields": bson.M{
				"type_name": bson.M{
					"$cond": bson.M{
						"if": bson.M{
							"$and": []bson.M{
								{"$in": []interface{}{"$type", []int{int(constants.TXN_VOUCHER_USED), int(constants.TXN_VOUCHER_EXPIRED)}}},
								{"$gt": []interface{}{bson.M{"$size": "$voucher_info"}, 0}},
								{"$gt": []interface{}{bson.M{"$size": "$service_info"}, 0}},
							},
						},
						"then": bson.M{"$arrayElemAt": []interface{}{"$service_info.name.en", 0}},
						"else": "$type_name",
					},
				},
			},
		},
	}

	return &transactionsvcs{MainService[*models.Transaction, *models.Transaction]{
		ctx:       ctx,
		db:        db,
		collName:  "loyaltyTransactions",
		MongoPipe: lookupPipeline,
	}}
}

// Override GetAll to use aggregation with lookup pipeline
func (t *transactionsvcs) GetAll(query string) ([]*models.Transaction, error) {
	coll := database.Mdb.Database(t.db).Collection(t.collName)

	filters, errQuery := getBsonFromString(query)
	if errQuery != nil {
		return nil, errQuery
	}

	// Build aggregation pipeline with match stage and lookup stages
	pipeline := []bson.M{{"$match": filters}}
	pipeline = append(pipeline, t.MongoPipe...)

	cursor, err := coll.Aggregate(t.ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var result []*models.Transaction
	if err := cursor.All(t.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (t *transactionsvcs) Get(args *util.ListArgs, userId *primitive.ObjectID) (*models.TransactionWithPagination, error) {
	coll := database.Mdb.Database(t.db).Collection(t.collName)

	if userId != nil {
		args.Query["user_id"] = userId
	}
	// Include the lookup pipeline stages
	pipeline := PaginationPipe("transactions", args, t.MongoPipe...)

	cursor, err := coll.Aggregate(t.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(t.ctx)

	// Decode directly into the expected structure
	var results []models.TransactionWithPagination
	if err := cursor.All(t.ctx, &results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &models.TransactionWithPagination{
			Transactions: []models.Transaction{},
			Pagination:   models.Pagination{},
		}, nil
	}

	return &results[0], nil
}
