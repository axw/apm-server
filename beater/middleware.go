package beater

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

const (
	rateLimitCacheSize       = 1000
	rateLimitBurstMultiplier = 2
)

var (
	serverMetrics = monitoring.Default.NewRegistry("apm-server.server", monitoring.PublishExpvar)
	counter       = func(s string) *monitoring.Int {
		return monitoring.NewInt(serverMetrics, s)
	}
	requestCounter    = counter("request.count")
	responseCounter   = counter("response.count")
	responseErrors    = counter("response.errors.count")
	responseSuccesses = counter("response.valid.count")

	okResponse = serverResponse{
		nil, http.StatusOK, counter("response.valid.ok"),
	}
	acceptedResponse = serverResponse{
		nil, http.StatusAccepted, counter("response.valid.accepted"),
	}
	forbiddenResponse = serverResponse{
		errors.New("forbidden request"), http.StatusForbidden, counter("response.errors.forbidden"),
	}
	unauthorizedResponse = serverResponse{
		errors.New("invalid token"), http.StatusUnauthorized, counter("response.errors.unauthorized"),
	}
	requestTooLargeResponse = serverResponse{
		errors.New("request body too large"), http.StatusRequestEntityTooLarge, counter("response.errors.toolarge"),
	}
	decodeCounter        = counter("response.errors.decode")
	cannotDecodeResponse = func(err error) serverResponse {
		return serverResponse{
			errors.Wrap(err, "data decoding error"), http.StatusBadRequest, decodeCounter,
		}
	}
	validateCounter        = counter("response.errors.validate")
	cannotValidateResponse = func(err error) serverResponse {
		return serverResponse{
			errors.Wrap(err, "data validation error"), http.StatusBadRequest, validateCounter,
		}
	}
	rateLimitedResponse = serverResponse{
		errors.New("too many requests"), http.StatusTooManyRequests, counter("response.errors.ratelimit"),
	}
	methodNotAllowedResponse = serverResponse{
		errors.New("only POST requests are supported"), http.StatusMethodNotAllowed, counter("response.errors.method"),
	}
	tooManyConcurrentRequestsResponse = serverResponse{
		errors.New("timeout waiting to be processed"), http.StatusServiceUnavailable, counter("response.errors.concurrency"),
	}
	fullQueueCounter  = counter("response.errors.queue")
	fullQueueResponse = func(err error) serverResponse {
		return serverResponse{
			errors.New("queue is full"), http.StatusServiceUnavailable, fullQueueCounter,
		}
	}
	serverShuttingDownCounter  = counter("response.errors.closed")
	serverShuttingDownResponse = func(err error) serverResponse {
		return serverResponse{
			errors.New("server is shutting down"), http.StatusServiceUnavailable, serverShuttingDownCounter,
		}
	}
)

func concurrencyLimitMiddleware(beaterConfig *Config) echo.MiddlewareFunc {
	return func(h echo.HandlerFunc) echo.HandlerFunc {
		semaphore := make(chan struct{}, beaterConfig.ConcurrentRequests)
		release := func() { <-semaphore }
		return func(c echo.Context) error {
			select {
			case semaphore <- struct{}{}:
				defer release()
				return h(c)
			case <-time.After(beaterConfig.MaxRequestQueueTime):
				return sendStatus(c, tooManyConcurrentRequestsResponse)
			}
		}
	}
}

type logContextKey struct{}

func loggerMiddleware() echo.MiddlewareFunc {
	return func(h echo.HandlerFunc) echo.HandlerFunc {
		logger := logp.NewLogger("request")
		return func(c echo.Context) error {
			req := c.Request()
			reqLogger := logger.With(
				"request_id", req.Header.Get(echo.HeaderXRequestID),
				"method", req.Method,
				"URL", req.URL,
				"content_length", req.ContentLength,
				"remote_address", c.RealIP(),
				"user-agent", req.Header.Get("User-Agent"))

			if err := h(c); err != nil {
				code := http.StatusInternalServerError
				if he, ok := err.(*echo.HTTPError); ok {
					code = he.Code
				}
				reqLogger.Errorw("error handling request", "response_code", code, "error", err)
				return err
			}
			reqLogger.Infow("handled request", "response_code", c.Response().Status)
			return nil
		}
	}
}

func metricsMiddleware() echo.MiddlewareFunc {
	return func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestCounter.Inc()
			err := h(c)
			responseCounter.Inc()
			if err == nil {
				responseSuccesses.Inc()
			} else {
				responseErrors.Inc()
			}
			// TODO(axw) increment response code counters
			// based on the response code (or error?)
			return err
		}
	}
}

func ipRateLimitMiddleware(rateLimit int) echo.MiddlewareFunc {
	return func(h echo.HandlerFunc) echo.HandlerFunc {
		cache, _ := lru.New(rateLimitCacheSize)

		var deny = func(ip string) bool {
			if !cache.Contains(ip) {
				cache.Add(ip, rate.NewLimiter(rate.Limit(rateLimit), rateLimit*rateLimitBurstMultiplier))
			}
			var limiter, _ = cache.Get(ip)
			return !limiter.(*rate.Limiter).Allow()
		}

		return func(c echo.Context) error {
			if deny(c.RealIP()) {
				return sendStatus(c, rateLimitedResponse)
			}
			return h(c)
		}
	}
}

func authMiddleware(secretToken string) echo.MiddlewareFunc {
	v := func(s string, c echo.Context) (bool, error) {
		return subtle.ConstantTimeCompare([]byte(s), []byte(secretToken)) == 1, nil
	}
	return middleware.KeyAuth(v)
}

// TODO(axw) this should be echo.HTTPErrorHandler
func sendStatus(c echo.Context, res serverResponse) error {
	contentType := "text/plain; charset=utf-8"
	req := c.Request()
	acceptsJSON := acceptsJSON(req)
	if acceptsJSON {
		contentType = "application/json"
	}
	resp := c.Response()
	resp.Header().Set("Content-Type", contentType)

	//res.counter.Inc()
	if res.err == nil {
		return c.NoContent(res.code)
	}

	errMsg := res.err.Error()
	if acceptsJSON {
		c.JSON(res.code, map[string]interface{}{"error": errMsg})
	} else {
		c.String(res.code, errMsg)
	}
	return res.err
}

func acceptsJSON(r *http.Request) bool {
	h := r.Header.Get("Accept")
	return strings.Contains(h, "*/*") || strings.Contains(h, "application/json")
}
