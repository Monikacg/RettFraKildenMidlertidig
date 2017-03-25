package timer

// MAIN FUNCTION USED TO TEST TIMER
// Må testes på nytt, lagt til ekstra funksjonalitet i udp_timer

import (
	"fmt"
	"testing"
	"time"
)

func TestFn(t *testing.T) {
	start_timer_chan := make(chan string, 100)
	time_out_chan := make(chan string, 100)

	go Timer_init(start_timer_chan, time_out_chan)

	go listen(time_out_chan)

	for i := 0; i < 5; i++ {
		start_timer_chan <- "DOOR_OPEN"
		fmt.Println("Starting timer for DOOR_OPEN")

		//start_timer_chan <- "UDP"
		//fmt.Println("Starting timer for UDP")

		time.Sleep(5 * time.Second)
	}
}

func listen(time_out_chan <-chan string) {
	for {
		select {
		case m := <-time_out_chan:
			if m == "DOOR_OPEN" {
				fmt.Println("DOOR_OPEN")
			}
			//if m == "UDP" {
			//  fmt.Println("UDP")
			//}
		}
	}
}
