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

	buttonChan := make(chan Button)
	floorSensorChan := make(chan int)

	localOrderChan := make(chan Order) //----""-----

	adminTChan := make(chan Udp, 100) // Må være asynkron
	adminRChan := make(chan Udp, 100) // ----"-------"---
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

	go LiftControl(buttonChan, floorSensorChan, localOrderChan)

	go Network(IDInput, adminTChan, adminRChan, backupTChan, backupRChan, peerChangeChan, peerInitializeChan)

	go Admin(IDInput, buttonChan, floorSensorChan,
		localOrderChan, adminTChan, adminRChan, backupTChan, backupRChan, peerChangeChan,
		peerInitializeChan, startTimerChan, timeOutChan)

	go Timer(startTimerChan, timeOutChan)
	wg.Wait()
}
