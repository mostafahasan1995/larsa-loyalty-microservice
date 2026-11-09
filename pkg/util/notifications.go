package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"loyalty/pkg/constants"
	"loyalty/pkg/models"
	"net/http"
	"os"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"git.larsa.io/mahdawi/microservices-commons.git/nats"
)

type LocalizeNotificationMsg struct {
	XClient   string
	XService  string
	Header    models.LocalizableString
	Content   models.LocalizableString
	Topic     string
	TopicType constants.NotificationTopicType
	Type      string
	Data      NotificationPayload
}

type NotificationMsg struct {
	XClient  string
	XService string
	Header   string
	Content  string
	Topic    string
	Type     string
	Data     NotificationPayload
}
type NotificationPayload struct {
	Id    string `json:"id,omitempty"`
	Model string `json:"model,omitempty"`
}

func Notify(msg NotificationMsg) {
	msg.XService = "loyalty"
	nats.Publish("go-notification:create", msg)
}

func LocalizeNotify(msg LocalizeNotificationMsg) {

	msg.XService = "loyalty"

	enMsg := NotificationMsg{
		XClient:  msg.XClient,
		XService: msg.XService,
		Header:   msg.Header.En,
		Content:  msg.Content.En,
		Topic:    msg.TopicType.Format(constants.EN, msg.Topic),
		Type:     msg.Type,
		Data:     msg.Data,
	}

	arMsg := NotificationMsg{
		XClient:  msg.XClient,
		XService: msg.XService,
		Header:   msg.Header.Ar,
		Content:  msg.Content.Ar,
		Topic:    msg.TopicType.Format(constants.AR, msg.Topic),
		Type:     msg.Type,
		Data:     msg.Data,
	}

	// nats.Publish("go-notification:create", enMsg)
	// nats.Publish("go-notification:create", arMsg)
	SendNotificationViaHTTP(msg.XClient, enMsg)
	SendNotificationViaHTTP(msg.XClient, arMsg)

}

// TODO GONOTIFICATIONS_URL in common
func getURL(ename string, port string) string {
	if url := os.Getenv(ename); url != "" {
		return url
	} else {
		return "http://localhost:" + port
	}
}

func CallService(reqOpt *common.RequestParams) (*http.Response, error) {
	serviceUrl := getURL("GONOTIFICATIONS_URL", "3228")
	path := reqOpt.Path
	url := fmt.Sprintf("%s/%s", serviceUrl, path)
	// fmt.Println("CallService:", url, ", Method:", reqOpt.Method)
	j, _ := json.Marshal(reqOpt.Data)
	req, err := http.NewRequest(reqOpt.Method, url, bytes.NewBuffer(j))
	if err != nil {
		return nil, err
	}
	req.Header.Add("x-client", reqOpt.Header.Client)
	req.Header.Add("x-service", reqOpt.Header.Service)
	req.Header.Add("x-service-token", reqOpt.Header.ServiceToken)
	req.Header.Add("x-user-id", reqOpt.Header.UserId)
	req.Header.Add("x-access-token", reqOpt.Header.AccessToken)
	req.Header.Add("accept-language", reqOpt.Header.AcceptLanguage)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, errResp
	}

	// fmt.Println("status", resp.StatusCode)

	return resp, nil

}

func SendNotificationViaHTTP(clientId string, notificationMessage NotificationMsg) error {
	serviceToken, err := common.GetServiceToken("loyalty", &common.HeaderParams{})
	if err != nil {
		return err
	}

	hp := common.HeaderParams{
		Client:       clientId,
		ServiceToken: serviceToken,
	}

	reqOp := &common.RequestParams{
		Service: "notifications",
		Path:    "notification/topic",
		Method:  "POST",
		Data: map[string]interface{}{
			"content": notificationMessage.Content,
			"topic":   notificationMessage.Topic,
			"type":    notificationMessage.Type,
			"data":    notificationMessage.Data,
		},
		Header: &hp,
	}

	_, err = CallService(reqOp)
	if err != nil {
		fmt.Println("🚀 ~ funcSendNotificationViaHTTP ~ err:", err)
		return err
	}

	return nil
}
