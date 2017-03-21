package admin

import (
	"fmt"
	"sort"
	"time"

	. "../definitions"
	. "./calculate_order"
	. "./orders"
	. "./properties"
)

func Admin(IDInput int, buttonPressedChan <-chan Button, floorSensorTriggeredChan <-chan int,
	liftInstructionChan chan<- Instruction, adminTChan chan<- Udp, adminRChan <-chan Udp, backupTChan chan<- BackUp, backupRChan <-chan BackUp,
	peerChangeChan <-chan Peer, peerInitializeChan <-chan CurrPeers, openDoorChan chan<- string, closeDoorChan <-chan string) {

	orders := InitializeOrders()
	properties := InitializeLiftProperties()
	ownID := IDInput
	var aliveLifts []int

	// Test and try to see if this is redundant => that New/Lost would be enough
getAliveLiftsLoop:
	for {
		select {
		case totalPeers := <-peerInitializeChan:
			aliveLifts = totalPeers.Peers
			break getAliveLiftsLoop
		case <-time.After(2 * time.Second):
			break getAliveLiftsLoop
		}
	}

	inList := false
	fmt.Println("Adm: aliveLifts etter første loop: ", aliveLifts)
	for _, peer := range aliveLifts {
		if peer == ownID {
			inList = true
		}
	}

	if !inList {
		aliveLifts = append(aliveLifts, ownID)
		sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] })
	}
	fmt.Println("Adm: aliveLifts etter test om egen id finnes der: ", aliveLifts)

	//All above this line should be redundant if New/Lost is enough.

searchingForBackupLoop:
	for {
		select {
		case backup := <-backupRChan:
			orders = backup.Orders
			properties = backup.Properties
			break searchingForBackupLoop

		case peerMsg := <-peerChangeChan:
			switch peerMsg.Change {
			case "New": //Må sjekke om peer allerede er i aliveLifts
				fmt.Println("Adm: Får inn New peer. Det er: ", peerMsg.ChangedPeer)
				if len(aliveLifts)-1 == 1 {
					backupTChan <- BackUp{"I was isolated", ownID, orders, properties}
				} else {
					backupTChan <- BackUp{"I was part of a group", ownID, orders, properties}
				}
				aliveLifts = append(aliveLifts, peerMsg.ChangedPeer)
				sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] }) //Bare problem på mac?

			case "Lost":
				for i, lostPeer := range aliveLifts {
					if lostPeer == peerMsg.ChangedPeer {
						aliveLifts = append(aliveLifts[:i], aliveLifts[i+1:]...)
						DeassignOuterOrders(orders, lostPeer)
						break
					}
				}

			}

		case <-time.After(5 * time.Second):
			break searchingForBackupLoop
		}
	}

