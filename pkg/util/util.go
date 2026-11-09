package util

import (
	"fmt"
	"loyalty/pkg/constants"
	"net/http"
	"os"
	"strconv"
	"time"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"github.com/greatcloak/decimal"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

type ListArgs struct {
	PerPage    string
	Page       string
	Search     string
	Query      bson.M
	Sort       bson.M
	Projection bson.M
	GetAll     bool
	GroupBy    string
}

func Paginate(r *http.Request) (*ListArgs, error) {
	var args ListArgs
	sort, sErr := GetBsonFromString(r.URL.Query().Get("sort"))
	if sErr != nil {
		return nil, sErr
	}
	if len(sort) == 0 {
		orderBy, orderByErr := strconv.Atoi(r.URL.Query().Get("orderBy"))
		if orderByErr != nil || (orderBy != -1 && orderBy != 1) {
			orderBy = -1
		}
		sortBy := r.URL.Query().Get("sortBy")
		if sortBy == "" {
			sortBy = "_id"
		}
		sort = bson.M{sortBy: orderBy}
	}
	queryBs, qErr := GetBsonFromString(r.URL.Query().Get("query"))
	if qErr != nil {
		return nil, qErr
	}
	projectionBs, pErr := GetBsonFromString(r.URL.Query().Get("project"))
	if pErr != nil {
		return nil, qErr
	}
	args.Sort = sort
	args.Page, _ = lo.Coalesce(r.URL.Query().Get("page"), "1")
	args.PerPage, _ = lo.Coalesce(r.URL.Query().Get("perPage"), "40")
	args.Search = r.URL.Query().Get("search")
	args.GroupBy = r.URL.Query().Get("groupBy")
	args.Query = queryBs
	args.Projection = projectionBs

	return &args, nil
}

func CacheKeyForMulti(key constants.CacheKey, db, path, lang, rawQuery string) string {
	pattern := "%s:p:c:%s:%s:%s?%s"
	return fmt.Sprintf(pattern, key, db, path, lang, rawQuery)
}

func CacheKeyForOne(key constants.CacheKey, db, id, lang string) string {
	pattern := "%s:g:c:%s:%s:%s"
	return fmt.Sprintf(pattern, key, db, id, lang)
}

func CacheKeyDeleteMul(key constants.CacheKey, db string) string {
	pattern := "%s:p:c:%s:*"
	return fmt.Sprintf(pattern, key, db)
}
func CacheKeyDeleteOne(key constants.CacheKey, db, id string) string {
	pattern := "%s:g:c:%s:%s:*"
	return fmt.Sprintf(pattern, key, db, id)
}

func Percent(val, percent decimal.Decimal) decimal.Decimal {
	return val.Div(decimal.NewFromInt(100)).Mul(percent)
}

func StartAndEndDay(date time.Time) (time.Time, time.Time) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	end := start.Add(24 * time.Hour)
	return start, end
}

func NumStringFromOid(id primitive.ObjectID) int64 {
	res, _ := strconv.ParseInt(fmt.Sprintf("%s%s", id.Hex()[:8], id.Hex()[18:]), 16, 64)
	return res
}

func FromSlicePtr[T any](collection []*T) []T {
	return lo.FilterMap(collection, func(item *T, _ int) (T, bool) {
		if item == nil {
			return lo.Empty[T](), false
		}
		return *item, true
	})
}

func RelevantDbs() []string {
	var dbs []string
	ch := make(chan string)
	go common.FetchSubscriptions(common.ClientSubscriptionInstance, ch)
	<-ch
	clients := lo.Filter(lo.Values(common.ClientSubscriptionInstance.SubscriptionData),
		func(item common.Subscription, i int) bool {
			return lo.Contains(item.Microservices, "loyalty")
		})
	for _, v := range clients {
		dbs = append(dbs, v.Id)
	}
	return dbs
}

func BsonToJSONString(doc bson.M) string {
	jsonBytes, _ := bson.MarshalExtJSON(doc, true, true)

	return string(jsonBytes)
}

func GetBsonFromString(query string) (bson.M, error) {
	var bs bson.M = bson.M{}
	if query != "" {
		err := bson.UnmarshalExtJSON([]byte(query), true, &bs)
		if err != nil {
			return nil, err
		}
	}
	return bs, nil
}
