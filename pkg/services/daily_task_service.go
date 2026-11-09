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
	"github.com/samber/lo"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DailyTaskService interface {
	GetOne(id string) (*models.DailyTaskResponse, error)
	Get(args *util.ListArgs, userId *primitive.ObjectID) (*models.DailyTaskWithPagination, error)
	GetAll(query string) ([]models.DailyTask, error)
	GetMonthlyTasksData(userId primitive.ObjectID, month, year int64) ([]models.DailyTasksWithDate, error)
	GetYearlyTasksData(userId primitive.ObjectID, year int64) ([]models.DailyTasksWithDate, error)
	Add(data *models.DailyTask) error
	Update(id string, data *models.DailyTask) error
	Delete(id string) error
	Interact(interact *models.DailyTaskInteraction) (*models.DailyTaskInteraction, error)
}

type dailyTasksvcs struct {
	ctx      context.Context
	db       string
	collName string
}

func NewDailyTaskSvcs(ctx context.Context, db string) DailyTaskService {
	return &dailyTasksvcs{
		ctx:      ctx,
		db:       db,
		collName: "loyaltyDailyTasks",
	}
}

func isAnsweredDailyTaskPipe(userId primitive.ObjectID, startDate, endDate time.Time) []bson.M {
	return []bson.M{{
		"$lookup": bson.M{
			"from": "loyaltyDailyTaskInteractions",
			"let":  bson.M{"taskId": "$_id"},
			"pipeline": []bson.M{
				{"$match": bson.M{
					"$expr": bson.M{
						"$and": []bson.M{
							{"$eq": bson.A{"$task_id", "$$taskId"}},
							{"$eq": bson.A{"$user_id", userId}},
							{"$gte": bson.A{"$createdAt", startDate}},
							{"$lte": bson.A{"$createdAt", endDate}},
						},
					},
				}},
			},
			"as": "interactions",
		}},
		{"$addFields": bson.M{
			"interaction": bson.M{
				"$arrayElemAt": bson.A{"$interactions", 0},
			},
		}},
		{"$addFields": bson.M{
			"is_completed": bson.M{
				"$cond": bson.M{
					"if":   bson.M{"$gt": bson.A{"$interactions", nil}},
					"then": "$interaction.is_completed",
					"else": false,
				},
			},
		},
		},
	}

}

func (w *dailyTasksvcs) GetOne(id string) (*models.DailyTaskResponse, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	_id, errId := primitive.ObjectIDFromHex(id)

	if errId != nil {
		return nil, errId
	}
	filterStage := bson.M{"$match": bson.M{"_id": _id}}

	now := time.Now()
	year, month, day := now.Date()
	// Start of today
	start := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
	// End of today (technically tomorrow's 00:00:00)
	end := start.AddDate(0, 0, 1)
	user := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)
	userId, _ := primitive.ObjectIDFromHex(user.User.Id)
	extraPipe := isAnsweredDailyTaskPipe(userId, start, end)
	pipeline := append([]bson.M{filterStage}, extraPipe...)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.DailyTaskResponse
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
	}
	return &result, nil
}

