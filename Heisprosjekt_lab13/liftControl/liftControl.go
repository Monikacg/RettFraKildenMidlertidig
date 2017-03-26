package liftControl

import (
	"fmt"
	"sync"
	"time"

	. "../definitions"
	. "../driver"
)

func LiftControl(buttonChan chan<- Button, floorSensorChan chan<- int,
	localOrderChan <-chan Order) {
	var lcWg sync.WaitGroup
	lcWg.Add(1)

	go checkSignals(buttonChan, floorSensorChan)
	go checkForOrdersFromAdmin(localOrderChan)

	lcWg.Wait()
}

func checkForOrdersFromAdmin(localOrderChan <-chan Order) { // Nytt navn
	for {
		select {
		case msg := <-localOrderChan:
			switch msg.Category {
			case "LIGHT":
				Elev_set_button_lamp(msg.Order, msg.Floor, msg.Value)
			case "Open the door":
				Elev_set_motor_direction(DIRN_STOP)
				Elev_set_door_open_lamp(ON)
			case "DOOR":
				Elev_set_door_open_lamp(msg.Value)
			case "DIRN":
				Elev_set_motor_direction(msg.Order)
			case "Entered STUCK state, stopping engine":
				Elev_set_motor_direction(DIRN_STOP)

			case "FLOOR_LIGHT":
				Elev_set_floor_indicator(msg.Floor)
				fmt.Println("Lift: Floor light set", msg.Floor)
			}
		}
	}
}

func checkSignals(buttonChan chan<- Button, floorSensorChan chan<- int) {
	i := 0
	for {
		//fmt.Println("checkSignals")
		if i%4 == 0 {
			checkIfButtonsArePressed(buttonChan)
		}
		checkFloorSensors(floorSensorChan)
		time.Sleep(70 * time.Millisecond)
		i++
		if i == 4000 {
			i = 0
		}
	}
}

func checkIfButtonsArePressed(buttonChan chan<- Button) {
	for floor := 0; floor < N_FLOORS; floor++ {
		if Elev_get_button_signal(BUTTON_COMMAND, floor) == 1 {
			buttonChan <- Button{floor, BUTTON_COMMAND}
			// Endre "floor", se notat.txt for ekstra. (evt floor_number)
		}
		if Elev_get_button_signal(BUTTON_CALL_UP, floor) == 1 {
			buttonChan <- Button{floor, BUTTON_CALL_UP}
		}
		if Elev_get_button_signal(BUTTON_CALL_DOWN, floor) == 1 {
			buttonChan <- Button{floor, BUTTON_CALL_DOWN}
		}
	}
}

func checkFloorSensors(floorSensorChan chan<- int) {
	floor := Elev_get_floor_sensor_signal()
	if floor != -1 {
		floorSensorChan <- floor
	}
}
