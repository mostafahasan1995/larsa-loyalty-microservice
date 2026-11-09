package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type ChallengeAnswer struct {
	Id          primitive.ObjectID   `bson:"_id,omitempty" json:"_id,omitempty"`
	UserId      primitive.ObjectID   `bson:"user_id" json:"user_id"`
	ChallengeId primitive.ObjectID   `bson:"challenge_id" json:"challenge_id"`
	Answer      ChallengeAnswerValue `bson:"answer" json:"answer"`
	IsAccepted  bool                 `bson:"is_accepted" json:"is_accepted"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type ChallengeAnswerValue struct {
	Text  string `bson:"text" json:"text"`
	Files []File `bson:"files" json:"files"`
}

type ChallengeAnswerResponse struct {
	ChallengeAnswer `bson:",inline" json:",inline"`
	Challenge       Challenge `bson:"challenge" json:"challenge"`
}

type ChallengeAnswerWithPagination struct {
	ChallengeAnswers []ChallengeAnswerResponse `bson:"challenge_answers" json:"challenge_answers"`
	Pagination       Pagination                `bson:"pagination" json:"pagination"`
}
