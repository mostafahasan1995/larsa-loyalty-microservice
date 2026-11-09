package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"git.larsa.io/mahdawi/microservices-commons.git/common"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
)

type Resource int

//go:generate go run github.com/dmarkham/enumer -type=Resource,Action
const (
	ChallengeAnswer Resource = iota
	Challenge
	CommunityPost
	DailyTask
	Feedback
	Level
	QuizAnswer
	Quiz
	ServiceGroup
	Service
	Setting
	SurveyAnswer
	Survey
	Transaction
	UserWallet
	VisitPackage
	Voucher
	DbManagement
)

type Action int

const (
	Post Action = iota
	PostOthers
	Put
	PutOthers
	Delete
	DeleteOthers
	Get
	GetOthers
	GetOne
	GetOneOthers
)

var Capabilities = make(map[string]map[string]string)

func initCapabilities() {
	for _, r := range ResourceStrings() {
		Capabilities[r] = make(map[string]string)
		for _, a := range ActionStrings() {
			capability := fmt.Sprintf("loyalty%s%s", a, r)
			Capabilities[r][a] = capability
		}
	}
}

func SetupCapabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	hp := common.ExtractHeaderParams(r)
	res := lop.Map(lo.Values(Capabilities), func(resource map[string]string, _ int) []string {
		return lo.Map(lo.Values(resource), func(capability string, _ int) string {
			data := map[string]interface{}{
				"name":  capability,
				"label": capability,
			}
			reqOp := &common.RequestParams{
				Service: "users",
				Path:    "capabilities",
				Method:  "POST",
				Data:    data,
				Header:  hp,
			}
			res, err := common.CallService(reqOp)
			if err != nil {
				fmt.Println(err)
				return ""
			}
			if lo.Contains([]int{400, 401, 403, 404, 500}, res.StatusCode) {
				return ""
			}
			return capability
		})
	})
	j, _ := json.Marshal(res)
	w.Write(j)
}
