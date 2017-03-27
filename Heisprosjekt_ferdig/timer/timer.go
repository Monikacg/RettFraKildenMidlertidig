package timer

import (
	"time"
)

// This module is used only to determine how long the door should stay open and
// how long we will wait until we do our reset task (drive down until we find a floor, 0 if no order at the others)
// after we have entered the STUCK state.

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
		case <-time.After(10 * time.Second):
			timeOutChan <- "Time to exit STUCK state and see if the engine is working"
			return
		}
	}
}
