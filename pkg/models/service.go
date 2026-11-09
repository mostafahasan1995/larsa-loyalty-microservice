package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	Id          primitive.ObjectID  `bson:"_id,omitempty" json:"_id,omitempty"`
	Name        LocalizableString   `bson:"name" json:"name"`
	Description LocalizableString   `bson:"description" json:"description"`
	Icon        []File              `bson:"icon" json:"icon"`
	Points      int64               `bson:"points" json:"points"`
	Group       *primitive.ObjectID `bson:"group" json:"group"`
	Level       *primitive.ObjectID `bson:"level,omitempty" json:"level,omitempty"`
	Discount    *Discount           `bson:"discount,omitempty" json:"discount,omitempty"`
	PartnerId   *primitive.ObjectID `bson:"partner_id" json:"partner_id"`
	Seq         int64               `bson:"seq,omitempty" json:"seq,omitempty"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type Discount struct {
	Type       string    `bson:"type" json:"type"`
	Value      float64   `bson:"value" json:"value"`
	ValidUntil time.Time `bson:"valid_until" json:"valid_until"`
}

type ServiceResponse struct {
	Service `bson:",inline" json:",inline"`
	Group   *ServiceGroup `bson:"group" json:"group"`
	Level   *Level        `bson:"level" json:"level"`
}

type RedeemRequest struct {
	ServiceId string `bson:"service_id" json:"service_id"`
	UserId    string `bson:"user_id" json:"user_id"`
}

type ServiceWithPagination struct {
	Services   []ServiceResponse `bson:"services" json:"services"`
	Pagination Pagination        `bson:"pagination" json:"pagination"`
}

//

type ServiceReorder struct {
	OldSeq int64 `json:"oldSeq"`
	NewSeq int64 `json:"newSeq"`
}
