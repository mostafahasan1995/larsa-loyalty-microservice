package models

import (
	"loyalty/pkg/constants"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Transaction struct {
	Id       primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	UserId   primitive.ObjectID `bson:"user_id" json:"user_id"`
	Activity string             `bson:"activity" json:"activity"`   // Description of the activity
	Type     constants.TxnType  `bson:"type" json:"type"`           // Type of transaction (e.g., "purchase", "quiz", "reward")
	TypeName string             `bson:"type_name" json:"type_name"` // Type of transaction (e.g., "purchase", "quiz", "reward")
	// DynamicTypeName interface{}         `bson:"dynamic_type_name,omitempty" json:"dynamic_type_name,omitempty"` // Dynamic type name for vouchers
	Amount float64             `bson:"amount" json:"amount"` // Amount associated with the transaction
	RefId  *primitive.ObjectID `bson:"ref_id,omitempty" json:"ref_id,,omitempty"`
	// VoucherDetails  *VoucherDetails     `bson:"voucher_details,omitempty" json:"voucher_details,omitempty"`

	BaseDbModel `bson:",inline" json:",inline"`
}

// type VoucherDetails struct {
// 	Id        primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
// 	ServiceId primitive.ObjectID `bson:"service_id" json:"service_id"`
// 	Points    int64              `bson:"points" json:"points"`
// 	Code      string             `bson:"code" json:"code"`
// 	Service   ServiceInfo        `bson:"service,omitempty" json:"service,omitempty"`
// }

type ServiceInfo struct {
	Id   primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name LocalizableString  `bson:"name" json:"name"`
}

type TransactionWithPagination struct {
	Transactions []Transaction `bson:"transactions" json:"transactions"`
	Pagination   Pagination    `bson:"pagination" json:"pagination"`
}
