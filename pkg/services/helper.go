package services

import (
	"context"
	"loyalty/pkg/database"
	"loyalty/pkg/util"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

func getBsonFromString(query string) (bson.M, error) {
	var bs bson.M = bson.M{}
	if query != "" {
		err := bson.UnmarshalExtJSON([]byte(query), true, &bs)
		if err != nil {
			return nil, err
		}
	}

	return bs, nil
}

func WithTransaction[T any](ctx context.Context, callBack func(sessCtx mongo.SessionContext) (T, error)) (T, error) {
	var zero T
	wc := writeconcern.Majority()
	rc := readconcern.Snapshot()
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)
	session, err := database.Mdb.StartSession()
	if err != nil {
		return zero, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(
		ctx,
		func(sessCtx mongo.SessionContext) (interface{}, error) {
			return callBack(sessCtx)
		},
		txnOpts)
	if err != nil {
		return zero, err
	}
	return result.(T), nil
}

func PaginationPipe(resourceName string, args *util.ListArgs, stagesPipes ...bson.M) []bson.M {
	args.Query["trash"] = bson.M{"$ne": true}
	pipe := append([]bson.M{
		{"$match": args.Query},
		{"$sort": args.Sort},
	}, stagesPipes...)
	limit, _ := strconv.ParseInt(args.PerPage, 10, 64)
	pageNum, _ := strconv.ParseInt(args.Page, 10, 32)
	skip := (pageNum - 1) * limit

	pipe = append(pipe, []bson.M{
		{"$facet": bson.M{
			"pagination": []bson.M{
				{"$count": "totalCount"},
				{"$addFields": bson.M{
					"perPage": limit,
					"totalPages": bson.M{
						"$ceil": bson.M{
							"$divide": bson.A{"$totalCount", limit},
						},
					},
				}},
			},
			resourceName: []bson.M{
				{"$skip": skip},
				{"$limit": limit},
				{"$sort": args.Sort},
			},
		}},
		{"$unwind": bson.M{"path": "$pagination", "preserveNullAndEmptyArrays": true}},
	}...)
	return pipe
}
