package handlers

import (
	"encoding/json"
	"fmt"
	"gateway/internal/producer"
	"net/http"
)

/*type ButtonEvents struct {
	buttonId   int
	buttonName string
	clicksCnt  int
}*/

type EventsHandler struct {
	buttonEventsCh chan producer.ButtonEvents
}

func NewEventsHandler(buttonEventsCh chan producer.ButtonEvents) *EventsHandler {
	return &EventsHandler{
		buttonEventsCh: buttonEventsCh,
	}
}

func (eventsHandler *EventsHandler) ButtonEventsHandler(w http.ResponseWriter, r *http.Request) {

	var be producer.ButtonEvents

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
	eventsHandler.buttonEventsCh <- be
	w.WriteHeader(http.StatusOK)
}
