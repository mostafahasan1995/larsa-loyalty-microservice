package models

import (
	"loyalty/pkg/constants"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Quiz struct {
	Id          primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Title       LocalizableString  `bson:"title" json:"title"`
	Description LocalizableString  `bson:"description" json:"description"`
	Image       []File             `bson:"image" json:"image"`
	Questions   []Question         `bson:"questions" json:"questions"`
	Points      int64              `bson:"points" json:"points"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type Option struct {
	Id   primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Text LocalizableString  `bson:"text" json:"text"`
}

type Question struct {
	Id      primitive.ObjectID     `bson:"_id,omitempty" json:"_id,omitempty"`
	Title   LocalizableString      `bson:"title" json:"title"`
	Type    constants.QuestionType `bson:"type" json:"type"`
	Answer  LocalizableString      `bson:"answer" json:"answer"`
	Options []Option               `bson:"options" json:"options"`
}

type QuizResponse struct {
	Quiz      `bson:",inline" json:",inline"`
	Responses []QuizAnswer `bson:"responses,omitempty" json:"responses,omitempty"`
}

type QuizWithPagination struct {
	Quizzes    []QuizResponse `bson:"quizzes" json:"quizzes"`
	Pagination Pagination     `bson:"pagination" json:"pagination"`
}
