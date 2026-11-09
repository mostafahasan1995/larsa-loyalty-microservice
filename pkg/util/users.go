package util

import (
	"encoding/json"
	"fmt"
	"strings"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
)

type DataWithUserId interface {
	GetUserId() string
	SetUser(user *common.User)
}

type DataWithPartnerId interface {
	GetPartnerId() string
	SetPartner(*common.User)
}

func FetchUsers(userIDs []string, clientId string) ([]*common.User, error) {
	serviceToken, err := common.GetServiceToken("loyalty", &common.HeaderParams{})
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("{\"_id\":{\"$in\":[\"%s\"]}}", strings.Join(userIDs, `","`))

	hp := common.HeaderParams{
		Client:       clientId,
		ServiceToken: serviceToken,
	}
	reqOp := &common.RequestParams{
		Service: "users",
		Path:    fmt.Sprintf("users/all?query=%s", query),
		Method:  "GET",
		Data:    map[string]interface{}{},
		Header:  &hp,
	}

	var result struct {
		Users []*common.User
	}
	res, err := common.CallService(reqOp)
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(res.Body).Decode(&result)

	return result.Users, err
}

func MapUsersToData[T DataWithUserId](users []*common.User, data []T) []T {
	userMap := make(map[string]*common.User)
	for _, user := range users {
		userMap[user.Id] = user
	}

	for i := range data {
		if user, exists := userMap[data[i].GetUserId()]; exists {
			data[i].SetUser(user)
		}
	}
	return data
}

func MapPartnerToData[T DataWithPartnerId](users []*common.User, data []T) []T {
	userMap := make(map[string]*common.User)
	for _, user := range users {
		userMap[user.Id] = user
	}

	for i := range data {
		if user, exists := userMap[data[i].GetPartnerId()]; exists {
			data[i].SetPartner(user)
		}
	}
	return data
}
