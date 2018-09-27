package datadog

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"time"

	"github.com/matthewmueller/go-datadog/internal/queue"

	"github.com/apex/log"
)

var endpoint = "intake.logs.datadoghq.com:10516"
var timeout = 20 * time.Second
var errNoKey = errors.New("No API key provided. Generate one here: https://app.datadoghq.com/account/settings#api")

// Config struct
type Config struct {
	// API Key (required)
	APIKey string

	// Optional
	// See: https://docs.datadoghq.com/logs/#reserved-attributes
	Host    string
	Service string
	Source  string
}

// Dial datadog
func Dial(cfg *Config) (*Datadog, error) {
	if cfg.APIKey == "" {
		return nil, errNoKey
	}

	conn, err := net.DialTimeout("tcp", endpoint, timeout)
	if err != nil {
		return nil, err
	}
	sslConn := tls.Client(conn, &tls.Config{
		ServerName: "*.logs.datadoghq.com",
	})

	d := &Datadog{
		Config: cfg,
		Conn:   sslConn,
		Queue:  queue.New(100, 1),
	}

	return d, nil
}

// Datadog struct
type Datadog struct {
	Config *Config

	Conn  net.Conn
	Queue *queue.Queue
}

var _ io.WriteCloser = (*Datadog)(nil)
var _ log.Handler = (*Datadog)(nil)

// HandleLog implements log.Handler
//
// Doesn't start blocking until the channel is full
// If the queue is closed, this function returns
// and error immediately
func (d *Datadog) HandleLog(l *log.Entry) error {
	return d.Queue.Push(func() { d.send(l) })
}

// Send the entry to datadog
// TODO: any better way to handle errors here?
func (d *Datadog) send(e *log.Entry) error {
	entry := map[string]interface{}{}

	entry["host"] = d.Config.Host
	entry["service"] = d.Config.Service
	entry["source"] = d.Config.Source

	for k, v := range e.Fields {
		entry[k] = v
	}

	entry["level"] = e.Level
	entry["message"] = e.Message
	entry["timestamp"] = e.Timestamp.Format(time.RFC3339)

	buf, err := json.Marshal(entry)
	if err != nil {
		os.Stderr.Write([]byte(err.Error()))
		return err
	}

	if _, err := d.Write(buf); err != nil {
		os.Stderr.Write([]byte(err.Error()))
		return err
	}

	return nil
}

// Write to the tcp connection
// TODO: retry until we reach n == len(buf)
func (d *Datadog) Write(b []byte) (int, error) {
	buf := d.Config.APIKey + " " + string(b) + "\n"
	if n, err := d.Conn.Write([]byte(buf)); err != nil {
		return n, err
	}
	return len(b), nil
}

// Close the datadog connection
func (d *Datadog) Close() error {
	// ignore any new logs that come in
	// process the logs that we have
	d.Queue.Close()

	// close the TCP connection
	return d.Conn.Close()
}
