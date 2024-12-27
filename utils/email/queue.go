package email

import "log/slog"

type emailTask struct {
	to      string
	subject string
	body    string
}

var emailQueue = make(chan *emailTask, 20)

func startWorker() {
	go func() {
		for {
			task := <-emailQueue

			slog.Debug("send mail task", "to", task.to, "subject", task.subject)

			err := SendMail(task.to, task.subject, task.body)
			if err != nil {
				slog.Error("send mail", "err", err)
			}
		}
	}()
}

func SendMailAsync(to, subject, body string) {
	emailQueue <- &emailTask{
		to:      to,
		subject: subject,
		body:    body,
	}
}
