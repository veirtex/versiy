package env

import (
	"os"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
}

func GetString(key, defult string) string {
	val := os.Getenv(key)
	if val == "" {
		return defult
	}
	return val
}
