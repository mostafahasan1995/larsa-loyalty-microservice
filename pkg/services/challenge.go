package services

import (
	"context"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/database"
	"loyalty/pkg/models"
	"loyalty/pkg/util"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ChallengesService interface {
	GetOne(id string) (*models.ChallengeResponse, error)
	Get(args *util.ListArgs, userId *primitive.ObjectID, isAnswered bool) (*models.ChallengeWithPagination, error)
	GetAll(query string) ([]models.Challenge, error)
	Add(data *models.Challenge) error
	Update(id string, data *models.Challenge) error
	Delete(id string) error
	Submit(challengeAnswer *models.ChallengeAnswer) (*models.ChallengeAnswerResponse, error)
}

type challengesvcs struct {
	ctx      context.Context
	db       string
	collName string
}

func NewChallengeSvcs(ctx context.Context, db string) ChallengesService {
	return &challengesvcs{
		ctx:      ctx,
		db:       db,
		collName: "loyaltyChallenges",
	}
}

func isAnsweredChallengePipe(userId primitive.ObjectID, isAnswered bool) []bson.M {
	return []bson.M{{
		"$lookup": bson.M{
			"from": "loyaltyChallengeAnswers",
			"let":  bson.M{"challengeId": "$_id"},
			"pipeline": []bson.M{
				{"$match": bson.M{
					"$expr": bson.M{
						"$and": []bson.M{
							{"$eq": bson.A{"$challenge_id", "$$challengeId"}},
							{"$eq": bson.A{"$user_id", userId}},
							{"$ne": bson.A{"$trash", true}},
						},
					},
				}},
			},
			"as": "responses",
		}},
		{"$match": bson.M{
			"responses": lo.Ternary(isAnswered, bson.M{"$ne": []bson.M{}}, bson.M{"$eq": []bson.M{}}),
		}},
	}
}

func (b *challengesvcs) GetOne(id string) (*models.ChallengeResponse, error) {
	coll := database.Mdb.Database(b.db).Collection(b.collName)

	_id, errId := primitive.ObjectIDFromHex(id)

	if errId != nil {
		return nil, errId
	}

	result := coll.FindOne(b.ctx, bson.M{"_id": _id})

	var data models.ChallengeResponse
	if decErr := result.Decode(&data); decErr != nil {
		return nil, decErr
	}

	return &data, nil
}

func (b *challengesvcs) Get(args *util.ListArgs, userId *primitive.ObjectID, isAnswered bool) (*models.ChallengeWithPagination, error) {
	coll := database.Mdb.Database(b.db).Collection(b.collName)

	extraPipe := []bson.M{}
	if userId != nil {
		extraPipe = isAnsweredChallengePipe(*userId, isAnswered)
	}
	pipeline := PaginationPipe("challenges", args, extraPipe...)

	cursor, err := coll.Aggregate(b.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.ChallengeWithPagination
	if cursor.Next(b.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (b *challengesvcs) GetAll(query string) ([]models.Challenge, error) {
	coll := database.Mdb.Database(b.db).Collection(b.collName)

	filters, errQuery := getBsonFromString(query)
	if errQuery != nil {
		return nil, errQuery
	}

	cur, err := coll.Find(b.ctx, filters)

	if err != nil {
		return nil, err
	}

	var result []models.Challenge
	if err := cur.All(b.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (w *challengesvcs) Add(data *models.Challenge) error {
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

func (b *challengesvcs) Update(id string, data *models.Challenge) error {

	_id, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return err
	}

	coll := database.Mdb.Database(b.db).Collection(b.collName)

	filter := bson.M{"_id": _id}
	update := bson.M{"$set": data}

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

	if errDec := result.Decode(&data); errDec != nil {
		return errDec
	}

	return nil
}

func (b *challengesvcs) Delete(id string) error {
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

func (b *challengesvcs) Submit(answer *models.ChallengeAnswer) (*models.ChallengeAnswerResponse, error) {
	challenge, err := b.GetOne(answer.ChallengeId.Hex())
	if err != nil {
		return nil, fmt.Errorf("challenge not found %w", err)
	}
	result, err := WithTransaction(b.ctx, func(sessCtx mongo.SessionContext) (*models.ChallengeAnswerResponse, error) {
		// check if user answered this challenge before
		challengeAnswerSvc := NewChallengeAnswerSvcs(sessCtx, b.db)
		_, err = challengeAnswerSvc.GetOneByUserId(answer.ChallengeId, answer.UserId)
		if err == nil {
			return nil, fmt.Errorf("user has done this challenge before")
		}
		answer.IsAccepted = false
		res, err := challengeAnswerSvc.Add(answer)
		if err != nil {
			return nil, err
		}
		return res, nil
	})

	if err == nil && result != nil {
		// Get challenge title for notification
		challengeTitle := challenge.Title.En
		if challengeTitle == "" {
			challengeTitle = challenge.Title.Ar
		}

		// Send notification with challenge title
		go NotifyDashboardOnChallengeSubmission(b.db, answer.UserId.Hex(), "", result.Id.Hex(), challengeTitle)
	}

	return result, err
}
