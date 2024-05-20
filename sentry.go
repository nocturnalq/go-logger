package gologger

import (
	"fmt"
	"time"

	"github.com/certifi/gocertifi"
	"github.com/getsentry/sentry-go"
)

func InitSentry(dsn string, debugMode bool) error {
	sentryClientOptions := sentry.ClientOptions{
		Dsn:   dsn,
		Debug: debugMode,
	}
	rootCAs, err := gocertifi.CACerts()
	if err != nil {
		return fmt.Errorf("could not load CA Certificates: %v", err)
	}
	sentryClientOptions.CaCerts = rootCAs

	if err := sentry.Init(sentryClientOptions); err != nil {
		return fmt.Errorf("InitSentry error; %w", err)
	}
	defer sentry.Flush(2 * time.Second)

	return nil
}

// CaptureErrorWithSentry sent error to sentry as event and return event id
func CaptureErrorWithSentry(err error) *sentry.EventID {
	return sentry.CaptureException(err)
}
