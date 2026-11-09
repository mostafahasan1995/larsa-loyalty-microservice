package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SurveyQuestionAnswer struct {
	QuestionId primitive.ObjectID `bson:"question_id" json:"question_id"`
	Answer     string             `bson:"answer" json:"answer"`
	Emoji      string             `bson:"emoji" json:"emoji"`
}

type SurveyAnswer struct {
	Id         primitive.ObjectID     `bson:"_id,omitempty" json:"_id,omitempty"`
	Answers    []SurveyQuestionAnswer `bson:"answers" json:"answers"`
	UserId     primitive.ObjectID     `bson:"user_id" json:"user_id"`
	SurveyId   primitive.ObjectID     `bson:"survey_id" json:"survey_id"`
	Points     int64                  `bson:"points" json:"points"`
	IsArchived bool                   `bson:"is_archived" json:"is_archived"`
	Visit      string                 `bson:"visit" json:"visit"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type SurveyAnswerResponse struct {
	SurveyAnswer `bson:",inline" json:",inline"`
	Survey       Survey `bson:"survey" json:"survey"`
}

type SurveyAnswerWithPagination struct {
	SurveyAnswers []SurveyAnswerResponse `bson:"survey_answers" json:"survey_answers"`
	Pagination    Pagination             `bson:"pagination" json:"pagination"`
}

type SurveyAnswerWithGroupVisit struct {
	SurveyAnswers []SurveyAnswerResponse `bson:"answers" json:"answers"`
	Id            string                 `bson:"_id" json:"_id"`
}
