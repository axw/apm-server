package beater

import (
	"expvar"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/elastic/apm-agent-go/module/apmecho"
	conf "github.com/elastic/apm-server/config"
	"github.com/elastic/apm-server/decoder"
	"github.com/elastic/apm-server/processor"
	perr "github.com/elastic/apm-server/processor/error"
	"github.com/elastic/apm-server/processor/sourcemap"
	"github.com/elastic/apm-server/processor/transaction"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

type ProcessorFactory func() processor.Processor

type serverResponse struct {
	err     error
	code    int
	counter *monitoring.Int
}

func newHandler(beaterConfig *Config, report reporter) http.Handler {
	e := echo.New()
	e.Use(apmecho.Middleware())
	e.Use(metricsMiddleware())
	e.Use(middleware.RequestID())
	e.Use(loggerMiddleware())

	// Based on echo.DefaultHTTPErrorHandler; no logging of errors
	// unless there is a failure to send.
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		var (
			code = http.StatusInternalServerError
			msg  interface{}
		)
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			msg = he.Message
			if he.Inner != nil {
				msg = fmt.Sprintf("%v, %v", err, he.Inner)
			}
		} else if e.Debug {
			msg = err.Error()
		} else {
			msg = http.StatusText(code)
		}
		if _, ok := msg.(string); ok {
			msg = echo.Map{"message": msg}
		}
		if !c.Response().Committed {
			if c.Request().Method == echo.HEAD {
				err = c.NoContent(code)
			} else {
				err = c.JSON(code, msg)
			}
			if err != nil {
				e.Logger.Error(err)
			}
		}
	}

	e.GET("/healthcheck", healthCheckHandler)

	intake := e.Group("")
	intake.Use(concurrencyLimitMiddleware(beaterConfig))
	if beaterConfig.SecretToken != "" {
		intake.Use(authMiddleware(beaterConfig.SecretToken))
	}

	v1 := intake.Group("/v1")
	v1.POST("/transactions", backendHandler(transaction.NewProcessor, beaterConfig, report))
	v1.POST("/errors", backendHandler(perr.NewProcessor, beaterConfig, report))

	corsMiddleware := middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: beaterConfig.Frontend.AllowOrigins,
		AllowMethods: []string{echo.POST, echo.OPTIONS},
		AllowHeaders: []string{"Content-Type", "Content-Encoding", "Accept"},
		MaxAge:       3600,
	})
	v1frontend := v1.Group(
		"/client-side",
		corsMiddleware,
		ipRateLimitMiddleware(beaterConfig.Frontend.RateLimit),
	)
	v1frontend.Match([]string{echo.OPTIONS, echo.POST}, "/transactions", frontendHandler(transaction.NewProcessor, beaterConfig, report))
	v1frontend.Match([]string{echo.OPTIONS, echo.POST}, "/errors", frontendHandler(perr.NewProcessor, beaterConfig, report))
	v1.POST("/client-side/sourcemaps", sourcemapHandler(sourcemap.NewProcessor, beaterConfig, report))

	if beaterConfig.Expvar.isEnabled() {
		e.GET(beaterConfig.Expvar.Url, echo.WrapHandler(expvar.Handler()))
	}

	logger := logp.NewLogger("handler")
	for _, route := range e.Routes() {
		logger.Infof("Route %q added: %s %s", route.Name, route.Method, route.Path)
	}
	return e
}

func backendHandler(pf ProcessorFactory, beaterConfig *Config, report reporter) echo.HandlerFunc {
	decoder := decoder.DecodeSystemData(decoder.DecodeLimitJSONData(beaterConfig.MaxUnzippedSize), beaterConfig.AugmentEnabled)
	return processRequestHandler(pf, conf.Config{}, report, decoder)
}

func frontendHandler(pf ProcessorFactory, beaterConfig *Config, report reporter) echo.HandlerFunc {
	if !beaterConfig.Frontend.isEnabled() {
		return func(echo.Context) error {
			return echo.ErrForbidden
		}
	}
	smapper, err := beaterConfig.Frontend.memoizedSmapMapper()
	if err != nil {
		logp.NewLogger("handler").Error(err.Error())
	}
	config := conf.Config{
		SmapMapper:          smapper,
		LibraryPattern:      regexp.MustCompile(beaterConfig.Frontend.LibraryPattern),
		ExcludeFromGrouping: regexp.MustCompile(beaterConfig.Frontend.ExcludeFromGrouping),
	}
	decoder := decoder.DecodeUserData(decoder.DecodeLimitJSONData(beaterConfig.MaxUnzippedSize), beaterConfig.AugmentEnabled)
	return processRequestHandler(pf, config, report, decoder)
}

func sourcemapHandler(pf ProcessorFactory, beaterConfig *Config, report reporter) echo.HandlerFunc {
	if !beaterConfig.Frontend.isEnabled() {
		return func(echo.Context) error {
			return echo.ErrForbidden
		}
	}
	smapper, err := beaterConfig.Frontend.memoizedSmapMapper()
	if err != nil {
		logp.NewLogger("handler").Error(err.Error())
	}
	return processRequestHandler(pf, conf.Config{SmapMapper: smapper}, report, decoder.DecodeSourcemapFormData)
}

func healthCheckHandler(c echo.Context) error {
	return c.NoContent(200)
}

func processRequestHandler(pf ProcessorFactory, config conf.Config, report reporter, decode decoder.Decoder) echo.HandlerFunc {
	return func(c echo.Context) error {
		res := processRequest(c.Request(), pf, config, report, decode)
		return sendStatus(c, res)
	}
}

func processRequest(r *http.Request, pf ProcessorFactory, config conf.Config, report reporter, decode decoder.Decoder) serverResponse {
	processor := pf()

	data, err := decode(r)
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			return requestTooLargeResponse
		}
		return cannotDecodeResponse(err)

	}

	if err = processor.Validate(data); err != nil {
		return cannotValidateResponse(err)
	}

	payload, err := processor.Decode(data)
	if err != nil {
		return cannotDecodeResponse(err)
	}

	if err = report(pendingReq{payload: payload, config: config}); err != nil {
		if strings.Contains(err.Error(), "publisher is being stopped") {
			return serverShuttingDownResponse(err)
		}
		return fullQueueResponse(err)
	}

	return acceptedResponse
}