func (w *dailyTasksvcs) Get(args *util.ListArgs, userId *primitive.ObjectID) (*models.DailyTaskWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	extraPipe := []bson.M{}
	if userId != nil {
		now := time.Now()
		year, month, day := now.Date()
		// Start of today
		start := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
		// End of today (technically tomorrow's 00:00:00)
		end := start.AddDate(0, 0, 1)
		extraPipe = isAnsweredDailyTaskPipe(*userId, start, end)
	}
	pipeline := PaginationPipe("daily_tasks", args, extraPipe...)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.DailyTaskWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *dailyTasksvcs) GetMonthlyTasksData(userId primitive.ObjectID, month, year int64) ([]models.DailyTasksWithDate, error) {
	// Convert month and year to a date range
	startDate := time.Date(int(year), time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)

	// Step 1: Retrieve all tasks
	coll := database.Mdb.Database(w.db).Collection(w.collName)
	cursor, err := coll.Find(w.ctx, bson.M{"trash": bson.M{"$ne": true}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(w.ctx)

	var tasks []models.DailyTask
	if err = cursor.All(w.ctx, &tasks); err != nil {
		return nil, err
	}

	// Step 2: Retrieve task interactions for the user for the specified month
	taskInteractionSvc := NewDailyTaskInteractionSvcs(w.ctx, w.db)
	taskInteractions, err := taskInteractionSvc.GetByUserId(userId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	loc, _ := time.LoadLocation("Asia/Dubai")
	// Step 3: Organize task interactions by date
	interactionsByDate := lo.GroupBy(taskInteractions, func(interaction models.DailyTaskInteraction) string {
		return interaction.CreatedAt.In(loc).Format("2006-01-02")
	})

	// Step 4: Create response structure
	var response []models.DailyTasksWithDate
	for day := 1; day <= endDate.Day(); day++ {
		currentDate := startDate.AddDate(0, 0, day-1).In(loc).Format("2006-01-02")
		interactions := interactionsByDate[currentDate]

		var dailyTasks []models.DailyTaskResponse
		for _, task := range tasks {
			interaction, found := lo.Find(interactions, func(i models.DailyTaskInteraction) bool {
				return i.TaskId == task.Id
			})

			isComplete := false
			if found {
				isComplete = interaction.IsCompleted
			}

			dailyTasks = append(dailyTasks, models.DailyTaskResponse{
				DailyTask:   task,
				IsCompleted: &isComplete,
			})
		}

		response = append(response, models.DailyTasksWithDate{
			Date:  currentDate,
			Tasks: dailyTasks,
		})
	}

	return response, nil
}

func (w *dailyTasksvcs) GetYearlyTasksData(userId primitive.ObjectID, year int64) ([]models.DailyTasksWithDate, error) {
	// Convert year to a date range
	startDate := time.Date(int(year), time.Month(1), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(1, 0, -1)

	// Step 1: Retrieve all tasks
	coll := database.Mdb.Database(w.db).Collection(w.collName)
	cursor, err := coll.Find(w.ctx, bson.M{"trash": bson.M{"$ne": true}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(w.ctx)

	var tasks []models.DailyTask
	if err = cursor.All(w.ctx, &tasks); err != nil {
		return nil, err
	}

	// Step 2: Retrieve task interactions for the user for the specified date range
	taskInteractionSvc := NewDailyTaskInteractionSvcs(w.ctx, w.db)
	taskInteractions, err := taskInteractionSvc.GetByUserId(userId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	loc, _ := time.LoadLocation("Asia/Dubai")
	// Step 3: Organize task interactions by date
	interactionsByDate := lo.GroupBy(taskInteractions, func(interaction models.DailyTaskInteraction) string {
		return interaction.CreatedAt.In(loc).Format("2006-01-02")
	})

	// Step 4: Create response structure
	var response []models.DailyTasksWithDate
	for day := 1; day <= endDate.YearDay(); day++ {
		currentDate := startDate.AddDate(0, 0, day-1).In(loc).Format("2006-01-02")
		interactions := interactionsByDate[currentDate]

		var dailyTasks []models.DailyTaskResponse
		for _, task := range tasks {
			interaction, found := lo.Find(interactions, func(i models.DailyTaskInteraction) bool {
				return i.TaskId == task.Id
			})

			isComplete := false
			if found {
				isComplete = interaction.IsCompleted
			}

			dailyTasks = append(dailyTasks, models.DailyTaskResponse{
				DailyTask:   task,
				IsCompleted: &isComplete,
			})
		}

		response = append(response, models.DailyTasksWithDate{
			Date:  currentDate,
			Tasks: dailyTasks,
		})
	}

	return response, nil
}

func (w *dailyTasksvcs) GetAll(query string) ([]models.DailyTask, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	filters, errQuery := getBsonFromString(query)
	if errQuery != nil {
		return nil, errQuery
	}

	cur, err := coll.Find(w.ctx, filters)

	if err != nil {
		return nil, err
	}

	var result []models.DailyTask
	if err := cur.All(w.ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (w *dailyTasksvcs) Add(data *models.DailyTask) error {
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

func (w *dailyTasksvcs) Delete(id string) error {
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

func (b *dailyTasksvcs) Update(id string, data *models.DailyTask) error {
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

func (b *dailyTasksvcs) Interact(interact *models.DailyTaskInteraction,
) (*models.DailyTaskInteraction, error) {
	_, err := b.GetOne(interact.TaskId.Hex())
	if err != nil {
		return nil, fmt.Errorf("daily_task not found %w", err)
	}
	return WithTransaction(b.ctx, func(sessCtx mongo.SessionContext) (*models.DailyTaskInteraction, error) {
		dailyTaskInteractionSvc := NewDailyTaskInteractionSvcs(sessCtx, b.db)
		startDate, endDate := util.StartAndEndDay(time.Now())
		filter := bson.M{
			"user_id": interact.UserId,
			"task_id": interact.TaskId,
			"createdAt": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
		}
		err := dailyTaskInteractionSvc.Upsert(filter, interact)
		if err != nil {
			return nil, err
		}
		// add points to wallet
		settingsService := NewSettingsSvcs(sessCtx, b.db)
		points := settingsService.GetValueByKeyAsInt(constants.SettingsKeyDefaultDailyTaskPoints, 1)
		walletService := NewUserWalletSvcs(sessCtx, b.db)
		wallet, err := walletService.GetWalletByUserId(interact.UserId.Hex())
		if err != nil {
			return nil, fmt.Errorf("failed to get wallet %w", err)
		}
		_, err = walletService.AddPoints(wallet.Id.Hex(), points, points)
		if err != nil {
			return nil, fmt.Errorf("failed to add points to wallet %w", err)
		}
		user := b.ctx.Value(constants.CtxAuthKey).(common.Credentials)
		userId, _ := primitive.ObjectIDFromHex(user.User.Id)
		transactionService := NewTransactionSvcs(sessCtx, b.db)
		newTransaction := &models.Transaction{
			UserId:   interact.UserId,
			Activity: "Daily task interacted",
			Type:     constants.TXN_DAILY_TASK_INTERACTED,
			TypeName: constants.TXN_DAILY_TASK_INTERACTED.String(),
			Amount:   float64(points),
			RefId:    &interact.TaskId,
		}
		newTransaction.SetCreatedBy(userId)
		_, err = transactionService.Add(newTransaction)
		if err != nil {
			return nil, err
		}

		return interact, nil
	})
}
