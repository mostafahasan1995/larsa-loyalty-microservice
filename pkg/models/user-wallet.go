package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserWallet struct {
	Id             primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Points         int64              `bson:"points" json:"points"`
	UserId         primitive.ObjectID `bson:"user_id" json:"user_id"`
	UserName       string             `bson:"user_name" json:"user_name"`
	Experience     int64              `bson:"experience" json:"experience"`
	UpdatedProfile bool               `bson:"updatedProfile" json:"updatedProfile"`
	VisitCount     int64              `bson:"visitCount" json:"visitCount"`
	Code           string             `bson:"code" json:"code"`
	CodeUsed       int64              `bson:"code_used" json:"code_used"`
	CodeSent       int64              `bson:"code_sent" json:"code_sent"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type UserWalletResponse struct {
	UserWallet `bson:",inline" json:",inline"`
	Level      *Level    `bson:"level" json:"level"`
	Vouchers   []Voucher `bson:"vouchers" json:"vouchers"`
}

type UserWalletBuyPointsRequest struct {
	MoneyAmount float64 `bson:"money_amount" json:"money_amount"`
}

type VerifyCodeRequest struct {
	Code string `bson:"code" json:"code"`
}

type UserWalletPointsPriceResponse struct {
	Points      int64   `bson:"points" json:"points"`
	MoneyAmount float64 `bson:"money_amount" json:"money_amount"`
}

type UserWalletWithPagination struct {
	UserWallets []UserWalletResponse `bson:"user_wallets" json:"user_wallets"`
	Pagination  Pagination           `bson:"pagination" json:"pagination"`
}
