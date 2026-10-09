package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/joho/godotenv"

	"internal/auth"
)

type ButtonEvents struct {
	buttonId   int
	buttonName string
	clicksCnt  int
}

type ProducerState struct {
	AsynProducer sarama.AsyncProducer
	InputCh      chan ButtonEvents
	EndCh        chan bool
}

func NewProducerState(host string, port string) (*ProducerState, error) {

	var ps *ProducerState
	asyncProducer, err := sarama.NewAsyncProducer([]string{host + ":" + port}, nil)
	if err != nil {
		return nil, err
	}
	inputCh := make(chan ButtonEvents, 0)
	endCh := make(chan bool, 0)
	ps = &ProducerState{AsynProducer: asyncProducer, InputCh: inputCh, EndCh: endCh}
	return ps, nil
}

func (ps *ProducerState) Close() {
	ps.AsynProducer.Close()
}

func (this *ProducerState) writeAsync(be ButtonEvents) {
	select {
	case this.AsynProducer.Input() <- &sarama.ProducerMessage{Topic: "buttons_events", Key: sarama.StringEncoder(be.buttonId) + ":" + be.buttonName, Value: sarama.StringEncoder(be.clicksCnt)}:

	case err := <-this.AsynProducer.Errors():
		log.Println("Failed to produce message", err)
	}
}

func (this *ProducerState) mainLoop(wg *sync.WaitGroup) {
	var be ButtonEvents
	defer wg.Done()

	for {
		select {
		case be = <-this.InputCh:
			this.writeAsync(be)
		case <-this.EndCh:
			return
		}
	}
}

func ButtonEventsHandler(ch chan ButtonEvents) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		var be ButtonEvents

		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&be)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Println("Send data to topic buttons_events")
		ch <- be
		w.WriteHeader(http.StatusOK)
	}
}

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
	var producerState *ProducerState
	args := os.Args

	if len(args) < 3 {
		fmt.Println("Please specify the Kafka IP address and port")
		return
	}
	if len(args) < 4 {
		fmt.Println("Please specify the API port")
		return
	}

	signalContex, stop := signal.NotifyContext(
		context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	kafkaHost := args[1]
	kafkaPort := args[2]
	APIPort := args[3]

	producerState, err := NewProducerState(kafkaHost, kafkaPort)
	if err != nil {
		fmt.Println("Could not create ProducerState")
		//log.Printf("")
		return
	}
	defer producerState.Close()

	err = auth.InitAuthStorage("authStoarge")
	if err != nil {
		fmt.Println("Could not init AuthStorage")
		//log.Printf("")
		return
	}

	wg.Add(1)
	go producerState.mainLoop(&wg)

	mux := http.NewServeMux()
	mux.HandleFunc("/post_buttons_events", ButtonEventsHandler(producerState.InputCh))
	mux.HandleFunc("/login", auth.LoginHandler())

	loggedMux := loggingMiddleware(mux)

	err = http.ListenAndServe(":"+APIPort, loggedMux)
	if err != nil {
		log.Printf("")
	}

	<-signalContex.Done()
	producerState.EndCh <- true
	wg.Wait()

	return
}

//curl -d '{"buttonId":"1", "buttonName":"test", "clicksCnt":"1"}' -H "Content-Type: application/json" -X POST http://localhost:22007/post_buttons_events
