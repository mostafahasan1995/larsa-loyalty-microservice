package constants

import (
	"fmt"
)

type NotificationTopicType string

const (
	NotificationTopicTypeGeneral       NotificationTopicType = "general"
	NotificationTopicTypeLevel         NotificationTopicType = "level"
	NotificationTopicTypeUser          NotificationTopicType = "user"
	NotificationTopicTypeDailyTasks    NotificationTopicType = "daily_tasks"
	NotificationTopicTypeQuiz          NotificationTopicType = "quiz"
	NotificationTopicTypeSurvey        NotificationTopicType = "survey"
	NotificationTopicTypeChallenge     NotificationTopicType = "challenge"
	NotificationTopicTypeAdmin         NotificationTopicType = "admin"
	NotificationTopicTypeReceptionist  NotificationTopicType = "receptionist"
	NotificationTopicTypePendingAction NotificationTopicType = "pending_action"
	NotificationTopicTypeDashboard     NotificationTopicType = "dashboard"
)

func (n NotificationTopicType) Format(lang Lang, param string) string {
	if param == "" {
		return fmt.Sprintf("%s-%s", lang, n)
	}
	return fmt.Sprintf("%s-%s-%s", lang, n, param)
}
