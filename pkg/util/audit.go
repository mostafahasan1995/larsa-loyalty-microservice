package util

import (
	"context"
	"errors"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/models"
	"time"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"git.larsa.io/mahdawi/microservices-commons.git/nats"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GenerateHumanReadableDescription creates a user-friendly description of the activity
func GenerateHumanReadableDescription(actionName string, actionData map[string]any, user common.Credentials) string {
	userName := "Unknown User"
	if user.User.FirstName != "" {
		userName = user.User.FirstName
	} else if user.User.Email != "" {
		userName = user.User.Email
	}

	// Simply prepend the user's first name to the action name from constants
	// The action names in constants are already human-readable
	return fmt.Sprintf("%s %s", userName, actionName)
}

func ActivityStream(ctx context.Context, act models.Activity) error {
	act.ServiceName = "loyalty"
	act.CreatedAt = time.Now()
	user, ok := ctx.Value(constants.CtxAuthKey).(common.Credentials)
	if !ok {
		return errors.New("error get user form ctx")
	}
	_userId, _ := primitive.ObjectIDFromHex(user.User.Id)
	act.CreatedBy = _userId

	// Generate human-readable description
	humanDescription := GenerateHumanReadableDescription(act.ActionName, act.ActionData, user)

	// Add the human-readable description to action data
	if act.ActionData == nil {
		act.ActionData = make(map[string]any)
	}
	act.ActionData["description"] = humanDescription
	act.ActionData["user_name"] = user.User.FirstName
	act.ActionData["user_email"] = user.User.Email

	nats.Publish("audit:createActivityAudit", act)
	return nil
}
