package main

// Broker is the single point of entry

import (
	"broker/internal/config"
	"broker/internal/handlers"
	"broker/internal/helpers"

	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// if compose without webserver
// const webPort = "80"
// with caddy
const webPort = "8080"

var app config.AppConfig
var version string
var buildstamp string
var githash string

// type Config struct {
// 	Rabbit *amqp.Connection
// }

func main() {

	// configure logger
	log, _ := zap.NewProduction(zap.WithCaller(false))
	defer func() {
		_ = log.Sync()
	}()

	// print current version
	log.Info("Broker", zap.String("build:", version), zap.String("-", githash), zap.String("Timestamp:",buildstamp))

	// try to connect to rabbitmq
	rabbitConn, err := connect()
	if err != nil {
		log.Error("Error connection to RabbitMQ", zap.Error(err))
		os.Exit(1)
	}
	defer rabbitConn.Close()

	repo := handlers.NewRepo(&app)
	handlers.NewHandlers(repo)
	helpers.NewHelpers(&app)

	// passing connection to the Config struct
	app.Rabbit = rabbitConn

	log.Info("Starting broker service on ", zap.String("port:", webPort))

	// define http server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: routes(log, &app),
	}

	// start the server
	err = srv.ListenAndServe()
	if err != nil {
		log.Panic("Broker panic", zap.Error(err))
	}
}

// Connect connects to the RabbitMQ server
// Returns connection if succeed otherwise err
func connect() (*amqp.Connection, error) {

	var counts int64
	var backOff = 1 * time.Second
	var connection *amqp.Connection

	// don't continue until rabbit is ready
	for {
		c, err := amqp.Dial("amqp://guest:guest@rabbitmq")
		if err != nil {
			log.Println("RabbitMQ not yet ready...")
			counts++
		} else {
			log.Println("Connected to RabbitMQ!")
			connection = c
			break
		}

		if counts > 5 {
			fmt.Println(err)
			return nil, err
		}
		// each delay will be longer during try
		backOff = time.Duration(math.Pow(float64(counts), 2)) * time.Second
		log.Println("backing off...")
		time.Sleep(backOff)
		continue
	}

	return connection, nil
}