initLoop:
	for {
		select {

		case peerMsg := <-peerChangeChan:
			switch peerMsg.Change {
			case "New":
				fmt.Println("Adm: Får inn New peer. Det er: ", peerMsg.ChangedPeer)
				if len(aliveLifts) == 1 {
					backupTChan <- BackUp{"I was isolated", ownID, orders, properties}
				} else {
					backupTChan <- BackUp{"I was part of a group", ownID, orders, properties}
				}
				aliveLifts = append(aliveLifts, peerMsg.ChangedPeer)
				sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] }) //Bare problem på mac?

			case "Lost":
				fmt.Println("Adm: Får inn Lost peer. Det er: ", peerMsg.ChangedPeer)
				for i, lostPeer := range aliveLifts {
					if lostPeer == peerMsg.ChangedPeer {
						aliveLifts = append(aliveLifts[:i], aliveLifts[i+1:]...)
						DeassignOuterOrders(orders, lostPeer)
						break
					}
				}

			}

		case backupMsg := <-backupRChan:
			fmt.Println("Adm: Fått inn melding fra backupRChan, melding: ", backupMsg)
			switch backupMsg.Info {
			case "I was isolated":
				// Legg inn alle INDRE ordre for backupMsg.SenderID
				OverwriteInnerOrders(orders, ownID, backupMsg.Orders, backupMsg.SenderID)
				// Ta inn properties for backupMsg.SenderID
				SetSingleLiftProperties(properties, backupMsg.SenderID, backupMsg.Properties)

			case "I was part of a group":
				// Skriv over alt i orders minus egne indre ordre.
				OverwriteEverythingButInternalOrders(orders, ownID, backupMsg.Orders)

				// Behold egne properties, skriv over resten.
				SetPropertiesFromBackup(properties, ownID, backupMsg.Properties)

			}

		case f := <-floorSensorTriggeredChan:
			fmt.Println("Adm: initLoop, floor Sensor")
			SetLastFloor(properties, ownID, f)
			liftInstructionChan <- Instruction{"Set floor indicator light", NOT_VALID, f, ON}
			liftInstructionChan <- Instruction{"Set motor direction", DIRN_STOP, NOT_VALID, ON}
			openDoorChan <- "Opening the door now"
			adminTChan <- Udp{ownID, "Stopped", f, NOT_VALID}
			break initLoop

		case <-time.After(3 * time.Second):
			SetState(properties, ownID, MOVING)
			liftInstructionChan <- Instruction{"Set motor direction", DIRN_DOWN, NOT_VALID, NOT_VALID}
			break initLoop
		}
	}

	for {
		select {

		case button := <-buttonPressedChan:
			adminTChan <- Udp{ownID, "ButtonPressed", button.Floor, button.ButtonDirection}

		case floor := <-floorSensorTriggeredChan:
			switch GetState(properties, ownID) {
			case DOOR_OPEN:
				//Intentionally blank, probably might as well just remove this case, right now for completeness
				// Just needs to break, which it will do without these. Remove in the end?
			case IDLE:
				// See DOOR_OPEN
			case MOVING:
				if floor != GetLastFloor(properties, ownID) {
					SetLastFloor(properties, ownID, floor)
					liftInstructionChan <- Instruction{"Set floor indicator light", NOT_VALID, floor, ON}
					//fmt.Println("Adm: Verdier på vei inn i Should_stop: (orders, properties, fs, ID)")
					//fmt.Println("Adm: ", orders, properties, fs, ID)
					if ShouldStop(orders, properties, floor, ownID) == true {
						//fmt.Println("Adm: Should_stop")
						liftInstructionChan <- Instruction{"Set motor direction", DIRN_STOP, NOT_VALID, ON}
						openDoorChan <- "Opening the door now"
						adminTChan <- Udp{ownID, "Stopped", floor, NOT_VALID}
					} else {
						//fmt.Println("Adm: Should_stop NOT")
						adminTChan <- Udp{ownID, "DrovePast", floor, NOT_VALID} // ID, "kjørte forbi", etasje
						//fmt.Println("Adm: Under teit beskjed")
					}
				}
			}

		case <-closeDoorChan:
			fmt.Println("Adm: Fikk timeout")
			liftInstructionChan <- Instruction{"Close the door", NOT_VALID, NOT_VALID, NOT_VALID}

			//TURN OFF LIGHTS! //NB! SKAL IKKE SKRU AV ALLE HVIS VI IKKE TAR ALLE. EVT MÅ
			liftInstructionChan <- Instruction{"Set light in button", BUTTON_COMMAND, GetLastFloor(properties, ownID), OFF}
			liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_UP, GetLastFloor(properties, ownID), OFF}
			liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_DOWN, GetLastFloor(properties, ownID), OFF}
			//Alternativt etter hver CompleteOrders, men der blir ikke helt bra med når ordre i samme etasje.

			findNewOrder(orders, ownID, properties, aliveLifts, openDoorChan, liftInstructionChan, adminTChan)

		case msg := <-adminRChan:
			fmt.Println("Adm: Fått inn melding fra adminRChan, melding: ", msg)
			switch msg.ID {
			case ownID:
				//Alt for egen heis
				switch msg.Type {
				case "ButtonPressed":
					fmt.Println("Adm: Får tilbake fra network, ButtonPressed")
					AddOrder(orders, msg.Floor, msg.ID, msg.ButtonDirection)
					liftInstructionChan <- Instruction{"Set light in button", msg.ButtonDirection, msg.Floor, ON}
					fmt.Println("Adm: Samme loop, state og orders: ", GetState(properties, msg.ID), orders)
					if GetState(properties, msg.ID) == IDLE {
						fmt.Println("Adm: State == IDLE når knapp trykket på")
						findNewOrder(orders, msg.ID, properties, aliveLifts, openDoorChan, liftInstructionChan, adminTChan)
					}
					fmt.Println("Adm: Properties inne i samme case: ", properties)
				case "Stopped":
					SetLastFloor(properties, msg.ID, msg.Floor)
					SetState(properties, msg.ID, DOOR_OPEN)
					AssignOrders(orders, msg.Floor, msg.ID)
					CompleteOrders(orders, msg.Floor, msg.ID)
					fmt.Println("Adm: Orders at ", msg.Floor, " when I get stopped back: ", orders)
					fmt.Println("Adm: Fått Stopped tilbake. Properties: ", properties)

				case "DrovePast":
					SetState(properties, msg.ID, MOVING)
					SetLastFloor(properties, msg.ID, msg.Floor)
					fmt.Println("Adm: DrovePast kommer rundt, setter lastFloor/state=MOVING. Properties: ", properties)
				case "NewOrder":
					// Gjør alt før, er bare ack her. Skal det i det hele tatt komme tilbake hit?
					AssignOrders(orders, msg.Floor, msg.ID)
					SetState(properties, msg.ID, MOVING)
					SetDirection(properties, msg.ID, GetNewDirection(msg.Floor, GetLastFloor(properties, msg.ID)))
					liftInstructionChan <- Instruction{"Set motor direction", GetNewDirection(msg.Floor, GetLastFloor(properties, msg.ID)), NOT_VALID, NOT_VALID}
					fmt.Println("Adm: Orders at floor ", msg.Floor, " now belongs to me. Orders now: ", orders)
					fmt.Println("Adm: NewOrder kommer rundt. Properties: ", properties)
				case "Idle":
					// Samme som over. Nada.
					SetState(properties, msg.ID, IDLE)
					fmt.Println("Adm: Idle kommer rundt, setter state=IDLE. Orders, properties: ", orders, properties)
				}

			default: //Any other lift
				switch msg.Type {
				case "ButtonPressed":
					fmt.Println("Adm: Får tilbake fra network, annen heis, ButtonPressed")
					AddOrder(orders, msg.Floor, msg.ID, msg.ButtonDirection)
					if msg.ButtonDirection == BUTTON_CALL_UP || msg.ButtonDirection == BUTTON_CALL_DOWN {
						liftInstructionChan <- Instruction{"Set light in button", msg.ButtonDirection, msg.Floor, ON}
						fmt.Println("Adm: Samme loop, state og orders: ", GetState(properties, msg.ID), orders)
						if GetState(properties, ownID) == IDLE {
							fmt.Println("Adm: State == IDLE når knapp trykket på, melding fra annen heis")
							findNewOrder(orders, ownID, properties, aliveLifts, openDoorChan, liftInstructionChan, adminTChan)
						}
					}
					fmt.Println("Adm: Properties inne i samme case: ", properties)
				case "Stopped":
					fmt.Println("Adm: Får tilbake fra network, annen heis, Stopped")
					AssignOrders(orders, msg.Floor, msg.ID)
					CompleteOrders(orders, msg.Floor, msg.ID)
					liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_UP, msg.Floor, OFF}
					liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_DOWN, msg.Floor, OFF}
					SetState(properties, msg.ID, DOOR_OPEN)
					SetLastFloor(properties, msg.ID, msg.Floor)
					fmt.Println("Adm: The ID of the lift that stopped, orders, properties: ", msg.ID, orders, properties)
				case "DrovePast":
					fmt.Println("Adm: Får tilbake fra network, annen heis, DrovePast")
					SetLastFloor(properties, msg.ID, msg.Floor)
					SetState(properties, msg.ID, MOVING)
					fmt.Println("Adm: Properties inne i samme case: ", properties)

				case "NewOrder":
					fmt.Println("Adm: Får tilbake fra network, annen heis, NewOrder")
					AssignOrders(orders, msg.Floor, msg.ID)
					SetState(properties, msg.ID, MOVING)
					SetDirection(properties, msg.ID, GetNewDirection(msg.Floor, GetLastFloor(properties, msg.ID)))
					fmt.Println("Adm: Orders at floor ", msg.Floor, " now belongs to ", msg.ID, " . Orders now: ", orders)
					fmt.Println("Adm: Properties inne i samme case: ", properties)

				case "Idle":
					fmt.Println("Adm: Får tilbake fra network, annen heis, Idle")
					SetState(properties, msg.ID, IDLE)
					fmt.Println("Adm: Orders, properties inne i samme case: ", orders, properties)
				}
			}

		case backupMsg := <-backupRChan:
			fmt.Println("Adm: Fått inn melding fra backupRChan, melding: ", backupMsg)
			switch backupMsg.Info {
			case "I was isolated":
				fmt.Println("Adm: Fått ny backup (I was alone). Backupmelding: ")
				fmt.Println(backupMsg)
				fmt.Println("Adm: Orders before backupcommands: ", orders)
				// Legg inn alle INDRE ordre for backupMsg.SenderID
				OverwriteInnerOrders(orders, ownID, backupMsg.Orders, backupMsg.SenderID)
				// Ta inn properties for backupMsg.SenderID
				SetSingleLiftProperties(properties, backupMsg.SenderID, backupMsg.Properties)

				fmt.Println("Adm: Orders after backupcommands: ", orders)

			case "I was part of a group":
				fmt.Println("Adm: Fått ny backup (I was NOT alone). Backup melding: ")
				fmt.Println(backupMsg)
				fmt.Println("Adm: Orders before backupcommands: ", orders)
				// Skriv over alt i orders minus egne indre ordre.
				OverwriteEverythingButInternalOrders(orders, ownID, backupMsg.Orders)

				// Behold egne properties, skriv over resten.
				SetPropertiesFromBackup(properties, ownID, backupMsg.Properties)
				fmt.Println("Adm: Orders after backupcommands: ", orders)

			}

		case peerMsg := <-peerChangeChan:
			switch peerMsg.Change {
			case "New":
				fmt.Println("Adm: Får inn New peerID. Det er: ", peerMsg.ChangedPeer)
				if len(aliveLifts) == 1 {
					backupTChan <- BackUp{"I was isolated", ownID, orders, properties}
				} else {
					backupTChan <- BackUp{"I was part of a group", ownID, orders, properties}
				}
				aliveLifts = append(aliveLifts, peerMsg.ChangedPeer)
				sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] }) //MÅ FIKSES. NB NB NB

			case "Lost":
				fmt.Println("Adm: Får inn Lost peer. Det er: ", peerMsg.ChangedPeer)
				for i, n := range aliveLifts {
					if n == peerMsg.ChangedPeer {
						lostPeer := n
						aliveLifts = append(aliveLifts[:i], aliveLifts[i+1:]...)
						DeassignOuterOrders(orders, lostPeer)
						if GetState(properties, ownID) == IDLE {
							fmt.Println("Adm: State == IDLE, en annen heis er død => kan være nye ordre")
							findNewOrder(orders, ownID, properties, aliveLifts, openDoorChan, liftInstructionChan, adminTChan)
						}
						break
					}
				}

			}

		}
	}
}

