package models

import (
	"time"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Voucher struct {
	Id             primitive.ObjectID  `bson:"_id,omitempty" json:"_id,omitempty"`
	ServiceId      primitive.ObjectID  `bson:"service_id" json:"service_id"`
	PartnerId      *primitive.ObjectID `bson:"partner_id,omitempty" json:"partner_id,omitempty"`
	Partner        *common.User        `bson:"-" json:"partner"`
	UserId         primitive.ObjectID  `bson:"user_id" json:"user_id"`
	Points         int64               `bson:"points" json:"points"`
	Discount       *Discount           `bson:"discount,omitempty" json:"discount,omitempty"`
	IsUsed         bool                `bson:"is_used" json:"is_used"`
	Code           string              `bson:"code" json:"code"`
	ExpiredAt      time.Time           `bson:"expired_at" json:"expired_at"`
	ExpiryNotified bool                `bson:"expiry_notified" json:"expiry_notified"`
	Status         string              `bson:"status,omitempty" json:"status,omitempty"`
	Comments       string              `bson:"comments,omitempty" json:"comments"`
	BaseDbModel    `bson:",inline" json:",inline"`
}

type VoucherResponse struct {
	Voucher       `bson:",inline" json:",inline"`
	Service       Service `bson:"service" json:"service"`
	VoucherStatus string  `bson:"-" json:"voucher_status,omitempty"` // Computed field: "active", "used", "expired"
}

// GetVoucherStatus computes the voucher status based on is_used and expired_at fields
func (v *VoucherResponse) GetVoucherStatus() string {
	if v.IsUsed {
		return "used"
	}

	if !v.ExpiredAt.IsZero() && v.ExpiredAt.Before(time.Now()) {
		return "expired"
	}

	return "active"
}

type VoucherWithPagination struct {
	Vouchers   []VoucherResponse `bson:"vouchers" json:"vouchers"`
	Pagination Pagination        `bson:"pagination" json:"pagination"`
}

// Voucher status constants
const (
	VoucherStatusPending    = "pending"
	VoucherStatusChecked    = "checked"
	VoucherStatusInProgress = "in_progress"
	VoucherStatusCanceled   = "canceled"
	VoucherStatusBooked     = "booked"
	VoucherStatusNotBooked  = "not_booked"
)

// Enhanced voucher response with user details for management interface
type VoucherManagementResponse struct {
	VoucherResponse `bson:",inline" json:",inline"`
	User            *VoucherUserInfo `bson:"user,omitempty" json:"user,omitempty"`
	ServiceGroup    *ServiceGroup    `bson:"service_group,omitempty" json:"service_group,omitempty"`
	VoucherStatus   string           `bson:"-" json:"voucher_status,omitempty"` // Computed field: "active", "used", "expired"
	RedeemAt        *time.Time       `bson:"-" json:"redeem_at,omitempty"`      // Timestamp when voucher was redeemed (only for used status)
}

// GetVoucherStatus computes the voucher status based on is_used and expired_at fields
// and populates RedeemAt field when status is "used"
func (v *VoucherManagementResponse) GetVoucherStatus() string {
	if v.IsUsed {
		// Set RedeemAt to UpdatedAt when voucher is used
		if !v.UpdatedAt.IsZero() {
			v.RedeemAt = &v.UpdatedAt
		}
		return "used"
	}

	// For non-used statuses, ensure RedeemAt is null
	v.RedeemAt = nil

	if !v.ExpiredAt.IsZero() && v.ExpiredAt.Before(time.Now()) {
		return "expired"
	}

	return "active"
}

type VoucherUserInfo struct {
	Id        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

type VoucherManagementWithPagination struct {
	Vouchers   []VoucherManagementResponse `bson:"vouchers" json:"vouchers"`
	Pagination Pagination                  `bson:"pagination" json:"pagination"`
}

func (c *Voucher) SetPartner(user *common.User) {
	c.Partner = user
}

func (c Voucher) GetPartnerId() string {
	if c.PartnerId != nil {
		return c.PartnerId.Hex()
	}
	return ""
}
