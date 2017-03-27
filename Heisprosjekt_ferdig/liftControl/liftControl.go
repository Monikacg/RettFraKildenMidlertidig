package liftControl

import (
	"sync"
	"time"

	. "../definitions"
	. "../driver"
)

func LiftControl(buttonPressedChan chan<- Button, floorSensorTriggeredChan chan<- int,
	liftInstructionChan <-chan Instruction) {
	var lcWg sync.WaitGroup
	lcWg.Add(1)

	go checkSignals(buttonPressedChan, floorSensorTriggeredChan)
	go checkForInstructionsFromAdmin(liftInstructionChan)

	lcWg.Wait()
}

func checkForInstructionsFromAdmin(liftInstructionChan <-chan Instruction) {
	for {
		select {
		case newInstruction := <-liftInstructionChan:
			switch newInstruction.Category {
			case "Set light in button":
				Elev_set_button_lamp(newInstruction.Order, newInstruction.Floor, newInstruction.Value)
			case "Open the door":
				Elev_set_motor_direction(DIRN_STOP)
				Elev_set_door_open_lamp(ON)
			case "Close the door":
				Elev_set_door_open_lamp(OFF)
			case "Set motor direction":
				Elev_set_motor_direction(newInstruction.Order)
			case "Entered STUCK state, stopping engine":
				Elev_set_motor_direction(DIRN_STOP)
			case "Set floor indicator light":
				Elev_set_floor_indicator(newInstruction.Floor)
			}
		}
	}
}

func checkSignals(buttonPressedChan chan<- Button, floorSensorTriggeredChan chan<- int) {
	for {
		checkIfButtonsArePressed(buttonPressedChan)
		checkFloorSensors(floorSensorTriggeredChan)
		time.Sleep(70 * time.Millisecond)
	}
}

func checkIfButtonsArePressed(buttonPressedChan chan<- Button) {
	for floor := 0; floor < N_FLOORS; floor++ {
		if Elev_get_button_signal(BUTTON_COMMAND, floor) == 1 {
			buttonPressedChan <- Button{floor, BUTTON_COMMAND}
		}
		if Elev_get_button_signal(BUTTON_CALL_UP, floor) == 1 {
			buttonPressedChan <- Button{floor, BUTTON_CALL_UP}
		}
		if Elev_get_button_signal(BUTTON_CALL_DOWN, floor) == 1 {
			buttonPressedChan <- Button{floor, BUTTON_CALL_DOWN}
		}
	}
}

func checkFloorSensors(floorSensorTriggeredChan chan<- int) {
	floor := Elev_get_floor_sensor_signal()
	if floor != -1 {
		floorSensorTriggeredChan <- floor
	}
}
