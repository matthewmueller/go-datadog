# Datadog Logger

Easily send structured logs to Datadog over TCP.

## Features

- Implements `io.Writer`

```go
key := os.Getenv("DATADOG_API_KEY")
dd, err := datadog.Dial(&datadog.Config{APIKey: key})
defer dd.Close()
client.Write([]byte("some log"))
```

- Implements `apex/log.Handler`

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