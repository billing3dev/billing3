package utils

import (
	"log/slog"
	"sync"
	"time"
)

var cronJobs []*CronJob
var initCronJobOnce sync.Once
var cronWg sync.WaitGroup
var cronStop = make(chan int)

type CronJob struct {
	lastRun  time.Time
	duration time.Duration
	name     string
	fn       func() error
}

func NewCronJob(d time.Duration, fn func() error, name string) *CronJob {
	slog.Info("new cron job", "name", name)

	initCronJobOnce.Do(initCronJob)

	j := &CronJob{
		lastRun:  time.Unix(0, 0),
		name:     name,
		fn:       fn,
		duration: d,
	}
	cronJobs = append(cronJobs, j)
	return j
}

func initCronJob() {
	go func() {

		for {
			select {
			case <-cronStop:
				slog.Info("cron jobs stopped")
				return
			case <-time.After(time.Second * 5):
			}

			for _, j := range cronJobs {
				if j.lastRun.Add(j.duration).Before(time.Now()) {
					slog.Info("cron job", "name", j.name)

					j.lastRun = time.Now()
					cronWg.Add(1)
					go func() {
						defer cronWg.Done()
						err := j.fn()
						if err != nil {
							slog.Error("cron job", "err", err, "name", j.name)
						}
					}()

				}
			}
		}

	}()
}

func StopCronJobs() {
	cronStop <- 1

	slog.Info("waiting for cron jobs to finish")
	c := make(chan int)
	go func() {
		cronWg.Wait()
		c <- 1
	}()
	select {
	case <-c:
	case <-time.After(time.Minute * 3):
		slog.Warn("timeout waiting for cron jobs to finish")
	}
}
