package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

type emailTaskArgs struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

var username, password, endpoint, port, tlsType, from string

func sendmail(to, subject, body string) error {
	slog.Info("sendmail", "to", to, "from", from, "subject", subject)

	var c *smtp.Client
	var err error
	switch tlsType {
	case "TLS":
		c, err = smtp.DialTLS(net.JoinHostPort(endpoint, port), &tls.Config{})
	case "STARTTLS":
		c, err = smtp.DialStartTLS(net.JoinHostPort(endpoint, port), &tls.Config{})
	case "NONE":
		c, err = smtp.Dial(net.JoinHostPort(endpoint, port))
	default:
		panic("unsupported tls type: " + tlsType)
	}
	if err != nil {
		return fmt.Errorf("send mail: dial: %w", err)
	}

	defer c.Close()

	saslClient := sasl.NewLoginClient(username, password)
	err = c.Auth(saslClient)
	if err != nil {
		return fmt.Errorf("send mail: auth: %w", err)
	}

	err = c.Mail(from, nil)
	if err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	err = c.Rcpt(to, nil)
	if err != nil {
		return fmt.Errorf("send mail: %w", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("send mail: %w", err)
	}

	_, err = fmt.Fprintf(w, "Subject: %s\r\nTo: %s\r\nContent-Type: text/html\r\n\r\n%s", subject, to, body)
	if err != nil {
		return fmt.Errorf("send mail: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("send mail: %w", err)
	}

	err = c.Quit()
	if err != nil {
		return fmt.Errorf("send mail: %w", err)
	}

	return nil
}

func SendMailAsync(ctx context.Context, to, subject, body string) error {
	go func() {
		err := sendmail(to, subject, body)
		if err != nil {
			slog.Error("send mail async", "error", err)
		}
	}()
	return nil
}
