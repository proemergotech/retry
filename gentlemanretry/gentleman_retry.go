package gentlemanretry

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"time"

	"gitlab.com/proemergotech/errors"
	"gitlab.com/proemergotech/retry"
	"gitlab.com/proemergotech/retry/backoff"
	gcontext "gopkg.in/h2non/gentleman.v2/context"
	"gopkg.in/h2non/gentleman.v2/plugin"
)

var noopCancel = func() {}

type Option func(*transport)

type evalFunc func(error, *http.Request, *http.Response) (retry bool, err error)

type transport struct {
	evaluator      evalFunc
	transport      http.RoundTripper
	gctx           *gcontext.Context
	backoffTimeout time.Duration
	requestTimeout time.Duration
	logger         retry.Logger
	loggingEnabled bool
	logRequest     bool
	logResponse    bool
}

// Middleware returns a gentleman plugin implementing retry logic for http requests implementing http round tripper interface.
func Middleware(options ...Option) plugin.Plugin {
	return plugin.NewPhasePlugin("before dial", func(gctx *gcontext.Context, handler gcontext.Handler) {
		t := &transport{
			evaluator:      defaultEvaluator,
			transport:      gctx.Client.Transport,
			gctx:           gctx,
			backoffTimeout: backoff.DefaultMaxElapsedTime,
		}

		for _, opt := range options {
			opt(t)
		}

		gctx.Client.Transport = t

		handler.Next(gctx)
	})
}

// RoundTrip is the implementation of the standard http round tripper interface.
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte

	if req.Body != nil {
		buf, err := ioutil.ReadAll(req.Body)
		if err != nil {
			_ = req.Body.Close()
			return t.gctx.Response, err
		}
		_ = req.Body.Close()

		body = buf
	}

	reqCopy := req
	resetTimeout := false

	if t.requestTimeout > 0 {
		if _, ok := req.Context().Deadline(); !ok {
			reqCopy = req.Clone(req.Context())
			resetTimeout = true
		}
	}

	return t.retry(reqCopy, body, reqCopy.Context().Done(), resetTimeout)
}

// BackoffTimeout returns an option setting the transports backoff timeout between retries to the given duration.
func BackoffTimeout(timeout time.Duration) Option {
	return func(t *transport) {
		t.backoffTimeout = timeout
	}
}

// RequestTimeout returns an option setting the transport request timeout to the given duration.
//
// If the original request had a deadline, than it will be used mainly.
// Without a deadline on the original request the given timeout is set per retry for the request.
func RequestTimeout(timeout time.Duration) Option {
	return func(t *transport) {
		t.requestTimeout = timeout
	}
}

// Evaluator returns an option containing an evaluator function capable of deciding whether a new retry is needed.
//
// If not set the defaultEvaluator will be used.
// If the request has resulted in an error, or the request status code indicates a server problem,
// or the request timed out, the default evaluator returns that a retry is needed.
func Evaluator(evalFn evalFunc) Option {
	return func(t *transport) {
		t.evaluator = evalFn
	}
}

// Logger enables logging and adds a logger interface to the transport struct.
func Logger(logger retry.Logger) Option {
	return func(t *transport) {
		t.loggingEnabled = true
		t.logger = logger
	}
}

// LogRequest returns an option which will enable the logging of requests using httputil.DumpResponse.
func LogRequest() Option {
	return func(t *transport) {
		t.logRequest = true
	}
}

// LogResponse returns an option which will enable the logging of responses using httputil.DumpResponse.
func LogResponse() Option {
	return func(t *transport) {
		t.logResponse = true
	}
}

func (t *transport) retry(req *http.Request, body []byte, done <-chan struct{}, resetTimeout bool) (*http.Response, error) {
	cancel := noopCancel
	retryCount := 0
	bOff := backoff.NewExponentialBackOff(t.backoffTimeout, backoff.DefaultMaxInterval, backoff.DefaultRandomizationFactor)
	origCtx := req.Context()

	for {
		if resetTimeout {
			ctx, c := context.WithTimeout(origCtx, t.requestTimeout)
			req = req.WithContext(ctx)
			cancel = c
		}

		if body != nil {
			req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			req.GetBody = func() (io.ReadCloser, error) {
				if t.loggingEnabled {
					t.logger.Debug(req.Context(), "GetBody called on request")
				}
				return ioutil.NopCloser(bytes.NewBuffer(body)), nil
			}
		}

		res, err := t.transport.RoundTrip(req)
		r, err := t.evaluator(err, req, res)
		if !r {
			return res, err
		}

		cancel()

		if res != nil {
			if !t.logRequest && !t.logResponse {
				_, _ = io.Copy(ioutil.Discard, res.Body)
			} else {
				fields := make([]interface{}, 0, 4)

				if t.logRequest {
					b, _ := httputil.DumpRequest(req, true)
					fields = append(fields, "request", string(b))
				}

				if t.logResponse {
					b, _ := httputil.DumpResponse(res, true)
					fields = append(fields, "response", string(b))
				}

				err = errors.WithFields(err, fields...)
			}

			_ = res.Body.Close()
		}

		hasNext, duration := bOff.NextBackOff()
		if !hasNext {
			return nil, err
		}

		select {
		case <-time.After(duration):
			retryCount++
			if t.loggingEnabled {
				t.logger.Warn(req.Context(), fmt.Sprintf("error during request, retry # %d", retryCount), "error", err)
			}
		case <-done:
			return nil, err
		}
	}
}

func defaultEvaluator(err error, req *http.Request, res *http.Response) (bool, error) {
	if err != nil {
		return true, err
	}

	if res.StatusCode >= 500 || res.StatusCode == http.StatusRequestTimeout {
		return true, errors.New("server response error")
	}

	return false, nil
}
