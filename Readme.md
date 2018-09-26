Easily send structured logs to [Datadog](https://www.datadoghq.com/) over TCP.

[![GoDoc](https://godoc.org/github.com/matthewmueller/go-datadog?status.svg)](https://godoc.org/github.com/matthewmueller/go-datadog)

## Features

- Implements `io.Writer`

```go
key := os.Getenv("DATADOG_API_KEY")
dd, err := datadog.Dial(&datadog.Config{APIKey: key})
defer dd.Close()
client.Write([]byte("some log"))
```

- Implements `github.com/apex/log.Handler`

```go
key := os.Getenv("DATADOG_API_KEY")
dd, err := datadog.Dial(&datadog.Config{APIKey: key})
defer dd.Close()
log := log.Logger{
  Level:   log.InfoLevel,
  Handler: dd,
}
log.Info("some log")
```

## License

MIT