package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type QuizQuestionAnswer struct {
	QuestionId primitive.ObjectID `bson:"question_id" json:"question_id"`
	Answer     LocalizableString  `bson:"answer" json:"answer"`
}

type QuizAnswer struct {
	Id        primitive.ObjectID   `bson:"_id,omitempty" json:"_id,omitempty"`
	Answers   []QuizQuestionAnswer `bson:"answers" json:"answers"`
	UserId    primitive.ObjectID   `bson:"user_id" json:"user_id"`
	QuizId    primitive.ObjectID   `bson:"quiz_id" json:"quiz_id"`
	IsCorrect bool                 `bson:"is_correct" json:"is_correct"`
	Points    int64                `bson:"points" json:"points"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type QuizAnswerResponse struct {
	QuizAnswer `bson:",inline" json:",inline"`
	Quiz       Quiz `bson:"quiz" json:"quiz"`
}

type QuizAnswerWithPagination struct {
	QuizAnswers []QuizAnswerResponse `bson:"quiz_answers" json:"quiz_answers"`
	Pagination  Pagination           `bson:"pagination" json:"pagination"`
}
