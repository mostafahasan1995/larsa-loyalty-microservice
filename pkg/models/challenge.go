package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Challenge struct {
	Id         primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Title      LocalizableString  `bson:"title" json:"title"`
	Body       LocalizableString  `bson:"body" json:"body"`
	Icon       []File             `bson:"icon,omitempty" json:"icon,omitempty"`
	URL        string             `bson:"url" json:"url"`
	Points     int64              `bson:"points" json:"points"`
	IsCampaign bool               `bson:"is_campaign" json:"is_campaign"`
	//file text multi_choice
	Type    string            `bson:"type,omitempty" json:"type,omitempty"`
	Options []ChallengeOption `bson:"options,omitempty" json:"options,omitempty"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type ChallengeOption struct {
	Text LocalizableString `bson:"text" json:"text"`
}

type ChallengeResponse struct {
	Challenge `bson:",inline" json:",inline"`
	Responses []ChallengeAnswer `bson:"responses,omitempty" json:"responses,omitempty"`
}

type ChallengeWithPagination struct {
	Challenges []ChallengeResponse `bson:"challenges" json:"challenges"`
	Pagination Pagination          `bson:"pagination" json:"pagination"`
}
