package timer

import (
	"time"
)

// Timer is used to determine how long a door should stay open.
func Timer(startTimerChan <-chan string, timeOutChan chan<- string) {
	for {
		select {
		case newStart := <-startTimerChan:
			switch newStart {
			case "Opening the door now":
				go startDoorOpenTimer(timeOutChan)
			case "Entered STUCK state":
				go startStuckWaitingPeriodTimer(timeOutChan)
			}

		}
	}
}

func startDoorOpenTimer(timeOutChan chan<- string) {
	for {
		select {
		case <-time.After(3 * time.Second):
			timeOutChan <- "Time to close the door"
			return
		}
	}
}

func startStuckWaitingPeriodTimer(timeOutChan chan<- string) {
	for {
		select {
		case <-time.After(15 * time.Second):
			timeOutChan <- "Time to exit STUCK state and see if the engine is working"
			return
		}
	}
}
