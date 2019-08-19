// +build integration

package factory

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)

	options := &DatabaseConnectionOptions{
		DSN:    "factory:factory@tcp(factory-db:3306)/factory?collation=utf8mb4_general_ci",
		Logger: "stdout",

		Retries:        100,
		RetryTimeout:   time.Second,
		ConnectTimeout: 2 * time.Minute,
	}

	if _, err := Database.TryToConnect(context.Background(), "default", options); err != nil {
		log.Fatalf("Error when trying to connect: %+v", err)
		return
	}

	os.Exit(m.Run())
}
