package services

import (
	"context"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/database"
	"loyalty/pkg/models"
	"loyalty/pkg/util"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type VisitPackageService interface {
	GetOne(id string) (*models.VisitPackage, error)
	Get(args *util.ListArgs) (*models.VisitPackageWithPagination, error)
	GetAll(query string) ([]*models.VisitPackage, error)
	Add(data *models.VisitPackage) (*models.VisitPackage, error)
	Update(id string, data *models.VisitPackage) (*models.VisitPackage, error)
	Delete(id string) error
	Redeem(id, userId string) (*models.UserWalletResponse, error)
}
type VisitPackageSvc struct {
	MainService[*models.VisitPackage, *models.VisitPackage]
}

func NewVisitPackageSvc(ctx context.Context, db string) VisitPackageService {
	return &VisitPackageSvc{
		MainService: MainService[*models.VisitPackage, *models.VisitPackage]{
			ctx:      ctx,
			db:       db,
			collName: "loyaltyVisitPackages",
		}}
}

func (w *VisitPackageSvc) Get(args *util.ListArgs) (*models.VisitPackageWithPagination, error) {
	coll := database.Mdb.Database(w.db).Collection(w.collName)

	pipeline := PaginationPipe("visit_packages", args)

	cursor, err := coll.Aggregate(w.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var result models.VisitPackageWithPagination
	if cursor.Next(w.ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no data found")
}

func (w *VisitPackageSvc) Redeem(id, userId string) (*models.UserWalletResponse, error) {
	return WithTransaction(w.ctx, func(sessCtx mongo.SessionContext) (*models.UserWalletResponse, error) {
		visitpkgSvc := NewVisitPackageSvc(sessCtx, w.db)
		pkg, err := visitpkgSvc.GetOne(id)
		if err != nil {
			return nil, fmt.Errorf("visit package not found: %w", err)
		}
		// get user wallet
		walletService := NewUserWalletSvcs(sessCtx, w.db)
		wallet, err := walletService.GetWalletByUserId(userId)
		if err != nil {
			return nil, fmt.Errorf("wallet not found: %w", err)
		}
		_, err = walletService.AddPoints(wallet.Id.Hex(), pkg.Points, pkg.Points)
		if err != nil {
			return nil, fmt.Errorf("failed to add points to wallet: %w", err)
		}
		// add transaction
		user := w.ctx.Value(constants.CtxAuthKey).(common.Credentials)
		userId, _ := primitive.ObjectIDFromHex(user.User.Id)
		transactionService := NewTransactionSvcs(sessCtx, w.db)
		newTransaction := &models.Transaction{
			UserId:   wallet.UserId,
			Activity: "Visit Package redeemed",
			Type:     constants.TXN_VISIT_PACKAGE_REDEEMED,
			TypeName: constants.TXN_VISIT_PACKAGE_REDEEMED.String(),
			Amount:   float64(pkg.Points),
			RefId:    &pkg.Id,
		}
		newTransaction.SetCreatedBy(userId)
		_, err = transactionService.Add(newTransaction)
		if err != nil {
			return nil, err
		}
		return wallet, nil
	})
}
