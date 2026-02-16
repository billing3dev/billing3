package service

import (
	"billing3/database"
	"billing3/utils"
	"context"
)

func InitCron() {
	utils.InitCronScheduler()

	utils.NewCronJob("0 * * * *", func() error {
		return CloseOverdueInvoices()
	}, "close overdue invoices")

	utils.NewCronJob("0 * * * *", func() error {
		return CancelOverdueServices()
	}, "cancel overdue services")

	utils.NewCronJob("0 * * * *", func() error {
		return database.Q.DeleteExpiredSessions(context.Background())
	}, "delete expired sessions")

	utils.NewCronJob("0 0 * * *", func() error {
		return GenerateRenewalInvoices()
	}, "generate renewal invoices")

	utils.StartCronScheduler()
}
