package services

import (
	"context"
	"loyalty/pkg/database"
	"loyalty/pkg/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DbManagementService interface {
	UpdateMany(clientId string, operator models.DbOperator) (result *mongo.UpdateResult, err error)
	DeleteMany(clientId string, operator models.DbOperator) (result *mongo.DeleteResult, err error)
	GetInfo(clientId string) (interface{}, error)
}

type dbManagementSvc struct {
	db string
}

func NewDbManagementSvc(db string) DbManagementService {
	return &dbManagementSvc{db: db}
}

func (d *dbManagementSvc) UpdateMany(clientId string, operator models.DbOperator) (result *mongo.UpdateResult, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := database.Mdb.Database(clientId).Collection(operator.Collection)

	var opts options.UpdateOptions
	if operator.ArrayFilters != nil {
		var arrayFilters []interface{}
		for i := range operator.ArrayFilters {
			arrayFilters = append(arrayFilters, operator.ArrayFilters[i])
		}
		opts.SetArrayFilters(options.ArrayFilters{
			Filters: arrayFilters,
		})
	}
	if operator.UpdateAgg != nil {
		result, err = col.UpdateMany(ctx, operator.Filter, operator.UpdateAgg, &opts)
	} else {
		result, err = col.UpdateMany(ctx, operator.Filter, operator.Update, &opts)
	}
	return
}

func (d *dbManagementSvc) DeleteMany(clientId string, operator models.DbOperator) (result *mongo.DeleteResult, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := database.Mdb.Database(clientId).Collection(operator.Collection)

	result, err = col.DeleteMany(ctx, operator.Filter)
	return
}

func (d *dbManagementSvc) GetInfo(clientId string) (interface{}, error) {
	var result bson.M
	err := database.Mdb.Database(clientId).RunCommand(context.Background(), bson.D{bson.E{Key: "buildInfo", Value: 1}}).Decode(&result)
	return result, err
}
