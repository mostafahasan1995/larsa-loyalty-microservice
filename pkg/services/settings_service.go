package services

import (
	"context"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/database"
	"loyalty/pkg/models"
	"loyalty/pkg/util"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
)

type SettingsService interface {
	GetOne(key string) (*models.Settings, error)
	Get(args *util.ListArgs) (*models.SettingsWithPagination, error)
	GetAll(query string) ([]*models.Settings, error)
	GetValueByKey(key constants.SettingsKey) (string, error)
	GetValueByKeyAsInt(key constants.SettingsKey, fallbackValue int64) int64
	GetValueByKeyAsFloat(key constants.SettingsKey, fallbackValue float64) float64
	Add(data *models.Settings) (*models.Settings, error)
	Delete(id string) error
	Update(id string, data *models.Settings) (*models.Settings, error)
}

type Settingssvcs struct {
	MainService[*models.Settings, *models.Settings]
}

func NewSettingsSvcs(ctx context.Context, db string) SettingsService {
	return &Settingssvcs{MainService: MainService[*models.Settings, *models.Settings]{
		ctx:      ctx,
		db:       db,
		collName: "loyaltySettings",
	}}
}

func (w *Settingssvcs) Get(args *util.ListArgs) (*models.SettingsWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	pipeline := PaginationPipe("settings", args)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.SettingsWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *Settingssvcs) GetValueByKey(key constants.SettingsKey) (string, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	res := coll.FindOne(w.ctx, bson.M{"key": key, "trash": bson.M{"$ne": true}})
	var setting models.Settings
	if err := res.Decode(&setting); err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (w *Settingssvcs) GetValueByKeyAsInt(key constants.SettingsKey, fallbackValue int64) int64 {
	value, err := w.GetValueByKey(key)
	if err != nil {
		return fallbackValue
	}
	res, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallbackValue
	}
	return res
}

func (w *Settingssvcs) GetValueByKeyAsFloat(key constants.SettingsKey, fallbackValue float64) float64 {
	value, err := w.GetValueByKey(key)
	if err != nil {
		return fallbackValue
	}
	res, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallbackValue
	}
	return res
}
