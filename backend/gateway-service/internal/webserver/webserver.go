package webserver

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

const (
	_shutdownPeriod      = 15 * time.Second
	_shutdownHardPeriod  = 3 * time.Second
	_readinessDrainDelay = 5 * time.Second
)

type ServerState struct {
	Instance              *http.Server
	OngoingCtx            context.Context
	StopOngoingGracefully context.CancelFunc
	Port                  string
}

func NewServerState(APIPort string, router *mux.Router) *ServerState {
	ongoingCtx, stopOngoingGracefully := context.WithCancel(context.Background())
	server := &http.Server{
		Addr:    ":" + APIPort,
		Handler: router,
		BaseContext: func(_ net.Listener) context.Context {
			return ongoingCtx
		},
	}
	return &ServerState{Instance: server, OngoingCtx: ongoingCtx, StopOngoingGracefully: stopOngoingGracefully, Port: APIPort}
}

func (server *ServerState) DestroyServerState() {
	server.Instance.Close()
}

func (server *ServerState) GracefulShutdown() {
	log.Println("Received shutdown signal, shutting down.")

	time.Sleep(_readinessDrainDelay)
	log.Println("Readiness check propagated, now waiting for ongoing requests to finish.")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), _shutdownPeriod)
	defer cancel()

	err := server.Instance.Shutdown(shutdownCtx)
	server.StopOngoingGracefully()
	if err != nil {
		log.Println("Failed to wait for ongoing requests to finish, waiting for forced cancellation.")
		time.Sleep(_shutdownHardPeriod)
	}
	log.Println("Server shut down gracefully.")
}

func (server *ServerState) MainLoop(wg *sync.WaitGroup) {
	defer wg.Done()

	log.Println("Server starting on :" + server.Port)
	if err := server.Instance.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
