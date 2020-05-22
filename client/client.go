package client

import (
	"context"
	"fmt"
	"github.com/itaiad200/rate-limiter/server"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"time"
)

type client struct {
	logger *zap.Logger

	// client simulates this clientID
	clientID   int

	// Listening server address
	serverAddr string
}

// 240 milliseconds means an average of 500 QPM, which makes the probability of
// rejection (after a minute) in somewhere around 50%
const maxSleepInMS = 240

func (c *client) run(ctx context.Context) {
	addr := fmt.Sprintf("%s/?%s=%d", c.serverAddr, server.ClientIDURLParam, c.clientID)
	done := false

	for !done {
		resp, err := http.Get(addr)
		if err == nil {
			c.logger.Debug("Received response",
				zap.Int("ClientID", c.clientID), zap.Int("StatusCode", resp.StatusCode))
		} else {
			c.logger.Warn("client error", zap.Error(err))
		}

		sleep := time.Duration(rand.Intn(maxSleepInMS)) * time.Millisecond
		c.logger.Debug("Client sleeping",
			zap.Int("ClientID", c.clientID), zap.Duration("sleepTime", sleep))

		select {
		case <-ctx.Done():
			done = true
		case <-time.After(sleep):
		}
	}
}
