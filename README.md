# Retry package for the DLiveR system

Retry package for our system. Currently only supports HTTP transport layer retries with gentleman.
Currently only exponential backoff is implemented as a retry logic.

## Installation

1. Add dependency to go mod
2. Run go build/run/tidy

```bash
go get -u github.com/proemergotech/retry v1.0.0
```

## Usage

```go
type ExampleHTTPClient struct {
	httpClient *gentleman.Client
}

func NewExampleHTTPClient(ctx context.Context, httpClient *gentleman.Client) *ExampleHTTPClient {
	return &ExampleHTTPClient{
        httpClient: httpClient,
    }
}

exampleHTTPClient := client.NewExampleHTTPClient(
    gentleman.New().BaseURL(
        fmt.Sprintf("%v://%v:%v", cfg.ExampleHTTPClientScheme, cfg.ExampleHTTPClientHost, cfg.ExampleHTTPClientPort),
    ).
        Use(client.Middleware(gentlemanretry.BackoffTimeout(10*time.Second), gentlemanretry.Logger(logger), gentlemanretry.RequestTimeout(2*time.Second))),
)
```

## Documentation

Private repos don't show up on godoc.org so you have to run it locally.

```
godoc -http=":6060"
```

Then open http://localhost:6060/pkg/github.com/proemergotech/retry/

## Development

- install go
- check out project to: $GOPATH/src/github.com/proemergotech/retry
