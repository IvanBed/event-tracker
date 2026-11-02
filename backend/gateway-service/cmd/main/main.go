package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"gateway/internal/auth"
	"gateway/internal/handlers"
	"gateway/internal/middleware"
	"gateway/internal/producer"
)

const (
	_shutdownPeriod      = 15 * time.Second
	_shutdownHardPeriod  = 3 * time.Second
	_readinessDrainDelay = 5 * time.Second
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("🚀 Старт обработки %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("🏁 Завершено за %v", time.Since(start))
	})
}

func loadEnv() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	requiredVars := []string{"DATABASE_URL", "JWT_SECRET"}
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			log.Fatalf("Required environment variable %s is not set", v)
		}
	}
}

func main() {
	var wg sync.WaitGroup
	var producerState *producer.ProducerState
	args := os.Args

	if len(args) < 3 {
		fmt.Println("Please specify the Kafka IP address and port")
		return
	}
	if len(args) < 4 {
		fmt.Println("Please specify the API port")
		return
	}

	signalContex, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	loadEnv()

	kafkaHost := args[1]
	kafkaPort := args[2]
	APIPort := args[3]

	producerState, err := producer.NewProducerState(kafkaHost, kafkaPort)
	if err != nil {
		fmt.Println("Could not create ProducerState")
		//log.Printf("")
		return
	}
	defer producerState.Close()
	wg.Add(1)
	go producerState.MainLoop(&wg)

	r := mux.NewRouter()

	fmt.Println("DATABASE_URL ", os.Getenv("DATABASE_URL"))
	fmt.Println("JWT_SECRET ", os.Getenv("JWT_SECRET"))

	authService := auth.NewAuthService(os.Getenv("DATABASE_URL"), os.Getenv("JWT_SECRET"), 15*time.Minute)
	if authService == nil {
		fmt.Println("Could not init auth service")
		return
	}
	defer authService.DestroyAuthService()

	authHandler := handlers.NewAuthHandler(authService)
	buttonEvents := handlers.NewEventsHandler(producerState.InputCh)

	r.HandleFunc("/api/auth/register", authHandler.Register).Methods("POST")
	r.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST")
	r.HandleFunc("/api/auth/refresh", authHandler.RefreshToken).Methods("POST")
	r.HandleFunc("/api/post_buttons_events", buttonEvents.ButtonEventsHandler).Methods("POST")
	r.Use(loggingMiddleware)

	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(middleware.AuthMiddleware(authService))
	//protected.HandleFunc("/profile", authService.Profile).Methods("GET")

	ongoingCtx, stopOngoingGracefully := context.WithCancel(context.Background())
	server := &http.Server{
		Addr:    ":" + APIPort,
		Handler: r,
		BaseContext: func(_ net.Listener) context.Context {
			return ongoingCtx
		},
	}
	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Println("Server starting on :" + APIPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	<-signalContex.Done()
	stop()
	producerState.EndCh <- true
	log.Println("Received shutdown signal, shutting down.")

	time.Sleep(_readinessDrainDelay)
	log.Println("Readiness check propagated, now waiting for ongoing requests to finish.")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), _shutdownPeriod)
	defer cancel()

	err = server.Shutdown(shutdownCtx)
	stopOngoingGracefully()
	if err != nil {
		log.Println("Failed to wait for ongoing requests to finish, waiting for forced cancellation.")
		time.Sleep(_shutdownHardPeriod)
	}
	log.Println("Server shut down gracefully.")
	wg.Wait()

	return
}

//curl -d '{"buttonId":"1", "buttonName":"test", "clicksCnt":"1"}' -H "Content-Type: application/json" -X POST http://localhost:20007/api/post_buttons_events
/*
curl -X POST http://localhost:20007/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "ServiceName": "test",
    "Password": "SecureP@ssw0rd!"
  }'*/
