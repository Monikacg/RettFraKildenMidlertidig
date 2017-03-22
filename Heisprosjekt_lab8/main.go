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

/*
ID = 0: 129.241.187.158 (vanlig pc)
ID = 1: 129.241.187.156 (bak oss)
ID = 2: 129.241.187.159 (til høyre)


For ID = 1:
ssh Student@129.241.187.156
scp -r /home/student/Desktop/RettFraKilden2-master/Heisprosjekt_lab4 Student@129.241.187.156:/home/student/Desktop/RettFraKilden2-master/Heisprosjekt_lab4
Endre plassering så du finner main.go
go run main.go -id=1

ITK2o17-kyb

For ID = 2:
ssh Student@129.241.187.159
scp -r /home/student/Desktop/RettFraKilden2-master/Heisprosjekt_lab4 Student@129.241.187.159:/home/student/Desktop/RettFraKilden2-master/Heisprosjekt_lab4
Endre plassering så du finner main.go
go run main.go -id=2

*/

func main() {

	//go run main.go -id=our_id

	buttonPressedChan := make(chan Button) // Endre til asynkron, siden du kan trykke inn en ytre knapp, så ville ha en lengre ned? test
	floorSensorTriggeredChan := make(chan int)

	liftInstructionChan := make(chan Instruction) //----""-----

	adminTChan := make(chan Udp, 100) // Må være asynkron
	adminRChan := make(chan Udp, 100) // ----"-------"---
	backupTChan := make(chan BackUp, 100)
	backupRChan := make(chan BackUp, 100)
	peerChangeChan := make(chan Peer, 100)

	peerInitializeChan := make(chan CurrPeers, 100)

	startTimerChan := make(chan string, 10)
	timeOutChan := make(chan string, 10)

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
