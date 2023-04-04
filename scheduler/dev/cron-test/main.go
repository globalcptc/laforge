package main

import (
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/sirupsen/logrus"
)

func main() {
	end_time := time.Now().Add(10 * time.Hour)
	expr, err := cronexpr.Parse("0 */1 * * *")
	if err != nil {
		logrus.Fatalf("failed to parse cron expression: %v", err)
		return
	}
	run_times := make([]time.Time, 0)
	nextTime := time.Now()
	for {
		nextTime = expr.Next(nextTime)
		if nextTime.After(end_time) {
			break
		}
		run_times = append(run_times, nextTime)
		logrus.Printf("next time: %s", nextTime)
	}
	logrus.Infof("%v", run_times)
}
