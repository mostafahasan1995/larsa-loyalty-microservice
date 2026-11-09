package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ServiceGroup struct {
	Id          primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name        LocalizableString  `bson:"name" json:"name"`
	Description LocalizableString  `bson:"description" json:"description"`
	Icon        []File             `bson:"icon" json:"icon"`
	IsSpecial   bool               `bson:"is_special" json:"is_special"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type ServiceGroupResponse struct {
	ServiceGroup `bson:",inline" json:",inline"`
	// Services      []Service `bson:"services" json:"services"`
	ServicesCount int64 `bson:"services_count" json:"services_count"`
}
type ServiceGroupWithPagination struct {
	ServiceGroups []ServiceGroupResponse `bson:"service_groups" json:"service_groups"`
	Pagination    Pagination             `bson:"pagination" json:"pagination"`
}
