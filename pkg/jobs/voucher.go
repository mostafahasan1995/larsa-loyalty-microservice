package jobs

import (
	"context"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/models"
	"loyalty/pkg/services"
	"loyalty/pkg/util"
	"time"

	"github.com/go-co-op/gocron/v2"
	"go.mongodb.org/mongo-driver/bson"
)

func InitNotifyOldVouchersDailyJob(ctx context.Context, scheduler gocron.Scheduler) {
	_, _ = scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(10, 0, 0))),
		gocron.NewTask(func(ctx context.Context) {
			fmt.Println("Notify old vouchers daily job started")
			dbs := util.RelevantDbs()
			for _, db := range dbs {
				voucherService := services.NewVoucherSvc(ctx, db)
				// get vouchers which created at 30 days before not more or less
				args := util.ListArgs{
					Query: bson.M{
						"createdAt": bson.M{
							"$gte": time.Now().Add(-30 * 24 * time.Hour),
							"$lt":  time.Now().Add(-29 * 24 * time.Hour),
						},
						"trash":   bson.M{"$ne": true},
						"is_used": bson.M{"$ne": true},
					},
					PerPage: "10000",
					Page:    "1",
					Sort:    bson.M{"createdAt": -1},
				}
				res, _ := voucherService.Get(&args, nil, nil)
				for _, voucher := range res.Vouchers {
					util.LocalizeNotify(
						util.LocalizeNotificationMsg{
							XClient: db,
							Header: models.LocalizableString{
								Ar: "تنبيه لقسيمة قديمة",
								En: "Notice for old voucher!",
							},
							Content: models.LocalizableString{
								Ar: fmt.Sprintf("الرجاء استخدام القسيمة الخاصة بالخدمة: %s", voucher.Service.Name.Localize("ar")),
								En: fmt.Sprintf("Please use your Voucher for service: %s", voucher.Service.Name.Localize("en")),
							},
							TopicType: constants.NotificationTopicTypeUser,
							Topic:     voucher.UserId.Hex(),
							Type:      "fcm",
							Data: util.NotificationPayload{
								Id:    voucher.Id.Hex(),
								Model: "voucher",
							},
						},
					)
				}
			}
		}, ctx),
	)
}

func InitNotifyExpiredVouchersDailyJob(ctx context.Context, scheduler gocron.Scheduler) {
	_, _ = scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(10, 0, 0))),
		gocron.NewTask(func(ctx context.Context) {
			fmt.Println("Notify expired vouchers job started")
			dbs := util.RelevantDbs()

			for _, db := range dbs {
				voucherService := services.NewVoucherSvc(ctx, db)
				// get vouchers that have expired (current time > expiry time)
				args := util.ListArgs{
					Query: bson.M{
						"expired_at": bson.M{
							"$lt": time.Now(),
						},
						"trash":   bson.M{"$ne": true},
						"is_used": bson.M{"$ne": true},
						// Only get vouchers that haven't been notified yet
						"expiry_notified": bson.M{"$ne": true},
					},
					PerPage: "10000",
					Page:    "1",
					Sort:    bson.M{"expired_at": -1},
				}

				res, err := voucherService.Get(&args, nil, nil)
				if err != nil {
					fmt.Println("🚀 ~ gocron.NewTask ~ err:", err)
					continue
				}

				for _, voucher := range res.Vouchers {
					go util.LocalizeNotify(
						util.LocalizeNotificationMsg{
							XClient: db,
							Header: models.LocalizableString{
								Ar: "قسيمة منتهية الصلاحية",
								En: "Expired Voucher Notice",
							},
							Content: models.LocalizableString{
								Ar: fmt.Sprintf("عذراً، انتهت صلاحية قسيمتك للخدمة: %s", voucher.Service.Name.Localize("ar")),
								En: fmt.Sprintf("Sorry, your voucher for service: %s has expired", voucher.Service.Name.Localize("en")),
							},
							TopicType: constants.NotificationTopicTypeUser,
							Topic:     voucher.UserId.Hex(),
							Type:      "fcm",
							Data: util.NotificationPayload{
								Id:    voucher.Id.Hex(),
								Model: "voucher",
							},
						},
					)

					// Mark the voucher as notified
					voucher.ExpiryNotified = true
					voucherService.Update(voucher.Id.Hex(), &voucher.Voucher)

					// add transaction
					voucherService.VoucherExpiredTxn(&voucher.Voucher)

				}
			}
		}, ctx),
	)
}
