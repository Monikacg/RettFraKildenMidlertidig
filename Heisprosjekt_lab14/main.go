package main

import (
	"flag"
	"fmt"
	"strconv"
	"sync"

	. "./admin"
	. "./definitions"
	. "./driver"
	. "./liftControl"
	. "./network"
	. "./timer"
)

func main() {

	//go run main.go -id=our_id

	buttonPressedChan := make(chan Button)
	floorSensorTriggeredChan := make(chan int)

	liftInstructionChan := make(chan Instruction)

	outgoingMessageChan := make(chan Message, 100) // Må være asynkron
	incomingMessageChan := make(chan Message, 100) // ----"-------"---

	incomingBackupChan := make(chan BackUp, 100)
	outgoingBackupChan := make(chan BackUp, 100)

	aliveLiftChangeChan := make(chan ChangedLift, 100)

	startTimerChan := make(chan string)
	timeOutChan := make(chan string)

	// NB NB NB!!!! HUSK Å SETT INN "SKRIV INN ID HER".
	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	if id == "" {
		fmt.Println("Main: You should have entered an ID. Now set to 0, no matter what the others are")
		id = "0"
	}

	IDInput, _ := strconv.Atoi(id)

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
