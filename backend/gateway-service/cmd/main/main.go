package main

import (
	"context"
	"fmt"
	"log"
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
	authService := auth.NewAuthService(os.Getenv("DATABASE_URL"), os.Getenv("JWT_SECRET"), 15*time.Minute)
	if authService == nil {
		fmt.Println("Could not init auth service")
		return
	}
	defer authService.DestroyAuthService()

	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(authService.ServiceRepo)
	buttonEvents := handlers.NewUserHandler(producerState.InputCh)

	r.HandleFunc("/api/auth/register", authHandler.Register).Methods("POST")
	r.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST")
	r.HandleFunc("/api/auth/refresh", authHandler.RefreshToken).Methods("POST")
	r.HandleFunc("/api/post_buttons_events", buttonEvents.ButtonEventsHandler).Methods("POST")
	r.Use(loggingMiddleware)

	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(middleware.AuthMiddleware(authService))
	protected.HandleFunc("/profile", userHandler.Profile).Methods("GET")

	err = http.ListenAndServe(":"+APIPort, r)
	if err != nil {
		log.Printf("")
	}

	<-signalContex.Done()
	producerState.EndCh <- true
	wg.Wait()

	return
}

//curl -d '{"buttonId":"1", "buttonName":"test", "clicksCnt":"1"}' -H "Content-Type: application/json" -X POST http://localhost:22007/post_buttons_events
