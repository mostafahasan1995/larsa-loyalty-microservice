package models

import (
	"loyalty/pkg/constants"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Survey struct {
	Id        primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Title     LocalizableString  `bson:"title" json:"title"`
	Questions []SurveyQuestion   `bson:"questions" json:"questions"`
	Points    int64              `bson:"points" json:"points"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type SurveyQuestion struct {
	Id      primitive.ObjectID   `bson:"_id,omitempty" json:"_id,omitempty"`
	Title   LocalizableString    `bson:"title" json:"title"`
	Type    constants.SurveyType `bson:"type" json:"type"`
	Options []SurveyOption       `bson:"options" json:"options"`
}

type SurveyOption struct {
	Id   primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Text LocalizableString  `bson:"text" json:"text"`
}

type SurveyResponse struct {
	Survey    `bson:",inline" json:",inline"`
	Responses []SurveyAnswer `bson:"responses,omitempty" json:"responses,omitempty"`
}

type SurveyWithPagination struct {
	Surveys    []SurveyResponse `bson:"surveys" json:"surveys"`
	Pagination Pagination       `bson:"pagination" json:"pagination"`
}
