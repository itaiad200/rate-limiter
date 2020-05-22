package server

import (
	"context"
	"github.com/itaiad200/rate-limiter/block"
	"github.com/itaiad200/rate-limiter/log"
	"go.uber.org/zap"
	"net/http"
	"time"
)

// Config is the rate-limiter server configuration
type Config struct {
	// The address that the server should listen to
	Addr string

	// RequestTH is maximum allowed requests per client in the Window
	RequestTH int

	// Window is the time frame where client requests are counted
	Window time.Duration

	// Print debug logs to console
	Verbose bool

	// Addresses of all other instances of this server.
	// Lookup is normally the way to go here.
	PeersAddr []string
}

// Start starts the server
func Start(config Config) {
	logger := log.New(config.Verbose)
	defer logger.Sync()

	done := make(chan bool, 1)
	quit := make(chan bool, 1)

	ctx, cancel := context.WithCancel(context.Background())
	updater := &updater{
		logger:    logger,
		interval:  5 * time.Second,
		peersAddr: config.PeersAddr,
		updates:   map[string]access{},
		ch:        make(chan userAccess, 1000),
	}
	go updater.Start(ctx)

	server := newWebserver(ctx, updater, logger, config)

	// making sure the server drains requests before shutting down completely
	go gracefullShutdown(server, logger, quit, done)

	// exit on Enter press
	go func() {
		block.Enter()
		cancel()
		quit <- true
	}()

	logger.Info("Server is ready to handle requests", zap.String("Addr", config.Addr))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Failed to listen", zap.String("Addr", config.Addr), zap.Error(err))
	}

	<-done
	logger.Info("Server stopped")
}

func gracefullShutdown(server *http.Server, logger *zap.Logger, quit chan bool, done chan<- bool) {
	<-quit
	logger.Info("Server is shutting down...")

	// 5 seconds is more than enough to drain our simple server.
	// We can always make this configurable later on.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Could not gracefully shutdown the server: %v", zap.Error(err))
	}
	close(done)
}

// updatesEP the endpoint of the updates sent between servers
const updatesEP = "/updates"

func newWebserver(ctx context.Context, up *updater, logger *zap.Logger, config Config) *http.Server {
	router := http.NewServeMux()


	// requestsHandler is handling a customer calling any of our API.
	// TODO: if we had real API calls here, this should be registered as a middleware
	requestsHandler := &handler{
		logger,
		newAccessDB(logger, up, config.RequestTH, config.Window),
	}

	// handle clients requests
	router.HandleFunc("/foo", requestsHandler.foo)
	router.HandleFunc(updatesEP, requestsHandler.updates)

	return &http.Server{
		Addr:    config.Addr,
		Handler: router,
	}
}
