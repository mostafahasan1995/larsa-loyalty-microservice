package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type DailyTaskInteraction struct {
	Id          primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	UserId      primitive.ObjectID `bson:"user_id" json:"user_id"`
	TaskId      primitive.ObjectID `bson:"task_id" json:"task_id"`
	IsCompleted bool               `bson:"is_completed" json:"is_completed"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type DailyTaskInteractionWithPagination struct {
	DailyTasks []DailyTaskInteraction `bson:"daily_tasks_interactions" json:"daily_tasks_interactions"`
	Pagination Pagination             `bson:"pagination" json:"pagination"`
}
