package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Settings struct {
	Id      primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Key     string             `bson:"key" json:"key"`
	Value   string             `bson:"value" json:"value"`
	Visible bool               `bson:"visible" json:"visible"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type SettingsWithPagination struct {
	Settings   []Settings `bson:"settings" json:"settings"`
	Pagination Pagination `bson:"pagination" json:"pagination"`
}
