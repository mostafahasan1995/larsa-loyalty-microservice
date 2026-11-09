package services

import (
	"context"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/database"
	"loyalty/pkg/models"
	"loyalty/pkg/util"
	"time"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DailyTaskInteractionService interface {
	GetOne(id string) (*models.DailyTaskInteraction, error)
	GetByUserId(userId primitive.ObjectID, startDate, endDate time.Time) ([]models.DailyTaskInteraction, error)
	Get(args *util.ListArgs) (*models.DailyTaskInteractionWithPagination, error)
	GetAll(query string) ([]models.DailyTaskInteraction, error)
	Add(data *models.DailyTaskInteraction) error
	Update(id string, data *models.DailyTaskInteraction) error
	Upsert(filter bson.M, data *models.DailyTaskInteraction) error
	Delete(id string) error
}

type dailyTaskInteractionSvcs struct {
	ctx      context.Context
	db       string
	collName string
}

func NewDailyTaskInteractionSvcs(ctx context.Context, db string) DailyTaskInteractionService {
	return &dailyTaskInteractionSvcs{
		ctx:      ctx,
		db:       db,
		collName: "loyaltyDailyTaskInteractions",
	}
}

func (w *dailyTaskInteractionSvcs) GetOne(id string) (*models.DailyTaskInteraction, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	_id, errId := primitive.ObjectIDFromHex(id)

	if errId != nil {
		return nil, errId
	}
	filterStage := bson.M{"$match": bson.M{"_id": _id}}
	pipeline := append([]bson.M{filterStage})
	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.DailyTaskInteraction
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *dailyTaskInteractionSvcs) GetByUserId(userId primitive.ObjectID, startDate, endDate time.Time) ([]models.DailyTaskInteraction, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)
	filter := bson.M{
		"user_id": userId,
		"createdAt": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
		"trash": bson.M{"$ne": true},
	}
	cursor, err := coll.Find(w.ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(w.ctx)

	var taskInteractions []models.DailyTaskInteraction
	if err = cursor.All(w.ctx, &taskInteractions); err != nil {
		return nil, err
	}
	return taskInteractions, nil
}

func (w *dailyTaskInteractionSvcs) Get(args *util.ListArgs) (*models.DailyTaskInteractionWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	pipeline := PaginationPipe("daily_task_interactions", args, surveyLookupPipe...)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.DailyTaskInteractionWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *dailyTaskInteractionSvcs) GetAll(query string) ([]models.DailyTaskInteraction, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filters, errQuery := getBsonFromString(query)
	if errQuery != nil {
		return nil, errQuery
	}

	cur, err := coll.Find(w.ctx, filters)

	if err != nil {
		return nil, err
	}

	var result []models.DailyTaskInteraction
	if err := cur.All(w.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (w *dailyTaskInteractionSvcs) Add(data *models.DailyTaskInteraction) error {
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

func (w *dailyTaskInteractionSvcs) Delete(id string) error {
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

func (w *dailyTaskInteractionSvcs) Update(id string, data *models.DailyTaskInteraction) error {
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

func (w *dailyTaskInteractionSvcs) Upsert(filter bson.M, data *models.DailyTaskInteraction) error {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	user := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)
	userId, _ := primitive.ObjectIDFromHex(user.User.Id)
	data.SetCreatedBy(userId)

	update := bson.M{"$set": data}

	upsert := true
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
