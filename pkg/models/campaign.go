package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Campaign struct {
	Id           primitive.ObjectID    `bson:"_id,omitempty" json:"_id,omitempty"`
	Title        string                `bson:"title" json:"title"`
	Description  string                `bson:"description" json:"description"`
	Image        []File                `bson:"image" json:"image"`
	Activities   []CampaignActivity    `bson:"activities" json:"activities"`
	Interactions []CampaignInteraction `bson:"interactions" json:"interactions"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type CampaignActivity struct {
	Id     primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Title  string             `bson:"title" json:"title"`
	Points int64              `bson:"points" json:"points"`
}

type CampaignInteraction struct {
	ActivityId primitive.ObjectID `bson:"activity_id" json:"activity_id"`
	MediaUrl   string             `bson:"media_url" json:"media_url"`
	UserId     primitive.ObjectID `bson:"user_id" json:"user_id"`
}

type CampaignWithPagination struct {
	Campaigns  []Campaign `bson:"campaigns" json:"campaigns"`
	Pagination Pagination `bson:"pagination" json:"pagination"`
}
