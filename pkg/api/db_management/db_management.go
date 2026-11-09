package dbManagement

import (
	"loyalty/pkg/models"
	"loyalty/pkg/services"
	"net/http"

	"github.com/goccy/go-json"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
)

func ApplyOperator(w http.ResponseWriter, r *http.Request) {
	headerParams := common.ExtractHeaderParams(r)
	clientId := headerParams.Client
	db := headerParams.Client

	var data *models.DbOperator
	errDec := json.NewDecoder(r.Body).Decode(&data)
	if errDec != nil {
		http.Error(w, errDec.Error(), http.StatusInternalServerError)
		return
	}
	service := services.NewDbManagementSvc(db)
	var (
		result interface{}
		err    error
	)
	switch data.Operator {
	case "updateMany":
		result, err = service.UpdateMany(clientId, *data)
	case "deleteMany":
		result, err = service.DeleteMany(clientId, *data)
	case "getInfo":
		result, err = service.GetInfo(clientId)
	default:
		http.Error(w, "invalid operator", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	j, _ := json.MarshalContext(r.Context(), result)
	w.Write(j)
}

// func (d *dbManagementHan) TempUpdate(w http.ResponseWriter, r *http.Request) {
// 	listArgs := types.ListArgs{
// 		GetAll:  true,
// 		SortBy:  "_id",
// 		OrderBy: -1,
// 		Query:   bson.M{"packaging.packSize": bson.M{"$ne": nil, "$type": bson.A{"double", "int", "long", "decimal"}}},
// 		Lang:    "en",
// 	}

// 	lookUpPipe := mongo.Pipeline{{
// 		{Key: "$lookup", Value: bson.M{
// 			"from":         "ecommerceUnits",
// 			"localField":   "packaging.unit",
// 			"foreignField": "_id",
// 			"as":           "packaging.unitDetails",
// 		}}},
// 		{{Key: "$unwind", Value: bson.M{
// 			"path": "$packaging.unitDetails",
// 		}}},
// 	}
// 	var products []models.Product
// 	err := d.ProductRepo.List(r.Context(), &listArgs, &products, lookUpPipe)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	var result []models.Product
// 	for _, p := range products {
// 		var u models.Product
// 		d.ProductRepo.Update(r.Context(), bson.M{"_id": p.Id}, nil, &u, bson.M{
// 			"$set": bson.M{
// 				"packaging.packSize": fmt.Sprintf("%v%s", *p.Packaging.PackSize, *p.Packaging.UnitDetails.Symbol),
// 			},
// 		})
// 		result = append(result, u)
// 	}
// 	j, _ := json.Marshal(&result)
// 	w.Write(j)
// }
