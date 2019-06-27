package redisengine

import (
	"testing"
	"time"

	"github.com/morfien101/go-metrics-reciever/config"
)

func TestValidation(t *testing.T) {
	config := config.RedisConfig{
		RedisHost: "172.17.0.1",
		RedisPort: "6379",
	}

	re := RedisEngine{
		config: config,
	}

	if err := re.Start(); err != nil {
		t.Logf("Can't run test with out redis")
		t.FailNow()
	}
	// Write a test string in
	username := "test_username"
	password := "AmazinglyStrongPassword!!"
	ok, err := re.client.SetNX(username, password, 10*time.Second).Result()
	if err != nil {
		t.Logf("Failed to write in test creds")
		t.Fail()
	}
	ok, err = re.ValidateAuth(username, password)
	if err != nil {
		t.Logf("Got an error checking auth. Error: %s\n", err)
		t.Fail()
	}

	if !ok {
		t.Logf("Check failed to validate username.")
		t.Fail()
	}

}
