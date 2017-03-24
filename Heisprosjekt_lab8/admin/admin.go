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
	liftInstructionChan chan<- Instruction, adminTChan chan<- Message, adminRChan <-chan Message, backupTChan chan<- BackUp, backupRChan <-chan BackUp,
	peerChangeChan <-chan Peer, peerInitializeChan <-chan CurrPeers, startTimerChan chan<- string, timeOutChan <-chan string) {

	orders := InitializeOrders()
	properties := InitializeLiftProperties()
	ownID := IDInput
	var aliveLifts []int

	//var stuckTimer *time.Timer
	//const stuckTimeout = 10 * time.Second

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
						DeassignOrders(orders, lostPeer)
						break
					}
				}

			}

		case <-time.After(5 * time.Second):
			break searchingForBackupLoop
		}
	}

initLoop: // Endre til case f := floorSensorTriggeredChan og den med After blir uten after, bare i default. Sikkert lurt å passe på New/Lost in that case.
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
				sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] })

			case "Lost":
				fmt.Println("Adm: Får inn Lost peer. Det er: ", peerMsg.ChangedPeer)
				for i, lostPeer := range aliveLifts {
					if lostPeer == peerMsg.ChangedPeer {
						aliveLifts = append(aliveLifts[:i], aliveLifts[i+1:]...)
						DeassignOrders(orders, lostPeer)
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
			startTimerChan <- "Opening the door now"
			adminTChan <- Message{"Stopped at floor", ownID, f, NOT_VALID}
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
			adminTChan <- Message{"Button pressed", ownID, button.Floor, button.ButtonDirection}

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
						startTimerChan <- "Opening the door now"
						adminTChan <- Message{"Stopped at floor", ownID, floor, NOT_VALID}
					} else {
						//fmt.Println("Adm: Should_stop NOT")
						adminTChan <- Message{"Drove past floor", ownID, floor, NOT_VALID} // ID, "kjørte forbi", etasje
						//fmt.Println("Adm: Under teit beskjed")
					}
				}
			}

		case timeOut := <-timeOutChan:
			fmt.Println("Adm: Fikk timeout")
			switch timeOut {
			case "Time to close the door":
				liftInstructionChan <- Instruction{"Close the door", NOT_VALID, NOT_VALID, NOT_VALID}

				//TURN OFF LIGHTS! //NB! SKAL IKKE SKRU AV ALLE HVIS VI IKKE TAR ALLE. EVT MÅ
				liftInstructionChan <- Instruction{"Set light in button", BUTTON_COMMAND, GetLastFloor(properties, ownID), OFF}
				liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_UP, GetLastFloor(properties, ownID), OFF}
				liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_DOWN, GetLastFloor(properties, ownID), OFF}
				//Alternativt etter hver CompleteOrders, men der blir ikke helt bra med når ordre i samme etasje.

				findNewOrder(orders, ownID, properties, aliveLifts, startTimerChan, liftInstructionChan, adminTChan)
			case "Time to exit STUCK state and see if the engine is working":
				select {
				case f := <-floorSensorTriggeredChan:
					fmt.Println("Adm: Stuck exited, floor Sensor")
					SetLastFloor(properties, ownID, f)
					liftInstructionChan <- Instruction{"Set floor indicator light", NOT_VALID, f, ON}
					liftInstructionChan <- Instruction{"Set motor direction", DIRN_STOP, NOT_VALID, ON}
					startTimerChan <- "Opening the door now"
					adminTChan <- Message{"Stopped at floor", ownID, f, NOT_VALID}

				default:
					SetState(properties, ownID, MOVING)
					SetDirection(properties, ownID, DIRN_DOWN)
					liftInstructionChan <- Instruction{"Set motor direction", DIRN_DOWN, NOT_VALID, NOT_VALID}
					//stuckTimer = time.NewTimer(stuckTimeout)
				}
			}

		case msg := <-adminRChan:
			fmt.Println("Adm: Fått inn melding fra adminRChan, melding: ", msg)
			switch msg.ID {
			case ownID:
				//Alt for egen heis
				switch msg.Info {
				case "Button pressed":
					fmt.Println("Adm: Får tilbake fra network, ButtonPressed")
					AddOrder(orders, msg.Floor, msg.ID, msg.ButtonDirection)
					liftInstructionChan <- Instruction{"Set light in button", msg.ButtonDirection, msg.Floor, ON}
					fmt.Println("Adm: Samme loop, state og orders: ", GetState(properties, msg.ID), orders)
					if GetState(properties, msg.ID) == IDLE {
						fmt.Println("Adm: State == IDLE når knapp trykket på")
						findNewOrder(orders, msg.ID, properties, aliveLifts, startTimerChan, liftInstructionChan, adminTChan)
					}
					fmt.Println("Adm: Properties inne i samme case: ", properties)
				case "Stopped at floor":
					SetLastFloor(properties, msg.ID, msg.Floor)
					SetState(properties, msg.ID, DOOR_OPEN)
					AssignOrders(orders, msg.Floor, msg.ID)
					CompleteOrders(orders, msg.Floor, msg.ID)
					fmt.Println("Adm: Orders at ", msg.Floor, " when I get stopped back: ", orders)
					fmt.Println("Adm: Fått Stopped tilbake. Properties: ", properties)
					/*
						if AnyAssignedOrdersLeft(orders, msg.ID) {
							stuckTimer = time.NewTimer(stuckTimeout)
						} else {
							stuckTimer.Stop()
						}*/

				case "Drove past floor":
					SetState(properties, msg.ID, MOVING)
					SetLastFloor(properties, msg.ID, msg.Floor)
					fmt.Println("Adm: DrovePast kommer rundt, setter lastFloor/state=MOVING. Properties: ", properties)
				case "Got assigned a new order":
					AssignOrders(orders, msg.Floor, msg.ID)
					SetState(properties, msg.ID, MOVING)
					SetDirection(properties, msg.ID, GetNewDirection(msg.Floor, GetLastFloor(properties, msg.ID)))
					liftInstructionChan <- Instruction{"Set motor direction", GetNewDirection(msg.Floor, GetLastFloor(properties, msg.ID)), NOT_VALID, NOT_VALID}
					//stuckTimer = time.NewTimer(stuckTimeout)

					fmt.Println("Adm: Orders at floor ", msg.Floor, " now belongs to me. Orders now: ", orders)
					fmt.Println("Adm: NewOrder kommer rundt. Properties: ", properties)

				case "I'm stuck":
					DeassignOrders(orders, msg.ID)
					SetState(properties, msg.ID, STUCK)
					startTimerChan <- "Entered STUCK state"

				case "Entered IDLE state":
					SetState(properties, msg.ID, IDLE)
					fmt.Println("Adm: Idle kommer rundt, setter state=IDLE. Orders, properties: ", orders, properties)
				}

			default: //Any other lift
				switch msg.Info {
				case "Button pressed":
					fmt.Println("Adm: Får tilbake fra network, annen heis, ButtonPressed")
					AddOrder(orders, msg.Floor, msg.ID, msg.ButtonDirection)
					if msg.ButtonDirection == BUTTON_CALL_UP || msg.ButtonDirection == BUTTON_CALL_DOWN {
						liftInstructionChan <- Instruction{"Set light in button", msg.ButtonDirection, msg.Floor, ON}
						fmt.Println("Adm: Samme loop, state og orders: ", GetState(properties, msg.ID), orders)
						if GetState(properties, ownID) == IDLE {
							fmt.Println("Adm: State == IDLE når knapp trykket på, melding fra annen heis")
							findNewOrder(orders, ownID, properties, aliveLifts, startTimerChan, liftInstructionChan, adminTChan)
						}
					}
					fmt.Println("Adm: Properties inne i samme case: ", properties)
				case "Stopped at floor":
					fmt.Println("Adm: Får tilbake fra network, annen heis, Stopped")
					AssignOrders(orders, msg.Floor, msg.ID)
					CompleteOrders(orders, msg.Floor, msg.ID)
					liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_UP, msg.Floor, OFF}
					liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_DOWN, msg.Floor, OFF}
					SetState(properties, msg.ID, DOOR_OPEN)
					SetLastFloor(properties, msg.ID, msg.Floor)
					fmt.Println("Adm: The ID of the lift that stopped, orders, properties: ", msg.ID, orders, properties)
				case "Drove past floor":
					fmt.Println("Adm: Får tilbake fra network, annen heis, DrovePast")
					SetLastFloor(properties, msg.ID, msg.Floor)
					SetState(properties, msg.ID, MOVING)
					fmt.Println("Adm: Properties inne i samme case: ", properties)

				case "Got assigned a new order":
					fmt.Println("Adm: Får tilbake fra network, annen heis, NewOrder")
					AssignOrders(orders, msg.Floor, msg.ID)
					SetState(properties, msg.ID, MOVING)
					SetDirection(properties, msg.ID, GetNewDirection(msg.Floor, GetLastFloor(properties, msg.ID)))
					fmt.Println("Adm: Orders at floor ", msg.Floor, " now belongs to ", msg.ID, " . Orders now: ", orders)
					fmt.Println("Adm: Properties inne i samme case: ", properties)

				case "I'm stuck":
					DeassignOrders(orders, msg.ID)
					SetState(properties, msg.ID, STUCK)

				case "Entered IDLE state":
					fmt.Println("Adm: Får tilbake fra network, annen heis, Idle")
					SetState(properties, msg.ID, IDLE)
					fmt.Println("Adm: Orders, properties inne i samme case: ", orders, properties)
				}
			}
			/*
				case <-stuckTimer.C:
					adminTChan <- Message{"I'm stuck", ownID, GetLastFloor(properties, ownID), NOT_VALID}
			*/

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
						DeassignOrders(orders, lostPeer)
						if GetState(properties, ownID) == IDLE {
							fmt.Println("Adm: State == IDLE, en annen heis er død => kan være nye ordre")
							findNewOrder(orders, ownID, properties, aliveLifts, startTimerChan, liftInstructionChan, adminTChan)
						}
						break
					}
				}

			}

		}
	}
}

