package main

import (
	"flag"
	"github.com/itaiad200/rate-limiter/server"
	"log"
	"strings"
	"time"
)

func main() {
	addr := flag.String("address", ":8080", "Address for the server to listen to")
	windowInSec := flag.Int64("window", 60, "Time window for each client requests in seconds")
	maxReq := flag.Int("max-requests", 500, "Maximum number of requests in the window")
	verbose := flag.Bool("verbose", false, "Verbose prints debug logs")
	peers := flag.String("peers", "", "Comma-delimited addresses of the server peers")

	flag.Parse()

	// input validation
	// addr is validated when the server tries to listen
	if windowInSec == nil || *windowInSec < 0 {
		log.Fatal("window must be a positive integer")
	}
	if maxReq == nil || *maxReq < 0 {
		log.Fatal("maxReq must be a positive integer")
	}

	server.Start(server.Config{
		Addr:      *addr,
		Window:    time.Duration(*windowInSec) * time.Second,
		RequestTH: *maxReq,
		Verbose:   *verbose,
		PeersAddr: strings.Split(*peers,","),
	})
}
