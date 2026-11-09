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

type QuizAnswerService interface {
	GetOne(id string) (*models.QuizAnswerResponse, error)
	GetOneByUserId(_id, userId primitive.ObjectID) (*models.QuizAnswer, error)
	Get(args *util.ListArgs) (*models.QuizAnswerWithPagination, error)
	GetAll(query string) ([]models.QuizAnswer, error)
	Add(data *models.QuizAnswer) error
	Update(id string, data *models.QuizAnswer) error
	Delete(id string) error
}

type quizAnswerSvcs struct {
	ctx      context.Context
	db       string
	collName string
}

func NewQuizAnswerSvcs(ctx context.Context, db string) QuizAnswerService {
	return &quizAnswerSvcs{
		ctx:      ctx,
		db:       db,
		collName: "loyaltyQuizAnswers",
	}
}

var quizLookupPipe = []bson.M{
	{
		"$lookup": bson.M{
			"from":         "loyaltyQuizzes",
			"localField":   "quiz_id",
			"foreignField": "_id",
			"as":           "quiz",
		},
	},
	{
		"$unwind": bson.M{
			"path":                       "$quiz",
			"preserveNullAndEmptyArrays": true,
		},
	},
}

func (w *quizAnswerSvcs) GetOne(id string) (*models.QuizAnswerResponse, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	_id, errId := primitive.ObjectIDFromHex(id)

	if errId != nil {
		return nil, errId
	}
	filterStage := bson.M{"$match": bson.M{"_id": _id}}
	pipeline := append([]bson.M{filterStage}, quizLookupPipe...)
	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.QuizAnswerResponse
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *quizAnswerSvcs) GetOneByUserId(_id, userId primitive.ObjectID) (*models.QuizAnswer, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)
	result := coll.FindOne(w.ctx, bson.M{"quiz_id": _id, "user_id": userId, "is_correct": true})

	var data models.QuizAnswer
	if decErr := result.Decode(&data); decErr != nil {
		return nil, decErr
	}

	return &data, nil
}

func (w *quizAnswerSvcs) Get(args *util.ListArgs) (*models.QuizAnswerWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	pipeline := PaginationPipe("quiz_answers", args, quizLookupPipe...)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.QuizAnswerWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *quizAnswerSvcs) GetAll(query string) ([]models.QuizAnswer, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filters, errQuery := getBsonFromString(query)
	if errQuery != nil {
		return nil, errQuery
	}

	cur, err := coll.Find(w.ctx, filters)

	if err != nil {
		return nil, err
	}

	var result []models.QuizAnswer
	if err := cur.All(w.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (w *quizAnswerSvcs) Add(data *models.QuizAnswer) error {
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

func (w *quizAnswerSvcs) Delete(id string) error {
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

func (w *quizAnswerSvcs) Update(id string, data *models.QuizAnswer) error {
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
