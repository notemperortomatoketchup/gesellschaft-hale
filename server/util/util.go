package util

import "time"

func GetCurrentTime() string {
	now := time.Now()
	postgresTimestamp := now.Format("2006-01-02 15:04:05-07")
	return postgresTimestamp
}
