package services

import (
	"context"
	"loyalty/pkg/database"
	"loyalty/pkg/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CounterService interface {
	GetAndUpdateSourceSeq(name string) (int64, error)
}

type countersvcs struct {
	ctx      context.Context
	db       string
	collName string
}

func NewCounterSvcs(ctx context.Context, db string) CounterService {
	return &countersvcs{
		ctx:      ctx,
		db:       db,
		collName: "loyaltyCounters",
	}
}

func (c *countersvcs) GetAndUpdateSourceSeq(name string) (int64, error) {

	coll := database.Mdb.Database(c.db).Collection(c.collName)

	filter := bson.M{"name": name}
	update := bson.M{"$inc": bson.M{"seq": 1}}

	upsert := true
	after := options.After
	opts := &options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
		Upsert:         &upsert,
	}

	result := coll.FindOneAndUpdate(
		c.ctx,
		filter,
		update,
		opts,
	)

	var data models.Counter
	if err := result.Decode(&data); err != nil {
		return 0, err
	}

	return data.Seq, nil
}
