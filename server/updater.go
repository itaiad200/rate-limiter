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
	updates  map[string]access
	ch       chan userAccess

	sync.Mutex
	peersAddr []string
}

type userAccess struct {
	access
	clientID string
}

func (up *updater) Start(ctx context.Context) {
	go up.sendPeriodically(ctx)
	up.readUpdate(ctx)
}

func (up *updater) readUpdate(ctx context.Context){
	for{
		var acc userAccess
		select {
		case <-ctx.Done():
			return
		case acc = <-up.ch:
		}

		up.addToMap(acc)
	}
}

func (up *updater) addToMap(acc userAccess) {
	up.Lock()
	defer up.Unlock()

	up.updates[acc.clientID] = access{
		ts:    acc.ts,
		times: acc.times + up.updates[acc.clientID].times,
	}
}

func (up *updater) sendPeriodically(ctx context.Context){
	for{
		select {
		case <-ctx.Done():
			return
		case <-time.After(up.interval):
		}

		// send updates to all servers
		up.sendUpdates()
	}
}

func (up *updater) sendUpdates() {
	up.Lock()
	defer up.Unlock()

	if len(up.updates) == 0{
		up.logger.Debug("Nothing to update")
		return
	}

	updatesAPI := Update{}
	for id, acc := range up.updates{
		updatesAPI.UsersAccess = append(updatesAPI.UsersAccess, Access{
			ClientID:    id,
			LastRequest: acc.ts,
			Count:       acc.times,
		})
	}

	var wg sync.WaitGroup
	for _, addr := range up.peersAddr{
		wg.Add(1)
		l := addr
		go func() {
			up.sendRequest(l, updatesAPI)
			wg.Done()
		}()
	}

	wg.Wait()

	// if some server request failed, it missed the update for good
	// TODO: keep unsuccessful updates per server for next time
	up.updates = map[string]access{}
}

func (up *updater) sendRequest(addr string, updateAPI Update) {
	body, err := json.Marshal(updateAPI)
	if err != nil{
		up.logger.Error("Failed to deserialize updates", zap.Error(err))
		return
	}

	req, err := http.NewRequest("POST", "http://" +addr+updatesEP, bytes.NewBuffer(body))
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
		return
	}

	up.logger.Debug("Updated successfully", zap.Any("updates", up.updates))
}
