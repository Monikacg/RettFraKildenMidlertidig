package timer

import (
	"time"
	//"fmt" Bare for test
)

func Timer(startTimerChan <-chan string, timeOutChan chan<- string) { //, interrupt_timer_chan <-chan string
	go timer(startTimerChan, timeOutChan) // Trengs egentlig "go" her? allerede kalt som goroutine fra main
} //, interrupt_timer_chan

func timer(startTimerChan <-chan string, timeOutChan chan<- string) { //, interrupt_timer_chan <-chan string

	for {
		select {
		case <-startTimerChan:
			go door_open_timer(timeOutChan)
		}
	}
}

func door_open_timer(timeOutChan chan<- string) {
	for {
		select {
		case <-time.After(3 * time.Second):
			timeOutChan <- "DOOR_OPEN"
			return
		}
	}
}

/*
func udp_timer(timeOutChan chan<- string)  { //Må testes på nytt //, interrupt_timer_chan <-chan string
  udp_time_out := time.NewTimer(100*time.Millisecond).C // Skal være lengre
  for {
    select {
    case <- udp_time_out:
      timeOutChan <- "UDP"
      return
    //case <- interrupt_timer_chan:
      //return
    }
  }
}

*/
// Note: Dette vil enten blokkere sin egen goroutine (som vi vil)
// eller blokkere hele (som vi ikke vil). Test og finn ut
// => implementer overalt hvis det fungerer.