func findNewOrder(orders [][]int, ownID int, properties []int, aliveLifts []int, openDoorChan chan<- string,
	liftInstructionChan chan<- Instruction, adminTChan chan<- Udp) {
	fmt.Println("Adm: Inne i findNewOrder. Orders, properties: ", orders, properties)

	newDirn, dest := CalculateNextOrder(orders, ownID, properties, aliveLifts)

	// Default dest and newDirn returned has to be undefined (-2,-2)
	fmt.Println("Adm: Got new direction", newDirn, dest)
	if newDirn == DIRN_STOP {
		fmt.Println("Adm: I DIRN_STOP for findNewOrder")
		liftInstructionChan <- Instruction{"Set floor indicator light", NOT_VALID, GetLastFloor(properties, ownID), ON}
		liftInstructionChan <- Instruction{"Set motor direction", DIRN_STOP, NOT_VALID, ON}
		openDoorChan <- "Opening the door now"
		adminTChan <- Udp{ownID, "Stopped", GetLastFloor(properties, ownID), NOT_VALID}
	} else if newDirn == DIRN_DOWN || newDirn == DIRN_UP {
		fmt.Println("Adm: I DIRN_DOWN/DIRN_UP for findNewOrder")
		adminTChan <- Udp{ownID, "NewOrder", dest, NOT_VALID} // ownID, "Moving, desting (new order)", etasje
	} else { // newDirn == -2 (NOT_VALID)
		fmt.Println("Adm: I IDLE for findNewOrder")
		adminTChan <- Udp{ownID, "Idle", dest, NOT_VALID} // ownID, "IDLE", etasje
	}
	fmt.Println("Adm: På vei ut av findNewOrder. Orders, properties: ", orders, properties)
}
