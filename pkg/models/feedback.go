package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Feedback struct {
	Id          primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Suggestion  string             `bson:"suggestion" json:"suggestion"`
	IsAnonymous bool               `bson:"is_anonymous" json:"is_anonymous"`
	PatientName string             `bson:"patient_name" json:"patient_name"`
	UserId      primitive.ObjectID `bson:"user_id" json:"user_id"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type FeedbackWithPagination struct {
	Feedbacks  []Feedback `bson:"feedbacks" json:"feedbacks"`
	Pagination Pagination `bson:"pagination" json:"pagination"`
}
