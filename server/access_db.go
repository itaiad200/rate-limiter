package server

import (
	"github.com/EagleChen/mapmutex"
	"go.uber.org/zap"
	"sync"
	"time"
)

// accessDB manages accesses to the server
type accessDB struct {
	logger        *zap.Logger

	// accessMap handles locks by client
	accessMap *mapmutex.Mutex

	// accessCount keeps the per-client-access data.
	// sync.Map is optimized for "multiple goroutines read, write, and overwrite entries for disjoint
	// sets of keys". With the accessMap above, we can guarantee that once a request is being handled,
	// other requests of the same client will wait.
	accessCount sync.Map

	// number of allowed accesses in the timeslot window
	reqThresehold int
	window        time.Duration

	updater *updater
}



// accessRecord is the record saved in the db
type accessRecord struct {
	requests []access
	requestsInWindow int
}

type access struct{
	t time.Time
	times int
}

func newAccessDB(logger *zap.Logger, up *updater, reqThresehold int, window time.Duration) *accessDB {
	return &accessDB{
		logger: logger,
		accessCount:   sync.Map{},
		updater: up,

		// TODO: custom mapmutex args?
		accessMap: mapmutex.NewCustomizedMapMutex(
			// max retries
			1000,
			// 5 second max delays
			5000000000,
			//  10 nanosecond base delay
		 10,
		 	// factor
		 1.1,
		 	// jitter
		  0.2),
		reqThresehold: reqThresehold,
		window:        window,
	}
}

func (db *accessDB) handleMultipleAccess(update Update) {
	for clientID, acc := range update.UsersAccess{
		// locking by client id, handles retries
		if db.accessMap.TryLock(clientID){
			defer db.accessMap.Unlock(clientID)

			accessRec, loaded := db.loadOrStore(clientID, acc.LastRequest, acc.Count)
			if !loaded{
				// first time client access
				continue
			}

			// Not iterating through the entire array saves us time,
			// but we are paying with slightly less accurate window.
			accessRec.requestsInWindow += acc.Count
			accessRec.requests = append(accessRec.requests, access{
				t:     acc.LastRequest,
				times: acc.Count,
			})
			db.accessCount.Store(clientID, accessRec)
		}
	}
}

func (db *accessDB) handleSingleAccess(clientID string) bool {
	// single timestamp for the record
	now := time.Now()

	// locking by client id, handles retries
	if db.accessMap.TryLock(clientID){
		defer db.accessMap.Unlock(clientID)

		if db.checkAndStoreAccess(clientID, now, 1){
			db.updater.addToNextUpdate(clientID, now)
			return true
		}

		return false
	}

	db.logger.Error("Lock wasn't acquired under 5 seconds, blocking access")
	return false
}

func (db *accessDB) checkAndStoreAccess(clientID string, now time.Time, times int) bool {
	accessRec, loaded := db.loadOrStore(clientID, now, times)
	if !loaded{
		// first client access
		return true
	}

	// find the first access in the window.
	// although this might look like an O(n) algorithm for each request,
	// we'll only loop stale requests once.
	i := 0
	var acc access
	for i, acc = range accessRec.requests {
		if acc.t.Add(db.window).After(now) {
			// first request in window
			break
		}
		accessRec.requestsInWindow -= acc.times
	}

	// trim stale times
	accessRec.requests = accessRec.requests[i:]

	if accessRec.requestsInWindow >= db.reqThresehold {
		// over the limit
		db.accessCount.Store(clientID, accessRec)
		return false
	}

	// client is under her limit
	accessRec.requests = append(accessRec.requests, access{t: now, times: times})
	accessRec.requestsInWindow += times
	db.accessCount.Store(clientID, accessRec)

	return true
}

func (db *accessDB) loadOrStore(clientID string, now time.Time, times int) (accessRecord, bool) {
	newWindowRecord := accessRecord{
		requests:         []access{{t: now, times: times}},
		requestsInWindow: times,
	}

	rec, loaded := db.accessCount.LoadOrStore(clientID, newWindowRecord)

	if !loaded {
		// first request by client
		return accessRecord{}, false
	}

	accessRec, ok := rec.(accessRecord)
	if !ok {
		db.logger.Error("Wrong record type in access db")
		return accessRec, true
	}

	return accessRec, true
}
