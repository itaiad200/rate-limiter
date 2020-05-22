package server

import (
	"bytes"
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"
)

// updater is in charge of updating all the server replicas on
// the current state of clients requests
type updater struct {
	logger *zap.Logger

	interval time.Duration
	updates map[string]Access
	sync.Mutex
	peersAddr []string
}

func (up *updater) addToNextUpdate(clientID string, t time.Time){
	up.Lock()
	defer up.Unlock()

	up.updates[clientID] = Access{
		LastRequest: t,
		Count:       up.updates[clientID].Count + 1,
	}
}

func (up *updater) Run(ctx context.Context) error{
	for{
		select {
		case <-ctx.Done():
			return ctx.Err()
			case <-time.After(up.interval):
		}

		// send updates to all servers
		up.sendUpdates()
	}
}

func (up *updater) sendUpdates() {
	up.Lock()
	defer up.Unlock()

	var wg sync.WaitGroup

	for _, addr := range up.peersAddr{
		wg.Add(1)
		l := addr
		go func() {
			up.sendRequest(l)
			wg.Done()
		}()
	}

	wg.Wait()

	// if some server request failed, it missed the update for good
	// TODO: keep unsuccessful updates per server for next time
	up.updates = nil
}

func (up *updater) sendRequest(addr string) {
	body, err := json.Marshal(up.updates)
	if err != nil{
		up.logger.Error("Failed to deserialize updates", zap.Error(err))
		return
	}

	req, err := http.NewRequest("POST", addr+updatesEP, bytes.NewBuffer(body))
	if err != nil{
		up.logger.Error("Failed to create request", zap.Error(err))
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		// TODO: configureable?
		Timeout: 2 *time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		up.logger.Error("Failed to send request", zap.Error(err))
		return
	}

	if resp.StatusCode != http.StatusOK{
		up.logger.Error("Bad server response", zap.Int("StatusCode", resp.StatusCode))
	}
}
