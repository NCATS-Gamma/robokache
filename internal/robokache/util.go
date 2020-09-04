package robokache

import (
	"os"
)

func getenv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return value
}

var (
	dataDir = getenv("ROBOKACHE_DATA_DIR", "./data")
	dbFile  = dataDir + "/q&a.db"
)
