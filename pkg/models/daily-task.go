package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type DailyTask struct {
	Id   primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name LocalizableString  `bson:"name" json:"name"`
	Icon []File             `bson:"icon" json:"icon"`

	BaseDbModel `bson:",inline" json:",inline"`
}

type DailyTaskResponse struct {
	DailyTask   `bson:",inline" json:",inline"`
	IsCompleted *bool `bson:"is_completed,omitempty" json:"is_completed,omitempty"`
}

type DailyTasksWithDate struct {
	Date  string              `json:"date"`
	Tasks []DailyTaskResponse `json:"tasks"`
}

type DailyTaskWithPagination struct {
	DailyTasks []DailyTaskResponse `bson:"daily_tasks" json:"daily_tasks"`
	Pagination Pagination          `bson:"pagination" json:"pagination"`
}
