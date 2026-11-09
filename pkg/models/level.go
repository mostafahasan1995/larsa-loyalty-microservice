package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Level struct {
	Id         primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Num        int                `bson:"num" json:"num"`
	Experience int64              `bson:"experience" json:"experience"`
	Icon       []File             `bson:"icon" json:"icon"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type LevelWithPagination struct {
	Levels     []Level    `bson:"levels" json:"levels"`
	Pagination Pagination `bson:"pagination" json:"pagination"`
}
