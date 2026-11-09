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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// BulkDeleteResponse represents the response for bulk delete operations
type BulkDeleteResponse struct {
	TotalCount   int      `json:"totalCount"`
	SuccessCount int      `json:"successCount"`
	Errors       []string `json:"errors"`
}

type FeedbackService interface {
	GetOne(id string) (*models.Feedback, error)
	Get(args *util.ListArgs) (*models.FeedbackWithPagination, error)
	GetAll(query string) ([]models.Feedback, error)
	Submit(data *models.Feedback) (*models.Feedback, error)
	Delete(id string) error
	BulkDelete(ids []string) (*BulkDeleteResponse, error)
}

type feedbacksvcs struct {
	ctx      context.Context
	db       string
	collName string
}

func NewFeedbackSvcs(ctx context.Context, db string) FeedbackService {
	return &feedbacksvcs{
		ctx:      ctx,
		db:       db,
		collName: "loyaltyFeedbacks",
	}
}

func (w *feedbacksvcs) GetOne(id string) (*models.Feedback, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	_id, errId := primitive.ObjectIDFromHex(id)

	if errId != nil {
		return nil, errId
	}

	result := coll.FindOne(w.ctx, bson.M{"_id": _id})

	var data models.Feedback
	if decErr := result.Decode(&data); decErr != nil {
		return nil, decErr
	}

	return &data, nil
}

func (w *feedbacksvcs) Get(args *util.ListArgs) (*models.FeedbackWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	pipeline := PaginationPipe("feedbacks", args)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.FeedbackWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *feedbacksvcs) GetAll(query string) ([]models.Feedback, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filters, errQuery := getBsonFromString(query)
	if errQuery != nil {
		return nil, errQuery
	}

	cur, err := coll.Find(w.ctx, filters)

	if err != nil {
		return nil, err
	}

	var result []models.Feedback
	if err := cur.All(w.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (w *feedbacksvcs) Submit(data *models.Feedback) (*models.Feedback, error) {
	return WithTransaction(w.ctx, func(sessCtx mongo.SessionContext) (*models.Feedback, error) {
		coll := database.Mdb.Database(w.db).Collection(w.collName)

		data.Id = primitive.NewObjectID()
		_, err := coll.InsertOne(sessCtx, data)
		if err != nil {
			return nil, err
		}
		user := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)
		userId, err := primitive.ObjectIDFromHex(user.User.Id)
		if err != nil {
			return nil, err
		}

		settingsSvc := NewSettingsSvcs(sessCtx, w.db)
		points := settingsSvc.GetValueByKeyAsInt(constants.SettingsKeyDefaultFeedbackPoints, 50)

		transactionService := NewTransactionSvcs(sessCtx, w.db)
		newTransaction := &models.Transaction{
			UserId:   userId,
			Activity: fmt.Sprintf("Feedback submitted: %s", data.Id),
			Type:     constants.TXN_FEEDBACK_SUBMIT,
			TypeName: constants.TXN_FEEDBACK_SUBMIT.String(),
			Amount:   float64(points),
			RefId:    &data.Id,
		}
		newTransaction.SetCreatedBy(userId)
		_, err = transactionService.Add(newTransaction)
		if err != nil {
			return nil, err
		}
		walletSvc := NewUserWalletSvcs(sessCtx, w.db)
		wallet, err := walletSvc.GetWalletByUserId(user.User.Id)
		if err != nil {
			return nil, fmt.Errorf("failed to get wallet: %v", err)
		}
		_, err = walletSvc.AddPoints(wallet.Id.Hex(), points, points)
		if err != nil {
			return nil, fmt.Errorf("failed to add points: %v", err)
		}

		go NotifyDashboardOnFeedback(w.db, user.User.Id, user.User.FirstName, data.Id.Hex())

		return data, nil
	})
}

func (w *feedbacksvcs) Delete(id string) error {
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

func (w *feedbacksvcs) BulkDelete(ids []string) (*BulkDeleteResponse, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	response := &BulkDeleteResponse{
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
				response.Errors = append(response.Errors, fmt.Sprintf("Feedback with ID %s not found", id))
			} else {
				response.Errors = append(response.Errors, fmt.Sprintf("Error deleting feedback %s: %v", id, result.Err()))
			}
			continue
		}

		response.SuccessCount++

		// Audit trail is handled by middleware
	}

	return response, nil
}
