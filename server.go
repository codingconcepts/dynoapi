package dynoapi

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// Server holds the routes available for extending the
// API surface.
type Server struct {
	router         *echo.Echo
	ssl            bool
	certsDir       string
	host           string
	port           int
	routes         RouteConfigs
	buildVersion   string
	buildTimestamp string
}

// NewServer returns a pointer to a new instance of Server.
func NewServer(host string, port int, options ...Option) (s *Server) {
	s = &Server{
		host: host,
		port: port,
	}

	router := echo.New()
	s.router = router
	s.router.Use(middleware.Recover())

	l := limiter.New(nil).SetMax(1).SetBurst(1).SetMessage("Too many requests.")
	s.router.Use(limitMiddleware(l))

	s.router.GET("/version", s.GetVersion)
	s.router.OPTIONS("/", s.GetConfig)
	s.router.POST("/", s.AddRoute)

	for _, option := range options {
		if err := option(s); err != nil {
			log.Fatal(err)
		}
	}

	return
}

func limitMiddleware(lmt *limiter.Limiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return echo.HandlerFunc(func(c echo.Context) error {
			httpError := tollbooth.LimitByRequest(lmt, c.Response(), c.Request())
			if httpError != nil {
				return c.String(httpError.StatusCode, httpError.Message)
			}
			return next(c)
		})
	}
}

// ServeHTTP serves from the server's router.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Start start the server's router.
func (s *Server) Start() (err error) {
	s.router.Server.ReadTimeout = 5 * time.Second
	s.router.Server.WriteTimeout = 5 * time.Second
	s.router.Server.IdleTimeout = 10 * time.Second

	if !s.ssl {
		return s.router.Start(fmt.Sprintf(":%d", s.port))
	}

	s.router.AutoTLSManager.HostPolicy = autocert.HostWhitelist(s.host)
	s.router.AutoTLSManager.Cache = autocert.DirCache(s.certsDir)
	s.router.AutoTLSManager.Prompt = autocert.AcceptTOS

	s.router.Server.TLSConfig = &tls.Config{
		GetCertificate:           s.router.AutoTLSManager.GetCertificate,
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
	}

	return s.router.StartAutoTLS(fmt.Sprintf(":%d", s.port))
}

// Stop stops the server's router.
func (s *Server) Stop() (err error) {
	return s.router.Close()
}

// AddRoute allows a user to add a new API route.
func (s *Server) AddRoute(c echo.Context) (err error) {
	var route RouteConfig
	if err = c.Bind(&route); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	s.add(route)
	return c.String(http.StatusOK, "successfully added route")
}

// GetVersion displays the build version and timestamp
// of the server.
func (s *Server) GetVersion(c echo.Context) (err error) {
	return c.String(http.StatusOK, fmt.Sprintf("%s\n%s", s.buildVersion, s.buildTimestamp))
}

// GetConfig displays the available routes.
func (s *Server) GetConfig(c echo.Context) (err error) {
	return c.JSON(http.StatusOK, s.routes)
}

func (s *Server) add(route RouteConfig) {
	// if a body template has been provided, parse it up-front
	// to make calls to the dynamic endpoint quicker.
	if route.Body != "" {
		route.BodyTemplate = template.Must(template.New(route.Body).Parse(route.Body))
	}

	handler := s.routeHandler(route)
	handlerOptions := routeHandlerOptions(route)

	// Add the dynamic route and an OPTIONS route so the caller
	// can find out how to interact with the route.
	s.router.Add(route.Method, route.URI, handler)
	s.router.OPTIONS(route.URI, handlerOptions)

	// keep track of the route within the server
	s.routes.Merge(route)
}

func (s *Server) routeHandler(r RouteConfig) func(echo.Context) error {
	return func(c echo.Context) (err error) {
		body := ParseArgs(c)

		if err = s.sleep(body, r, c); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}

		// if a body hasn't be configured, don't bother continuing
		if r.Body == "" {
			return c.String(r.StatusCode, "")
		}

		if r.BodyContentType != "" {
			c.Response().Header().Set("Content-Type", r.BodyContentType)
		}

		c.Response().WriteHeader(r.StatusCode)
		template, err := r.BodyTemplate.Parse(r.Body)
		if err = template.Execute(c.Response().Writer, body); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}

		return
	}
}

func routeHandlerOptions(r RouteConfig) func(echo.Context) error {
	return func(c echo.Context) (err error) {
		return c.JSON(http.StatusOK, r)
	}
}

func (s *Server) sleep(args map[string]interface{}, r RouteConfig, c echo.Context) (err error) {
	if r.DurationParam == "" {
		return
	}

	rawDuration, ok := args[r.DurationParam]
	if !ok {
		return
	}

	duration, err := time.ParseDuration(rawDuration.(string))
	if err != nil {
		return
	}

	time.Sleep(duration)
	return
}
