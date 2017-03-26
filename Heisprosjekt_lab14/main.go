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

	adminTChan := make(chan Message, 100) // Må være asynkron
	adminRChan := make(chan Message, 100) // ----"-------"---
	backupTChan := make(chan BackUp, 100)
	backupRChan := make(chan BackUp, 100)
	peerChangeChan := make(chan Peer, 100)

	peerInitializeChan := make(chan CurrPeers, 100)

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

	go Network(IDInput, adminTChan, adminRChan, backupTChan, backupRChan, peerChangeChan, peerInitializeChan)

	go Admin(IDInput, buttonPressedChan, floorSensorTriggeredChan,
		liftInstructionChan, adminTChan, adminRChan, backupTChan, backupRChan, peerChangeChan,
		peerInitializeChan, startTimerChan, timeOutChan)

	go Timer(startTimerChan, timeOutChan)
	wg.Wait()
}
