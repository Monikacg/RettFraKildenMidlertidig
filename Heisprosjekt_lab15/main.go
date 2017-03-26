package main

import (
	"fmt"
	"sync"

	. "./admin"
	. "./definitions"
	. "./driver"
	. "./liftControl"
	. "./network"
	. "./timer"
)

func main() {

	buttonPressedChan := make(chan Button)
	floorSensorTriggeredChan := make(chan int)

	liftInstructionChan := make(chan Instruction)

	outgoingMessageChan := make(chan Message, 100)
	incomingMessageChan := make(chan Message, 100)

	incomingBackupChan := make(chan BackUp, 100)
	outgoingBackupChan := make(chan BackUp, 100)

	aliveLiftChangeChan := make(chan ChangedLift, 100)

	startTimerChan := make(chan string)
	timeOutChan := make(chan string)

	var IDInput int
	fmt.Scanf("%d", &IDInput)

	Elev_init()
	var wg sync.WaitGroup
	wg.Add(1)

	go LiftControl(buttonPressedChan, floorSensorTriggeredChan, liftInstructionChan)

	go Network(IDInput, outgoingMessageChan, incomingMessageChan, outgoingBackupChan, incomingBackupChan, aliveLiftChangeChan)

	go Admin(IDInput, buttonPressedChan, floorSensorTriggeredChan,
		liftInstructionChan, outgoingMessageChan, incomingMessageChan, outgoingBackupChan, incomingBackupChan, aliveLiftChangeChan, startTimerChan, timeOutChan)

	go Timer(startTimerChan, timeOutChan)

	wg.Wait()
}
