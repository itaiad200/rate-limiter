package main

import (
	"flag"
	"github.com/itaiad200/rate-limiter/client"
	"log"
	"math"
)

func main() {
	numClients := flag.Int("num-of-clients", 100, "Number of clients to simulate")
	maxClientID := flag.Int("max-client-id", math.MaxInt32, "Maximum client id (Used for creating multiple client per id)")
	serverAddr := flag.String("address", "http://localhost:8080/foo", "The clients endpoint of the server")
	verbose := flag.Bool("verbose", false, "Verbose prints debug logs")
	flag.Parse()

	if numClients == nil || *numClients <= 0 {
		log.Fatal("number of clients must be a positive number")
	}

	client.Run(client.Config{
		ServerAddr: *serverAddr,
		NumClients: *numClients,
		MaxClientID: *maxClientID,
		Verbose:    *verbose,
	})
}
