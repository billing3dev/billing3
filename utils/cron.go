package utils

import (
	"log/slog"

	"github.com/go-co-op/gocron/v2"
)

var scheduler gocron.Scheduler

func InitCronScheduler() {
	s, err := gocron.NewScheduler()
	if err != nil {
		slog.Error("failed to create cron scheduler", "err", err)
		panic(err)
	}
	scheduler = s
}

func StartCronScheduler() {
	scheduler.Start()
}

func NewCronJob(crontab string, fn func() error, name string) {
	slog.Info("new cron job", "name", name)

	_, err := scheduler.NewJob(
		gocron.CronJob(crontab, false),
		gocron.NewTask(func() {
			err := fn()
			if err != nil {
				slog.Error("cron job", "err", err, "name", name)
			}
		}),
		gocron.WithName(name),
		gocron.WithStartAt(gocron.WithStartImmediately()),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		panic(err)
	}
}

func StopCronJobs() {
	slog.Info("stopping cron jobs")
	err := scheduler.Shutdown()
	if err != nil {
		slog.Error("failed to stop cron scheduler", "err", err)
	}
	slog.Info("cron jobs stopped")
}
