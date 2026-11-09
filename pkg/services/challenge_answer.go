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

type ChallengeAnswerService interface {
	GetOne(id string) (*models.ChallengeAnswerResponse, error)
	GetOneByUserId(_id, userId primitive.ObjectID) (*models.ChallengeAnswer, error)
	Get(args *util.ListArgs) (*models.ChallengeAnswerWithPagination, error)
	GetAll(query string) ([]models.ChallengeAnswer, error)
	Add(data *models.ChallengeAnswer) (*models.ChallengeAnswerResponse, error)
	Update(id string, data *models.ChallengeAnswer) error
	Delete(id string) error
	Accept(id string) error
	Reject(id string) error
}

type challengeAnswerSvcs struct {
	MainService[*models.ChallengeAnswer, *models.ChallengeAnswerResponse]
}

func NewChallengeAnswerSvcs(ctx context.Context, db string) ChallengeAnswerService {
	return &challengeAnswerSvcs{
		MainService: MainService[*models.ChallengeAnswer, *models.ChallengeAnswerResponse]{
			ctx:       ctx,
			db:        db,
			collName:  "loyaltyChallengeAnswers",
			MongoPipe: challengeLookupPipe,
		},
	}
}

var challengeLookupPipe = []bson.M{
	{
		"$lookup": bson.M{
			"from":         "loyaltyChallenges",
			"localField":   "challenge_id",
			"foreignField": "_id",
			"as":           "challenge",
		},
	},
	{
		"$unwind": bson.M{
			"path":                       "$challenge",
			"preserveNullAndEmptyArrays": true,
		},
	},
}

func (w *challengeAnswerSvcs) GetOneByUserId(_id, userId primitive.ObjectID) (*models.ChallengeAnswer, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)
	result := coll.FindOne(w.ctx, bson.M{"challenge_id": _id, "user_id": userId, "trash": bson.M{"$ne": true}})

	var data models.ChallengeAnswer
	if decErr := result.Decode(&data); decErr != nil {
		return nil, decErr
	}

	return &data, nil
}

func (w *challengeAnswerSvcs) Get(args *util.ListArgs) (*models.ChallengeAnswerWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	pipeline := PaginationPipe("challenge_answers", args, challengeLookupPipe...)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.ChallengeAnswerWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *challengeAnswerSvcs) GetAll(query string) ([]models.ChallengeAnswer, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filters, errQuery := getBsonFromString(query)
	if errQuery != nil {
		return nil, errQuery
	}

	cur, err := coll.Find(w.ctx, filters)

	if err != nil {
		return nil, err
	}

	var result []models.ChallengeAnswer
	if err := cur.All(w.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (w *challengeAnswerSvcs) Delete(id string) error {
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

func (w *challengeAnswerSvcs) Update(id string, data *models.ChallengeAnswer) error {
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

// TODO: use transactions
func (w *challengeAnswerSvcs) Accept(id string) error {
	answer, err := w.GetOne(id)
	if err != nil || answer.Trash {
		return fmt.Errorf("answer not found %w", err)
	}
	if answer.IsAccepted {
		return fmt.Errorf("answer already accepted")
	}
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	_id, errId := primitive.ObjectIDFromHex(id)
	if errId != nil {
		return errId
	}

	filter := bson.M{"_id": _id}
	update := bson.M{"$set": bson.M{"is_accepted": true}}

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
	if err := result.Err(); err != nil {
		return err
	}
	user := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)
	userId, _ := primitive.ObjectIDFromHex(user.User.Id)
	transactionService := NewTransactionSvcs(w.ctx, w.db)
	newTransaction := &models.Transaction{
		UserId:   answer.UserId,
		Activity: "Challenge answer is accepted",
		Type:     constants.TXN_CHALLENGE_ANSWER_ACCEPTED,
		TypeName: constants.TXN_CHALLENGE_ANSWER_ACCEPTED.String(),
		Amount:   float64(answer.Challenge.Points),
		RefId:    &_id,
	}
	newTransaction.SetCreatedBy(userId)

	_, err = transactionService.Add(newTransaction)
	if err != nil {
		return err
	}
	userWalletService := NewUserWalletSvcs(w.ctx, w.db)
	wallet, err := userWalletService.GetWalletByUserId(answer.UserId.Hex())
	if err != nil {
		return fmt.Errorf("wallet not found")
	}
	_, err = userWalletService.AddPoints(wallet.Id.Hex(), answer.Challenge.Points, answer.Challenge.Points)
	if err != nil {
		return fmt.Errorf("wallet not updated %w", err)
	}
	return nil
}

func (w *challengeAnswerSvcs) Reject(id string) error {
	answer, err := w.GetOne(id)
	if err != nil || answer.Trash {
		return fmt.Errorf("answer not found %w", err)
	}
	if answer.IsAccepted {
		return fmt.Errorf("answer already accepted, could not reject it")
	}
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	_id, errId := primitive.ObjectIDFromHex(id)
	if errId != nil {
		return errId
	}

	filter := bson.M{"_id": _id}
	update := bson.M{"$set": bson.M{"is_accepted": false, "trash": true}}

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
	if err := result.Err(); err != nil {
		return err
	}

	return nil
}
