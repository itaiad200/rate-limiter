package client

import (
	"context"
	"github.com/itaiad200/rate-limiter/block"
	"github.com/itaiad200/rate-limiter/log"
	"sync"
)

// Config is the rate-limiter client configuration
type Config struct {
	// The server address
	ServerAddr string

	// Number of client to invoke
	NumClients int

	// Maximum id of any client
	MaxClientID int

	// Verbose prints debug logs
	Verbose     bool
}

func Run(config Config) {
	var wg sync.WaitGroup

	logger := log.New(config.Verbose)
	defer logger.Sync()

	// Blocking until Enter pressed
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		block.Enter()
		cancel()
	}()

	for i := 0; i < config.NumClients; i++ {
		client := client{
			logger:     logger,
			clientID:   i%config.MaxClientID + 1,
			serverAddr: config.ServerAddr,
		}

		wg.Add(1)
		go func() {
			client.run(ctx)
			wg.Done()
		}()
	}

	// Waiting for clients to drain
	wg.Wait()
}
