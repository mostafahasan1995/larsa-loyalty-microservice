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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SurveyService interface {
	GetOne(id string) (*models.SurveyResponse, error)
	Get(args *util.ListArgs, userId *primitive.ObjectID, isAnswered bool, justWithoutArchived bool) (*models.SurveyWithPagination, error)
	GetAll(query string) ([]models.Survey, error)
	Add(data *models.Survey) error
	Update(id string, data *models.Survey) (*models.SurveyResponse, error)
	Delete(id string) error
	Submit(surveyAnswer *models.SurveyAnswer) (*models.SurveyAnswer, error)
	GetUserServey(id string) ([]models.Survey, error)
}

type surveysvcs struct {
	MainService[*models.Survey, *models.SurveyResponse]
}

func NewSurveysvcs(ctx context.Context, db string) SurveyService {
	return &surveysvcs{
		MainService[*models.Survey, *models.SurveyResponse]{
			ctx:      ctx,
			db:       db,
			collName: "loyaltySurveys",
		}}
}

func isAnsweredSurveyPipe(userId primitive.ObjectID, isAnswered bool, justWithoutArchived bool) []bson.M {
	return []bson.M{{
		"$lookup": bson.M{
			"from": "loyaltySurveyAnswers",
			"let":  bson.M{"surveyId": "$_id"},
			"pipeline": []bson.M{
				{"$match": bson.M{
					"$expr": bson.M{
						"$and": []bson.M{
							{"$eq": bson.A{"$survey_id", "$$surveyId"}},
							{"$eq": bson.A{"$user_id", userId}},
							{"$ne": bson.A{"$trash", true}},
							lo.Ternary(justWithoutArchived, bson.M{"$ne": bson.A{"$is_archived", true}}, bson.M{}),
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

func (w *surveysvcs) GetOne(id string) (*models.SurveyResponse, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	_id, errId := primitive.ObjectIDFromHex(id)

	if errId != nil {
		return nil, errId
	}

	result := coll.FindOne(w.ctx, bson.M{"_id": _id})

	var data models.SurveyResponse
	if decErr := result.Decode(&data); decErr != nil {
		return nil, decErr
	}

	return &data, nil
}

func (w *surveysvcs) Get(args *util.ListArgs, userId *primitive.ObjectID, isAnswered bool, justWithoutArchived bool) (*models.SurveyWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	extraPipe := []bson.M{}
	if userId != nil {
		extraPipe = isAnsweredSurveyPipe(*userId, isAnswered, justWithoutArchived)
	}
	pipeline := PaginationPipe("surveys", args, extraPipe...)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.SurveyWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *surveysvcs) GetAll(query string) ([]models.Survey, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filters, errQuery := getBsonFromString(query)
	if errQuery != nil {
		return nil, errQuery
	}

	cur, err := coll.Find(w.ctx, filters)

	if err != nil {
		return nil, err
	}

	var result []models.Survey
	if err := cur.All(w.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (w *surveysvcs) Add(data *models.Survey) error {
	user := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)
	userId, _ := primitive.ObjectIDFromHex(user.User.Id)
	data.SetCreatedBy(userId)

	wc := writeconcern.Majority()
	rc := readconcern.Snapshot()
	txnOptions := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)
	// Starts a session on the client
	session, err := database.Mdb.StartSession()
	if err != nil {
		return err
	}
	// Defers ending the session after the transaction is committed or ended
	defer session.EndSession(w.ctx)

	_, errTnx := session.WithTransaction(w.ctx, func(ctx mongo.SessionContext) (interface{}, error) {
		surveycoll := database.Mdb.Database(w.db).Collection(w.collName)

		if len(data.Questions) == 0 {
			return nil, fmt.Errorf("questions is required")
		}

		var questions []models.SurveyQuestion
		for _, q := range data.Questions {
			if len(q.Options) == 0 {
				return nil, fmt.Errorf("question must have options")
			}

			var ops []models.SurveyOption
			for _, o := range q.Options {
				ops = append(ops, models.SurveyOption{Id: primitive.NewObjectID(), Text: o.Text})
			}
			// save options
			q.Options = ops

			questions = append(questions, models.SurveyQuestion{
				Id:      primitive.NewObjectID(),
				Title:   q.Title,
				Type:    q.Type,
				Options: q.Options,
			})
		}

		data.Questions = questions

		res, err := surveycoll.InsertOne(w.ctx, data)
		if err != nil {
			return nil, fmt.Errorf("failed to insert survey")
		}
		data.Id, _ = res.InsertedID.(primitive.ObjectID)
		return data, nil
	}, txnOptions)

	if errTnx != nil {
		return errTnx
	}

	return nil
}

func (w *surveysvcs) Delete(id string) error {
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

func (b *surveysvcs) Submit(answer *models.SurveyAnswer) (*models.SurveyAnswer, error) {
	survey, err := b.GetOne(answer.SurveyId.Hex())
	if err != nil {
		return nil, fmt.Errorf("survey not found %w", err)
	}
	result, err := WithTransaction(b.ctx, func(sessCtx mongo.SessionContext) (*models.SurveyAnswer, error) {
		// check if user answered this survey before
		surveyAnswerSvc := NewSurveyAnswerSvcs(sessCtx, b.db)
		_, err = surveyAnswerSvc.GetOneByUserId(answer.SurveyId, answer.UserId)
		if err == nil {
			return nil, fmt.Errorf("user has answered this survey before")
		}

		transactionService := NewTransactionSvcs(sessCtx, b.db)
		newTransaction := &models.Transaction{
			UserId:   answer.UserId,
			Activity: "Survey submitted",
			Type:     constants.TXN_SURVEY_SUBMIT,
			TypeName: constants.TXN_SURVEY_SUBMIT.String(),
			Amount:   float64(survey.Points),
			RefId:    &survey.Id,
		}
		newTransaction.SetCreatedBy(answer.UserId)

		_, err = transactionService.Add(newTransaction)
		if err != nil {
			return nil, err
		}
		userWalletService := NewUserWalletSvcs(sessCtx, b.db)
		wallet, _ := userWalletService.GetWalletByUserId(answer.UserId.Hex())

		_, err = userWalletService.AddPoints(wallet.Id.Hex(), survey.Points, survey.Points)
		if err != nil {
			return nil, err
		}
		answer.Points = survey.Points
		answer.Visit = fmt.Sprintf("visit_%v", wallet.VisitCount)
		err = surveyAnswerSvc.Add(answer)
		if err != nil {
			return nil, err
		}

		user := b.ctx.Value(constants.CtxAuthKey).(common.Credentials)
		go NotifyDashboardOnSurveyAnswer(b.db, user.User.Id, user.User.FirstName, answer.Id.Hex())

		return answer, nil
	})
	return result, err
}

func (w *surveysvcs) GetUserServey(id string) ([]models.Survey, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)
	surveyAnswerColl := database.Mdb.Database(w.db).Collection("loyaltysurveyanswer")

	_id, errId := primitive.ObjectIDFromHex(id)
	if errId != nil {
		return nil, errId
	}

	// Retrieve the survey IDs the user has answered
	cursor, err := surveyAnswerColl.Find(w.ctx, bson.M{"user_id": _id})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(w.ctx)

	var userAnswers []models.SurveyAnswer
	if err := cursor.All(w.ctx, &userAnswers); err != nil {
		return nil, err
	}

	answeredSurveyIds := make(map[primitive.ObjectID]struct{})
	for _, answer := range userAnswers {
		answeredSurveyIds[answer.SurveyId] = struct{}{}
	}

	// Get the list of all surveys
	surveyCursor, err := coll.Find(w.ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer surveyCursor.Close(w.ctx)

	var allSurveys []models.Survey
	if err := surveyCursor.All(w.ctx, &allSurveys); err != nil {
		return nil, err
	}

	// dentify the surveys the user has not answered
	var notAnsweredSurveys []models.Survey
	for _, survey := range allSurveys {
		if _, answered := answeredSurveyIds[survey.Id]; !answered {
			notAnsweredSurveys = append(notAnsweredSurveys, survey)
		}
	}

	// Use the not answered surveys as needed
	for _, survey := range notAnsweredSurveys {
		fmt.Println(survey)
	}

	return notAnsweredSurveys, nil
}