func findNewOrder(orders [][]int, ownID int, properties []int, aliveLifts []int, startTimerChan chan<- string,
	liftInstructionChan chan<- Instruction, adminTChan chan<- Message) {
	fmt.Println("Adm: Inne i findNewOrder. Orders, properties: ", orders, properties)

	newDirn, destination := CalculateNextOrder(orders, properties, aliveLifts, ownID)

	// Default destination and newDirn returned has to be undefined (-2,-2)
	fmt.Println("Adm: Got new direction", newDirn, destination)
	if newDirn == DIRN_STOP {
		fmt.Println("Adm: I DIRN_STOP for findNewOrder")
		liftInstructionChan <- Instruction{"Set floor indicator light", NOT_VALID, GetLastFloor(properties, ownID), ON}
		liftInstructionChan <- Instruction{"Set motor direction", DIRN_STOP, NOT_VALID, ON}
		startTimerChan <- "Opening the door now"
		adminTChan <- Message{"Stopped at floor", ownID, GetLastFloor(properties, ownID), NOT_VALID}
	} else if newDirn == DIRN_DOWN || newDirn == DIRN_UP {
		fmt.Println("Adm: I DIRN_DOWN/DIRN_UP for findNewOrder")
		adminTChan <- Message{"Got assigned a new order", ownID, destination, NOT_VALID} // ownID, "Moving, desting (new order)", etasje
	} else {
		fmt.Println("Adm: I IDLE for findNewOrder")
		adminTChan <- Message{"Entered IDLE state", ownID, destination, NOT_VALID} // ownID, "IDLE", etasje
	}
	fmt.Println("Adm: På vei ut av findNewOrder. Orders, properties: ", orders, properties)
}
