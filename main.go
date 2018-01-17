package main

import (
	"log"
	"time"
	"strconv"
	"os"
)

const iCalDateFormat = "20060102"

var db *pgDb

func main() {
	var dbHost, dbUser, dbPass, dbName, updateInterval string
	var updateIntervalInt int
	dbHost = os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "127.0.0.1"
	}
	dbUser = os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPass = os.Getenv("DB_PASS")
	dbName = os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "airbnb_cal"
	}
	updateInterval = os.Getenv("UPDATE_INTERVAL")
	if updateInterval == "" {
		updateInterval = "600"
	}
	if updateIntervalFloat, err := strconv.ParseFloat(updateInterval, 64); err != nil {
		log.Fatalf("Need an integer update interval. %v is not valid", updateInterval)
	} else {
		updateIntervalInt = int(updateIntervalFloat)
	}

	log.Println("Connecting to database")
	var err error
	db, err = initDb(&dbConfig{
		dbHost: dbHost,
		dbUser: dbUser,
		dbPass: dbPass,
		dbName: dbName,
	})
	if err != nil {
		log.Fatalf("Error initializing database: %v\n", err)
	}

	log.Println("Setting up tables")
	err = db.createTablesIfNotExist()
	if err != nil {
		log.Fatalf("Error creating database tables: %v\n", err)
	}

	log.Println("Starting event update goroutine")
	updateTicker := time.NewTicker(time.Second * time.Duration(updateIntervalInt))
	go func() {
		for range updateTicker.C {
			log.Println("Updating events")
			updateEvents()
		}
	}()

	log.Println("Starting http server")
	serveHttp()
}
