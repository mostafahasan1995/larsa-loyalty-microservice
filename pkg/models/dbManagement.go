package models

import "go.mongodb.org/mongo-driver/bson"

type DbOperator struct {
	Operator     string   `json:"operator"`
	Collection   string   `json:"collection"`
	Filter       bson.M   `bson:"filter" json:"filter"`
	Update       bson.M   `bson:"update" json:"update"`
	UpdateAgg    bson.A   `bson:"updateAgg" json:"updateAgg"`
	ArrayFilters []bson.M `bson:"arrayFilters" json:"arrayFilters"`
}
