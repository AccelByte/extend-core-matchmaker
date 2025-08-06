// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package common

import (
	"encoding/json"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func GetEnvInt(key string, fallback int) int {
	str := GetEnv(key, strconv.Itoa(fallback))
	val, err := strconv.Atoi(str)
	if err != nil {
		return fallback
	}

	return val
}

// GenerateRandomInt generate a random int that is not determined
func GenerateRandomInt() int {
	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)

	return random.Intn(10000)
}

// LogJSONFormatter is printing the data in log
func LogJSONFormatter(data interface{}) string {
	response, err := json.Marshal(data)
	if err != nil {
		logrus.Errorf("failed to marshal json.")

		return ""
	} else {
		logrus.SetFormatter(&logrus.JSONFormatter{PrettyPrint: false})

		return string(response)
	}
}
