package producer

import (
	"log"
	"strconv"
	"sync"

	"github.com/IBM/sarama"
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

	key := strconv.Itoa(be.buttonId) + ":" + be.buttonName
	select {
	case this.AsynProducer.Input() <- &sarama.ProducerMessage{Topic: "buttons_events", Key: sarama.StringEncoder(key), Value: sarama.StringEncoder(be.clicksCnt)}:

	case err := <-this.AsynProducer.Errors():
		log.Println("Failed to produce message", err)
	}
}

func (this *ProducerState) MainLoop(wg *sync.WaitGroup) {
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
