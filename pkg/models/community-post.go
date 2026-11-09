package models

import (
	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CommunityPost struct {
	Id        primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Title     string             `bson:"title" json:"title"`
	Body      string             `bson:"body" json:"body"`
	Files     []File             `bson:"files,omitempty" json:"files,omitempty"`
	UserId    primitive.ObjectID `bson:"user_id" json:"user_id"`
	User      *common.User       `bson:"-" json:"user"`
	Reactions []Reaction         `bson:"reactions,omitempty" json:"reactions,omitempty"`
	IsHidden  bool               `bson:"is_hidden" json:"is_hidden"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type Reaction struct {
	Id     primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	UserId primitive.ObjectID `bson:"user_id" json:"user_id"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type CommunityPostWithPagination struct {
	CommunityPosts []CommunityPost `bson:"community_posts" json:"community_posts"`
	Pagination     Pagination      `bson:"pagination" json:"pagination"`
}

func (c *CommunityPost) SetUser(user *common.User) {
	c.User = user
}

func (c CommunityPost) GetUserId() string {
	return c.UserId.Hex()
}
