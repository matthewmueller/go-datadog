// TODO: somehow verify the logs are in datadog
package datadog_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/apex/log"
	"github.com/matthewmueller/go-datadog"
)

func TestEnv(t *testing.T) {
	key := os.Getenv("DATADOG_API_KEY")
	if key == "" {
		t.Fatal("no DATADOG_API_KEY set")
	}
}

func TestConnect(t *testing.T) {
	key := os.Getenv("DATADOG_API_KEY")
	dd, err := datadog.Dial(&datadog.Config{
		APIKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := dd.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestWrite(t *testing.T) {
	key := os.Getenv("DATADOG_API_KEY")
	dd, err := datadog.Dial(&datadog.Config{
		APIKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer dd.Close()

	msg := `me too`
	n, err := dd.Write([]byte(msg))
	if err != nil {
		t.Fatal(err)
	} else if len(msg) != n {
		t.Fatal("length mismatch")
	}
}

func TestMultilineWrite(t *testing.T) {
	key := os.Getenv("DATADOG_API_KEY")
	dd, err := datadog.Dial(&datadog.Config{
		APIKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer dd.Close()

	msg := `me
	too
	too`
	n, err := dd.Write([]byte(msg))
	if err != nil {
		t.Fatal(err)
	} else if len(msg) != n {
		t.Fatal("length mismatch")
	}
}

func TestApex(t *testing.T) {
	key := os.Getenv("DATADOG_API_KEY")
	dd, err := datadog.Dial(&datadog.Config{
		APIKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer dd.Close()

	log := log.Logger{
		Level:   log.InfoLevel,
		Handler: dd,
	}

	log.WithField("some", "error").Error("error")
	log.WithField("some", "warning").Warn("warning")
}

func TestApexCloseEarly(t *testing.T) {
	var stderr bytes.Buffer

	key := os.Getenv("DATADOG_API_KEY")
	dd, err := datadog.Dial(&datadog.Config{APIKey: key})
	if err != nil {
		t.Fatal(err)
	}

	log := log.Logger{
		Level:   log.InfoLevel,
		Handler: dd,
	}

	log.WithField("some", "error").Error("error")
	log.WithField("some", "warning").Warn("warning")
	dd.Close()
	log.WithField("some", "info").Info("info")

	fmt.Println(stderr.String())
	// if stderr.String() !=
}
