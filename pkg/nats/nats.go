package nats

import (
	"errors"
	"fmt"
	"loyalty/pkg/util"
	"log"
	"time"

	"git.larsa.io/mahdawi/microservices-commons.git/interval"
	"git.larsa.io/mahdawi/microservices-commons.git/nats"
	"github.com/google/uuid"
)

func InitNats() {
	url := fmt.Sprintf(`http://%v:%v`, util.GetEnv("NATS_HOST", "localhost"), util.GetEnv("NATS_PORT", "4223"))
	client := nats.Connect("microservices", uuid.NewString(), url)
	if client == nil {
		err := interval.Do(2, func(attempt int) (bool, error) {
			client = nats.Connect("microservices", uuid.NewString(), url)
			fmt.Println("trying to connect nats server", time.Now())
			if client == nil {
				return true, errors.New("could not connect to nats server")
			}
			return true, nil
		})
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println("connected to nats server")
		}
	} else {
		fmt.Println("connected to nats server")

	}
}
