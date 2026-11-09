package services

import (
	"context"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/database"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MainDataType interface {
	SetCreatedBy(id primitive.ObjectID)
}

type MainService[T MainDataType, R MainDataType] struct {
	ctx             context.Context
	db              string
	collName        string
	MongoPipe       []bson.M
	AfterCreateHook func(clientId string, data R)
}

func (s *MainService[T, R]) Add(data T) (R, error) {
	coll := database.Mdb.Database(s.db).Collection(s.collName)
	var zero R

	if s.ctx.Value(constants.CtxAuthKey) != nil {
		user := s.ctx.Value(constants.CtxAuthKey).(common.Credentials)
		userId, _ := primitive.ObjectIDFromHex(user.User.Id)
		data.SetCreatedBy(userId)
	}

	res, err := coll.InsertOne(s.ctx, data)
	if err != nil {
		return zero, err
	}
	result, err := s.GetOne(res.InsertedID.(primitive.ObjectID).Hex())
	if s.AfterCreateHook != nil {
		go s.AfterCreateHook(s.db, result)
	}
	return result, err
}

func (s *MainService[T, R]) GetOne(id string) (R, error) {
	coll := database.Mdb.Database(s.db).Collection(s.collName)
	var zero R

	_id, errId := primitive.ObjectIDFromHex(id)

	if errId != nil {
		return zero, errId
	}
	filterStage := bson.M{"$match": bson.M{"_id": _id}}
	pipeline := append([]bson.M{filterStage}, s.MongoPipe...)
	cursor, err := coll.Aggregate(s.ctx, pipeline)
	if err != nil {
		return zero, err
	}
	var result R
	if cursor.Next(s.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return zero, err
		}
		return result, nil
	}
	return zero, fmt.Errorf("no data found")
}

func (s *MainService[T, R]) Update(id string, data T) (R, error) {
	var zero R
	_id, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return zero, err
	}

	coll := database.Mdb.Database(s.db).Collection(s.collName)

	update := bson.M{"$set": data}
	_, err = coll.UpdateByID(s.ctx, _id, update)
	if err != nil {
		return zero, err
	}

	return s.GetOne(_id.Hex())
}

func (s *MainService[T, R]) Delete(id string) error {
	_id, errId := primitive.ObjectIDFromHex(id)
	if errId != nil {
		return errId
	}

	coll := database.Mdb.Database(s.db).Collection(s.collName)

	filter := bson.M{"_id": _id}
	update := bson.M{"$set": bson.M{"trash": true}}

	upsert := false
	after := options.After
	opt := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
		Upsert:         &upsert,
	}
	result := coll.FindOneAndUpdate(
		s.ctx,
		filter,
		update,
		&opt,
	)

	if result.Err() != nil {
		return result.Err()
	}

	return nil
}

func (s *MainService[T, R]) GetAll(query string) ([]T, error) {
	coll := database.Mdb.Database(s.db).Collection(s.collName)

	filters, errQuery := getBsonFromString(query)
	if errQuery != nil {
		return nil, errQuery
	}

	cur, err := coll.Find(s.ctx, filters)

	if err != nil {
		return nil, err
	}

	var result []T
	if err := cur.All(s.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}
