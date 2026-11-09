package services

import (
	"context"
	"fmt"
	"log"
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
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type QuizzesService interface {
	GetOne(id string) (*models.QuizResponse, error)
	Get(args *util.ListArgs, userId *primitive.ObjectID, isAnswered bool) (*models.QuizWithPagination, error)
	GetAll(query string) ([]models.Quiz, error)
	Add(data *models.Quiz) error
	Update(id string, data *models.Quiz) error
	Delete(id string) error
	Submit(quizAnswer *models.QuizAnswer) (*models.QuizAnswerResponse, error)
	GetUserQuizzes(id string) ([]models.Quiz, error)
}

type quizsvcs struct {
	ctx      context.Context
	db       string
	collName string
}

func NewQuizSvcs(ctx context.Context, db string) QuizzesService {
	return &quizsvcs{
		ctx:      ctx,
		db:       db,
		collName: "loyaltyQuizzes",
	}
}

func isAnsweredQuizPipe(userId primitive.ObjectID, isAnswered bool) []bson.M {
	return []bson.M{{
		"$lookup": bson.M{
			"from": "loyaltyQuizAnswers",
			"let":  bson.M{"quizId": "$_id"},
			"pipeline": []bson.M{
				{"$match": bson.M{
					"$expr": bson.M{
						"$and": []bson.M{
							{"$eq": bson.A{"$quiz_id", "$$quizId"}},
							{"$eq": bson.A{"$user_id", userId}},
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

func (b *quizsvcs) GetOne(id string) (*models.QuizResponse, error) {
	coll := database.Mdb.Database(b.db).Collection(b.collName)

	_id, errId := primitive.ObjectIDFromHex(id)

	if errId != nil {
		return nil, errId
	}

	result := coll.FindOne(b.ctx, bson.M{"_id": _id})

	var data models.QuizResponse
	if decErr := result.Decode(&data); decErr != nil {
		return nil, decErr
	}

	return &data, nil
}

func (b *quizsvcs) Get(args *util.ListArgs, userId *primitive.ObjectID, isAnswered bool) (*models.QuizWithPagination, error) {
	coll := database.Mdb.Database(b.db).Collection(b.collName)

	extraPipe := []bson.M{}
	if userId != nil {
		extraPipe = isAnsweredQuizPipe(*userId, isAnswered)
	}
	pipeline := PaginationPipe("quizzes", args, extraPipe...)

	cursor, err := coll.Aggregate(b.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.QuizWithPagination
	if cursor.Next(b.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (b *quizsvcs) GetAll(query string) ([]models.Quiz, error) {
	coll := database.Mdb.Database(b.db).Collection(b.collName)

	filters, errQuery := getBsonFromString(query)
	if errQuery != nil {
		return nil, errQuery
	}

	cur, err := coll.Find(b.ctx, filters)

	if err != nil {
		return nil, err
	}

	var result []models.Quiz
	if err := cur.All(b.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (b *quizsvcs) Add(data *models.Quiz) error {
	user := b.ctx.Value(constants.CtxAuthKey).(common.Credentials)
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
	defer session.EndSession(b.ctx)

	_, errTnx := session.WithTransaction(b.ctx, func(ctx mongo.SessionContext) (interface{}, error) {
		quizcoll := database.Mdb.Database(b.db).Collection(b.collName)

		if len(data.Questions) == 0 {
			return nil, fmt.Errorf("questions is required")
		}

		//quizId := primitive.NewObjectID()
		var questions []models.Question
		for _, q := range data.Questions {
			if len(q.Options) == 0 {
				return nil, fmt.Errorf("question must have options")
			}

			var ops []models.Option
			for _, o := range q.Options {
				ops = append(ops, models.Option{Id: primitive.NewObjectID(), Text: o.Text})
			}
			// save options
			q.Options = ops

			//question, err := questioncoll.InsertOne(b.ctx, q)
			//if err != nil {
			//	return nil, fmt.Errorf("failed to insert the question")
			//}

			questions = append(questions, models.Question{
				Id:      primitive.NewObjectID(),
				Title:   q.Title,
				Type:    q.Type,
				Answer:  q.Answer,
				Options: q.Options,
			})
		}

		data.Questions = questions
		//data.Id = quizId

		res, err := quizcoll.InsertOne(b.ctx, data)
		if err != nil {
			return nil, fmt.Errorf("failed to insert quiz")
		}
		data.Id, _ = res.InsertedID.(primitive.ObjectID)
		return data, nil
	}, txnOptions)

	if errTnx != nil {
		return errTnx
	}

	return nil
}

func (b *quizsvcs) Update(id string, data *models.Quiz) error {

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

func (b *quizsvcs) Delete(id string) error {
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

func (b *quizsvcs) Submit(answer *models.QuizAnswer) (*models.QuizAnswerResponse, error) {
	quiz, err := b.GetOne(answer.QuizId.Hex())
	if err != nil {
		return nil, fmt.Errorf("quiz not found %w", err)
	}
	result, err := WithTransaction(b.ctx, func(sessCtx mongo.SessionContext) (*models.QuizAnswer, error) {
		// check if user answered this quiz before
		quizAnswerSvc := NewQuizAnswerSvcs(sessCtx, b.db)
		_, err = quizAnswerSvc.GetOneByUserId(answer.QuizId, answer.UserId)
		if err == nil {
			return nil, fmt.Errorf("user has answered this quiz before")
		}
		answersMap := lo.KeyBy(answer.Answers, func(a models.QuizQuestionAnswer) string {
			return a.QuestionId.Hex()
		})
		isCorrect := true

		for _, q := range quiz.Questions {
			if answer, exist := answersMap[q.Id.Hex()]; !exist || !IsAnswerCorrect(answer, q) {
				isCorrect = false
				break
			}
		}
		transactionService := NewTransactionSvcs(sessCtx, b.db)

		newTransaction := &models.Transaction{
			UserId:   answer.UserId,
			Activity: "Quiz submitted",
			Type:     constants.TXN_QUIZ_SUBMIT,
			TypeName: constants.TXN_QUIZ_SUBMIT.String(),
			Amount:   float64(quiz.Points),
			RefId:    &quiz.Id,
		}
		newTransaction.SetCreatedBy(answer.UserId)

		_, err = transactionService.Add(newTransaction)
		if err != nil {
			return nil, err
		}
		if isCorrect {
			userWalletService := NewUserWalletSvcs(sessCtx, b.db)
			wallet, _ := userWalletService.GetWalletByUserId(answer.UserId.Hex())

			updatedWallet, err := userWalletService.AddPoints(wallet.Id.Hex(), quiz.Points, quiz.Points)
			if err != nil {
				return nil, err
			}
			wallet.UserWallet = *updatedWallet
			answer.Points = quiz.Points
		}
		answer.IsCorrect = isCorrect
		err := quizAnswerSvc.Add(answer)
		if err != nil {
			return nil, err
		}

		return answer, nil
	})
	return &models.QuizAnswerResponse{QuizAnswer: *result, Quiz: quiz.Quiz}, err
}

func (b *quizsvcs) GetUserQuizzes(id string) ([]models.Quiz, error) {

	quizColl := database.Mdb.Database(b.db).Collection(b.collName)
	quizAnswerColl := database.Mdb.Database(b.db).Collection("loyaltyquizanswer")

	userId, errId := primitive.ObjectIDFromHex(id)
	if errId != nil {
		log.Fatal(errId)
	}

	// Step 1: Retrieve all quizzes
	allQuizCursor, err := quizColl.Find(b.ctx, bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	defer allQuizCursor.Close(b.ctx)

	var allQuizzes []models.Quiz
	if err := allQuizCursor.All(b.ctx, &allQuizzes); err != nil {
		log.Fatal(err)
	}

	// Step 2: Retrieve the user's quiz answers
	userAnswersCursor, err := quizAnswerColl.Find(b.ctx, bson.M{"user_id": userId})
	if err != nil {
		log.Fatal(err)
	}
	defer userAnswersCursor.Close(b.ctx)

	var userQuizAnswers []models.QuizAnswer
	if err := userAnswersCursor.All(b.ctx, &userQuizAnswers); err != nil {
		log.Fatal(err)
	}

	// Create a map to track answered questions by quiz
	userAnswersMap := make(map[primitive.ObjectID]map[primitive.ObjectID]struct{})
	for _, quizAnswer := range userQuizAnswers {
		if userAnswersMap[quizAnswer.QuizId] == nil {
			userAnswersMap[quizAnswer.QuizId] = make(map[primitive.ObjectID]struct{})
		}
		for _, answer := range quizAnswer.Answers {
			userAnswersMap[quizAnswer.QuizId][answer.QuestionId] = struct{}{}
		}
	}

	// Step 3: Identify the quizzes the user has not fully answered
	var notFullyAnsweredQuizzes []models.Quiz
	for _, quiz := range allQuizzes {
		allQuestionsAnswered := true
		for _, question := range quiz.Questions {
			if _, answered := userAnswersMap[quiz.Id][question.Id]; !answered {
				allQuestionsAnswered = false
				break
			}
		}
		if !allQuestionsAnswered {
			notFullyAnsweredQuizzes = append(notFullyAnsweredQuizzes, quiz)
		}
	}

	// Use the not fully answered quizzes as needed
	for _, quiz := range notFullyAnsweredQuizzes {
		fmt.Println(quiz)
	}

	return notFullyAnsweredQuizzes, nil
}

func IsAnswerCorrect(answer models.QuizQuestionAnswer, question models.Question) bool {
	return answer.Answer.Ar != "" && answer.Answer.Ar == question.Answer.Ar ||
		answer.Answer.En != "" && answer.Answer.En == question.Answer.En
}
