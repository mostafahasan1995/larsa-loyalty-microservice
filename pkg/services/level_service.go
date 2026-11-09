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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LevelService interface {
	GetOne(id string) (*models.Level, error)
	GetOneByExperience(experience int64) (*models.Level, error)
	Get(args *util.ListArgs) (*models.LevelWithPagination, error)
	GetAll(query string) ([]models.Level, error)
	Add(data *models.Level) error
	Update(id string, data *models.Level) error
	Delete(id string) error
}

type levelsvcs struct {
	ctx      context.Context
	db       string
	collName string
}

func NewLevelSvcs(ctx context.Context, db string) LevelService {
	return &levelsvcs{
		ctx:      ctx,
		db:       db,
		collName: "loyaltyLevels",
	}
}

func (w *levelsvcs) GetOne(id string) (*models.Level, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	_id, errId := primitive.ObjectIDFromHex(id)

	if errId != nil {
		return nil, errId
	}

	result := coll.FindOne(w.ctx, bson.M{"_id": _id})

	var data models.Level
	if decErr := result.Decode(&data); decErr != nil {
		return nil, decErr
	}

	return &data, nil
}

func (w *levelsvcs) GetOneByExperience(experience int64) (*models.Level, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	result := coll.FindOne(w.ctx,
		bson.M{"experience": bson.M{"$lte": experience}},
		options.FindOne().SetSort(bson.M{"experience": -1}),
	)
	var data models.Level
	if decErr := result.Decode(&data); decErr != nil {
		return nil, decErr
	}

	return &data, nil
}

func (w *levelsvcs) Get(args *util.ListArgs) (*models.LevelWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	pipeline := PaginationPipe("levels", args)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.LevelWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *levelsvcs) GetAll(query string) ([]models.Level, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filters, errQuery := getBsonFromString(query)
	if errQuery != nil {
		return nil, errQuery
	}

	cur, err := coll.Find(w.ctx, filters)

	if err != nil {
		return nil, err
	}

	var result []models.Level
	if err := cur.All(w.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (w *levelsvcs) Add(data *models.Level) error {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	user := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)
	userId, _ := primitive.ObjectIDFromHex(user.User.Id)
	data.SetCreatedBy(userId)

	res, err := coll.InsertOne(w.ctx, data)

	if err != nil {
		return err
	}
	data.Id, _ = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (w *levelsvcs) Delete(id string) error {
	_id, errId := primitive.ObjectIDFromHex(id)
	if errId != nil {
		return errId
	}

	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filter := bson.M{"_id": _id}
	update := bson.M{"$set": bson.M{"trash": true}}

	upsert := false
	after := options.After
	opt := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
		Upsert:         &upsert,
	}
	result := coll.FindOneAndUpdate(
		w.ctx,
		filter,
		update,
		&opt,
	)

	if result.Err() != nil {
		return result.Err()
	}

	return nil
}

func (w *levelsvcs) Update(id string, data *models.Level) error {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	_id, errId := primitive.ObjectIDFromHex(id)
	if errId != nil {
		return errId
	}

	filter := bson.M{"_id": _id}
	update := bson.M{"$set": data}

	upsert := false
	after := options.After
	opt := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
		Upsert:         &upsert,
	}
	result := coll.FindOneAndUpdate(
		w.ctx,
		filter,
		update,
		&opt,
	)

	if errDec := result.Decode(&data); errDec != nil {
		return errDec
	}

	return nil
}
