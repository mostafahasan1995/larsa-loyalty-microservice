package models

import (
	"time"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Activity struct {
	Headers     *common.HeaderParams `json:"headers"`
	ServiceName string               `json:"serviceName"`
	ActionName  string               `json:"actionName"`
	ActionData  map[string]any       `json:"actionData"`
	CreatedAt   time.Time            `json:"createdAt"`
	CreatedBy   primitive.ObjectID   `json:"createdBy"`
}
