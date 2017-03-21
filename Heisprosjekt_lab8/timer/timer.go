package timer

import (
	"time"
)

// Timer is used to determine how long a door should stay open.
func Timer(openDoorChan <-chan string, closeDoorChan chan<- string) {
	for {
		select {
		case <-openDoorChan:
			go startDoorOpenTimer(closeDoorChan)
		}
	}
}

func startDoorOpenTimer(closeDoorChan chan<- string) {
	for {
		select {
		case <-time.After(3 * time.Second):
			closeDoorChan <- "Close the door now"
			return
		}
	}
}
