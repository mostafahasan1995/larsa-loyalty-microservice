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

type CommunityPostService interface {
	GetOne(id string) (*models.CommunityPost, error)
	Get(args *util.ListArgs, userId *primitive.ObjectID) (*models.CommunityPostWithPagination, error)
	GetAll(query string) ([]*models.CommunityPost, error)
	Add(data *models.CommunityPost) (*models.CommunityPost, error)
	Update(id string, data *models.CommunityPost) (*models.CommunityPost, error)
	Delete(id string) error
	React(id string) (*models.CommunityPost, error)
	UnReact(id string) (*models.CommunityPost, error)
}
type CommunityPostSvc struct {
	MainService[*models.CommunityPost, *models.CommunityPost]
}

func NewCommunityPostSvc(ctx context.Context, db string) CommunityPostService {
	return &CommunityPostSvc{
		MainService: MainService[*models.CommunityPost, *models.CommunityPost]{
			ctx:      ctx,
			db:       db,
			collName: "loyaltyCommunityPosts",
		}}
}

func (w *CommunityPostSvc) Get(args *util.ListArgs, userId *primitive.ObjectID) (*models.CommunityPostWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	if userId != nil {
		args.Query["user_id"] = userId
	}
	pipeline := PaginationPipe("community_posts", args)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.CommunityPostWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		userIds := lo.Map(result.CommunityPosts, func(post models.CommunityPost, _ int) string {
			return post.GetUserId()
		})
		users, _ := util.FetchUsers(userIds, w.db)
		res := util.MapUsersToData(users, lo.ToSlicePtr(result.CommunityPosts))
		result.CommunityPosts = util.FromSlicePtr(res)
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *CommunityPostSvc) GetOne(id string) (*models.CommunityPost, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	_id, errId := primitive.ObjectIDFromHex(id)

	if errId != nil {
		return nil, errId
	}
	filterStage := bson.M{"$match": bson.M{"_id": _id}}
	pipeline := append([]bson.M{filterStage}, w.MongoPipe...)
	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result *models.CommunityPost
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		userIds := []string{result.GetUserId()}
		users, _ := util.FetchUsers(userIds, w.db)
		if len(users) > 0 {
			result.SetUser(users[0])
		}
		return result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *CommunityPostSvc) Add(data *models.CommunityPost) (*models.CommunityPost, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	user := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)
	userId, _ := primitive.ObjectIDFromHex(user.User.Id)
	return WithTransaction(w.ctx, func(sessCtx mongo.SessionContext) (*models.CommunityPost, error) {
		// // check if previous post created from 1 day before
		// previousPostRes := coll.FindOne(sessCtx,
		// 	bson.M{"user_id": userId, "trash": bson.M{"$ne": true}},
		// 	options.FindOne().SetSort(bson.M{"_id": -1}),
		// )
		// previousPost := &models.CommunityPost{}
		// if err := previousPostRes.Decode(previousPost); err == nil {
		// 	if previousPost.CreatedAt.AddDate(0, 0, 1).After(time.Now()) {
		// 		return nil, fmt.Errorf("you can only create a new post once a day")
		// 	}
		// }
		// add points to wallet
		settingsSvc := NewSettingsSvcs(sessCtx, w.db)
		points := settingsSvc.GetValueByKeyAsInt(constants.SettingsKeyDefaultCommunityPoints, 10)

		walletSvc := NewUserWalletSvcs(sessCtx, w.db)
		wallet, err := walletSvc.GetWalletByUserId(user.User.Id)
		if err != nil {
			return nil, fmt.Errorf("failed to get wallet: %v", err)
		}
		_, err = walletSvc.AddPoints(wallet.Id.Hex(), points, points)
		if err != nil {
			return nil, fmt.Errorf("failed to add points to wallet: %v", err)
		}
		data.SetCreatedBy(userId)
		res, err := coll.InsertOne(sessCtx, data)
		if err != nil {
			return nil, err
		}
		data.Id = res.InsertedID.(primitive.ObjectID)

		transactionService := NewTransactionSvcs(sessCtx, w.db)
		newTransaction := &models.Transaction{
			UserId:   data.UserId,
			Activity: "Post added",
			Type:     constants.TXN_POST_ADDED,
			TypeName: constants.TXN_POST_ADDED.String(),
			Amount:   float64(points),
			RefId:    &data.Id,
		}
		newTransaction.SetCreatedBy(userId)
		_, err = transactionService.Add(newTransaction)
		if err != nil {
			return nil, err
		}

		return data, nil
	})
}

func (w *CommunityPostSvc) React(id string) (*models.CommunityPost, error) {
	_id, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return nil, err
	}
	coll := database.Mdb.Database(w.db).Collection(w.collName)
	user := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)
	userId, _ := primitive.ObjectIDFromHex(user.User.Id)

	filter := bson.M{"_id": _id, "reactions.user_id": bson.M{"$ne": userId}}
	reaction := models.Reaction{
		Id:     primitive.NewObjectID(),
		UserId: userId,
	}
	update := bson.M{"$push": bson.M{"reactions": reaction}}
	reaction.SetCreatedBy(userId)

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(false)
	result := coll.FindOneAndUpdate(w.ctx, filter, update, opts)
	if result.Err() != nil {
		return nil, result.Err()
	}
	var post models.CommunityPost
	if err := result.Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

func (w *CommunityPostSvc) UnReact(id string) (*models.CommunityPost, error) {
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	coll := database.Mdb.Database(w.db).Collection(w.collName)
	user := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)
	userId, _ := primitive.ObjectIDFromHex(user.User.Id)

	filter := bson.M{"_id": _id, "reactions.user_id": userId}
	update := bson.M{"$pull": bson.M{"reactions": bson.M{"user_id": userId}}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(false)
	result := coll.FindOneAndUpdate(w.ctx, filter, update, opts)
	if result.Err() != nil {
		return nil, result.Err()
	}
	var post models.CommunityPost
	if err := result.Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}
