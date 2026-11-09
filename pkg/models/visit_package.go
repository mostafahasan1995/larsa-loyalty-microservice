package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type VisitPackage struct {
	Id          primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name        LocalizableString  `bson:"name" json:"name"`
	Description LocalizableString  `bson:"description" json:"description"`
	Icon        []File             `bson:"icon" json:"icon"`
	Points      int64              `bson:"points" json:"points"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type VisitPackageWithPagination struct {
	VisitPackages []VisitPackage `bson:"visit_packages" json:"visit_packages"`
	Pagination    Pagination     `bson:"pagination" json:"pagination"`
}
