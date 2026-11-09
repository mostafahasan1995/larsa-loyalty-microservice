package services

import (
	"context"
	"fmt"
	"loyalty/pkg/database"
	"loyalty/pkg/models"
	"loyalty/pkg/util"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SurveyAnswerBulkDeleteResponse represents the response for bulk delete operations
type SurveyAnswerBulkDeleteResponse struct {
	TotalCount   int      `json:"totalCount"`
	SuccessCount int      `json:"successCount"`
	Errors       []string `json:"errors"`
}

type SurveyAnswerService interface {
	GetOne(id string) (*models.SurveyAnswerResponse, error)
	GetOneByUserId(_id, userId primitive.ObjectID) (*models.SurveyAnswer, error)
	Get(args *util.ListArgs) (*models.SurveyAnswerWithPagination, error)
	GetGroupByVisitByUserId(args *util.ListArgs, userId primitive.ObjectID) ([]models.SurveyAnswerWithGroupVisit, error)
	GetAll(query string) ([]models.SurveyAnswer, error)
	Add(data *models.SurveyAnswer) error
	Update(id string, data *models.SurveyAnswer) error
	Archive(userId string) error
	Delete(id string) error
	BulkDelete(ids []string) (*SurveyAnswerBulkDeleteResponse, error)
}

type surveyAnswerSvcs struct {
	ctx      context.Context
	db       string
	collName string
}

func NewSurveyAnswerSvcs(ctx context.Context, db string) SurveyAnswerService {
	return &surveyAnswerSvcs{
		ctx:      ctx,
		db:       db,
		collName: "loyaltySurveyAnswers",
	}
}

var surveyLookupPipe = []bson.M{
	{
		"$lookup": bson.M{
			"from":         "loyaltySurveys",
			"localField":   "survey_id",
			"foreignField": "_id",
			"as":           "survey",
		},
	},
	{
		"$unwind": bson.M{
			"path":                       "$survey",
			"preserveNullAndEmptyArrays": true,
		},
	},
}

func (w *surveyAnswerSvcs) GetOne(id string) (*models.SurveyAnswerResponse, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	_id, errId := primitive.ObjectIDFromHex(id)

	if errId != nil {
		return nil, errId
	}
	filterStage := bson.M{"$match": bson.M{"_id": _id}}
	pipeline := append([]bson.M{filterStage}, surveyLookupPipe...)
	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.SurveyAnswerResponse
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *surveyAnswerSvcs) GetGroupByVisitByUserId(args *util.ListArgs, userId primitive.ObjectID) ([]models.SurveyAnswerWithGroupVisit, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filterStage := bson.M{
		"$match": bson.M{"user_id": userId},
	}

	group := bson.M{
		"$group": bson.M{
			"_id": bson.M{
				"$ifNull": []string{"$visit", "old visits"},
			},
			"answers": bson.M{
				"$push": "$$ROOT",
			},
		},
	}

	// Build pipeline with survey lookup before grouping
	pipeline := []bson.M{filterStage}
	pipeline = append(pipeline, surveyLookupPipe...)
	pipeline = append(pipeline, group)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var result []models.SurveyAnswerWithGroupVisit
	if err := cursor.All(w.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (w *surveyAnswerSvcs) GetOneByUserId(_id, userId primitive.ObjectID) (*models.SurveyAnswer, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filter := bson.M{"_id": _id, "user_id": userId}

	result := coll.FindOne(w.ctx, filter)

	var data models.SurveyAnswer
	if decErr := result.Decode(&data); decErr != nil {
		return nil, decErr
	}

	return &data, nil
}

func (w *surveyAnswerSvcs) Get(args *util.ListArgs) (*models.SurveyAnswerWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	pipeline := PaginationPipe("survey_answers", args, surveyLookupPipe...)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.SurveyAnswerWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *surveyAnswerSvcs) GetAll(query string) ([]models.SurveyAnswer, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filters, errQuery := getBsonFromString(query)
	if errQuery != nil {
		return nil, errQuery
	}

	cur, err := coll.Find(w.ctx, filters)

	if err != nil {
		return nil, err
	}

	var result []models.SurveyAnswer
	if err := cur.All(w.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (w *surveyAnswerSvcs) Add(data *models.SurveyAnswer) error {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	res, err := coll.InsertOne(w.ctx, data)

	if err != nil {
		return err
	}
	data.Id, _ = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (w *surveyAnswerSvcs) Delete(id string) error {
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

func (w *surveyAnswerSvcs) BulkDelete(ids []string) (*SurveyAnswerBulkDeleteResponse, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	response := &SurveyAnswerBulkDeleteResponse{
		TotalCount: len(ids),
	}

	for _, id := range ids {
		_id, errId := primitive.ObjectIDFromHex(id)
		if errId != nil {
			response.Errors = append(response.Errors, fmt.Sprintf("Invalid ID %s: %v", id, errId))
			continue
		}

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
			if result.Err() == mongo.ErrNoDocuments {
				response.Errors = append(response.Errors, fmt.Sprintf("Survey answer with ID %s not found", id))
			} else {
				response.Errors = append(response.Errors, fmt.Sprintf("Error deleting survey answer %s: %v", id, result.Err()))
			}
			continue
		}

		response.SuccessCount++

		// Audit trail is handled by middleware
	}

	return response, nil
}

func (w *surveyAnswerSvcs) Update(id string, data *models.SurveyAnswer) error {
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

	if result.Err() != nil {
		return result.Err()
	}

	return nil
}

func (w *surveyAnswerSvcs) Archive(userId string) error {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	_id, errId := primitive.ObjectIDFromHex(userId)
	if errId != nil {
		return errId
	}

	filter := bson.M{"user_id": _id}
	update := bson.M{"$set": bson.M{"is_archived": true}}

	_, err := coll.UpdateMany(
		w.ctx,
		filter,
		update,
	)
	if err != nil {
		return err
	}

	return nil
}
