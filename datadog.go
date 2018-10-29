package datadog

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
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

	d := &Datadog{
		Config: cfg,
		Queue:  queue.New(100, 1),
	}

	// establish the initial connection
	if err := d.dial(); err != nil {
		return nil, err
	}

	return d, nil
}

// Datadog struct
type Datadog struct {
	Config *Config
	Queue  *queue.Queue

	// mu protects the connection
	mu   sync.Mutex
	conn net.Conn
}

var _ io.WriteCloser = (*Datadog)(nil)
var _ log.Handler = (*Datadog)(nil)

// Dial with reconnect
func (d *Datadog) redial() error {
	backo := backoff.NewExponentialBackOff()
retry:
	err := d.dial()
	if err == nil {
		return nil
	}
	sleep := backo.NextBackOff()
	if sleep == backoff.Stop {
		return errors.New("failed to reconnect")
	}
	time.Sleep(sleep)
	goto retry
}

func (d *Datadog) dial() error {
	dialer := &net.Dialer{
		KeepAlive: 5 * time.Minute,
		Timeout:   timeout,
	}
	conn, err := dialer.Dial("tcp", endpoint)
	if err != nil {
		return err
	}
	sslConn := tls.Client(conn, &tls.Config{
		ServerName: "*.logs.datadoghq.com",
	})
	// test the handshake beforehand
	if err := sslConn.Handshake(); err != nil {
		return err
	}

	// update the connection
	d.mu.Lock()
	d.conn = sslConn
	d.mu.Unlock()

	return nil
}

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
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	if _, err := d.Write(buf); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	return nil
}

// Write to the tcp connection
// TODO: retry until we reach n == len(buf)
func (d *Datadog) Write(b []byte) (int, error) {
	var buf bytes.Buffer
	buf.WriteString(d.Config.APIKey)
	buf.WriteString(" ")
	buf.Write(b)
	buf.WriteString("\n")

	d.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	n, err := io.Copy(d.conn, &buf)
	if err != nil {
		if err := d.redial(); err != nil {
			return int(n), err
		}
		return d.Write(b)
	}
	return len(b), nil
}

// Close the datadog connection
func (d *Datadog) Close() error {
	// ignore any new logs that come in
	// process the logs that we have
	d.Queue.Close()

	// close the TCP connection
	return d.conn.Close()
}

// Flush the queue
func (d *Datadog) Flush() {
	d.Queue.Wait()
}
