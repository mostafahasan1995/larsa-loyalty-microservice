package main

import (
	"fmt"
	"log"
	"loyalty/pkg/api"
	"loyalty/pkg/caching"
	"loyalty/pkg/database"
	"loyalty/pkg/jobs"
	"loyalty/pkg/nats"
	"loyalty/pkg/util"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	decimal.MarshalJSONWithoutQuotes = true

	database.InitDB()
	caching.InitRedis()
	nats.InitNats()
	jobs.InitScheduler()

	database.SeedConfig()

	r := api.InitRouter()

	port := util.GetEnv("port", "3232")
	fmt.Println("Server started on port " + port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
