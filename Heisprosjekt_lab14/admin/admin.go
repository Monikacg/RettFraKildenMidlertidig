package admin

import (
	"fmt"
	"time"

	"sort"

	. "../definitions"
	. "./calculate_order"
	. "./properties"
	. "./orders"
)

func Admin(IDInput int, buttonPressedChan <-chan Button, floorSensorTriggeredChan <-chan int,
	liftInstructionChan chan<- Instruction, outgoingMessageChan chan<- Message, incomingMessageChan <-chan Message, outgoingBackupChan chan<- BackUp, incomingBackupChan <-chan BackUp,
	aliveLiftChangeChan <-chan ChangedLift, startTimerChan chan<- string, timeOutChan <-chan string) {

	const stuckTimeout = 10 * time.Second

	orders := InitializeOrders()
	properties := InitializeLiftProperties()
	ID := IDInput

	var aliveLifts []int
	lastBackUpRecevied := make([]BackUp, MAX_N_LIFTS)
	for i := range lastBackUpRecevied {
		lastBackUpRecevied[i].Orders = InitializeOrders()
		lastBackUpRecevied[i].Properties = InitializeLiftProperties()
	}
	stuckTimer := time.NewTimer(stuckTimeout)


searchingForInitialBackupLoop:
	for {
		select {

		case backup := <-incomingBackupChan:
			orders = backup.Orders
			properties = backup.Properties

			// No one else would have the right information about our properties, so reset those:
			SetState(properties, ID, INIT)
			SetDirn(properties, ID, DIRN_DOWN)
			SetLastFloor(properties, ID, NOT_VALID)
			break searchingForInitialBackupLoop

		case liftChange := <-aliveLiftChangeChan:
			switch liftChange.TypeOfChange {
			case "New": //Må sjekke om peer allerede er i aliveLifts
				fmt.Println("Adm: Får inn New peer. Det er: ", liftChange.IDofChangedLift)
				outgoingBackupChan <- BackUp{"IWasAlone", ID, orders, properties}
				aliveLifts = append(aliveLifts, liftChange.IDofChangedLift)
				sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] }) //Bare problem på mac?

			case "Lost":
				for i, lostPeer := range aliveLifts {
					if lostPeer == liftChange.IDofChangedLift {
						aliveLifts = append(aliveLifts[:i], aliveLifts[i+1:]...)
						if lostPeer != ID {
							DeassignOrders(orders, lostPeer)
						}
						break
					}
				}
			}

		case <-time.After(5 * time.Second):
			break searchingForInitialBackupLoop
		}
	}

initLoop:
	for {
		select {

		case floor := <-floorSensorTriggeredChan:
			fmt.Println("Adm: initLoop, floor Sensor") // NOTE: NEEDS DIRN_DOWN SET FURTHER UP.
			SetLastFloor(properties, ID, floor)
			liftInstructionChan <- Instruction{"Set floor indicator light", NOT_VALID, floor, ON}
			liftInstructionChan <- Instruction{"Open the door", DIRN_STOP, NOT_VALID, NOT_VALID}
			startTimerChan <- "Opening the door now"
			outgoingMessageChan <- Message{"Stopped", ID, floor, NOT_VALID}
			stuckTimer.Stop()
			break initLoop

		default:
			SetState(properties, ID, MOVING)
			liftInstructionChan <- Instruction{"Set motor direction", DIRN_DOWN, NOT_VALID, NOT_VALID}
			stuckTimer = time.NewTimer(stuckTimeout)
			break initLoop
		}
	}


	for {
		select {

		case button := <-buttonPressedChan:
			if !isButtonAlreadyRegistrered(orders, button, ID) {
				outgoingMessageChan <- Message{"ButtonPressed", ID, button.Floor, button.ButtonType}
			}

		case floor := <-floorSensorTriggeredChan:
			switch GetState(properties, ID) {
			case MOVING:
				if floor != GetLastFloor(properties, ID) {
					SetLastFloor(properties, ID, floor)
					liftInstructionChan <- Instruction{"Set floor indicator light", NOT_VALID, floor, ON}
					//fmt.Println("Adm: Verdier på vei inn i Should_stop: (orders, properties, fs, ID)")
					//fmt.Println("Adm: ", orders, properties, fs, ID)
					if ShouldStop(orders, properties, floor, ID) == true {
						//fmt.Println("Adm: Should_stop")
						liftInstructionChan <- Instruction{"Open the door", DIRN_STOP, NOT_VALID, NOT_VALID}
						startTimerChan <- "Opening the door now"
						outgoingMessageChan <- Message{"Stopped", ID, floor, NOT_VALID}
					} else {
						//fmt.Println("Adm: Should_stop NOT")
						outgoingMessageChan <- Message{"DrovePast", ID, floor, NOT_VALID} // ID, "kjørte forbi", etasje
						//fmt.Println("Adm: Under teit beskjed")
					}
				}
			}

		case timeOut := <-timeOutChan:
			fmt.Println("Adm: Fikk timeout")
			switch timeOut {
			case "Time to close the door":
				liftInstructionChan <- Instruction{"Close the door", NOT_VALID, NOT_VALID, NOT_VALID}

				// Turning off lights for this floor, as we complete the orders here.
				liftInstructionChan <- Instruction{"Set light in button", BUTTON_COMMAND, GetLastFloor(properties, ID), OFF}
				liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_UP, GetLastFloor(properties, ID), OFF}
				liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_DOWN, GetLastFloor(properties, ID), OFF}

				findNewOrder(orders, ID, properties, aliveLifts, startTimerChan, liftInstructionChan, outgoingMessageChan)

			case "Time to exit STUCK state and see if the engine is working":
				select {
				case floor := <-floorSensorTriggeredChan:
					fmt.Println("Adm: Stuck exited, floor Sensor")
					SetLastFloor(properties, ID, floor)
					liftInstructionChan <- Instruction{"Set floor indicator light", NOT_VALID, floor, ON}
					liftInstructionChan <- Instruction{"Open the door", DIRN_STOP, NOT_VALID, NOT_VALID}
					startTimerChan <- "Opening the door now"
					outgoingMessageChan <- Message{"Stopped", ID, floor, NOT_VALID}

				default:
					SetState(properties, ID, MOVING)
					SetDirn(properties, ID, DIRN_DOWN)
					liftInstructionChan <- Instruction{"Set motor direction", DIRN_DOWN, NOT_VALID, NOT_VALID}
					stuckTimer = time.NewTimer(stuckTimeout)
				}
			}

		case m := <-incomingMessageChan:
			fmt.Println("Adm: Fått inn melding fra incomingMessageChan, melding: ", m)
			switch m.ID {
			case ID:
				//Alt for egen heis
				switch m.Info {
				case "ButtonPressed":
					fmt.Println("Adm: Får tilbake fra network, ButtonPressed")
					AddOrder(orders, m.Floor, m.ID, m.ButtonType)
					liftInstructionChan <- Instruction{"Set light in button", m.ButtonType, m.Floor, ON}
					fmt.Println("Adm: Samme loop, state og orders: ", GetState(properties, ID), orders)
					if GetState(properties, ID) == IDLE {
						fmt.Println("Adm: State == IDLE når knapp trykket på")
						findNewOrder(orders, ID, properties, aliveLifts, startTimerChan, liftInstructionChan, outgoingMessageChan)
					}
					fmt.Println("Adm: Properties inne i samme case: ", properties)
				case "Stopped":
					SetLastFloor(properties, ID, m.Floor)
					SetState(properties, ID, DOOR_OPEN)
					AssignOrders(orders, m.Floor, ID) // også nederst nå.
					CompleteOrders(orders, m.Floor, ID)
					fmt.Println("Adm: Orders at ", m.Floor, " when I get stopped back: ", orders)
					fmt.Println("Adm: Fått Stopped tilbake. Properties: ", properties)
					if !AnyAssignedOrdersLeft(orders, m.ID) {
						stuckTimer.Stop()
					} else {
						stuckTimer = time.NewTimer(stuckTimeout)
					}

				case "DrovePast":
					SetLastFloor(properties, m.ID, m.Floor)
					SetState(properties, m.ID, MOVING)
					stuckTimer = time.NewTimer(stuckTimeout)
					fmt.Println("Adm: DrovePast kommer rundt, setter lastFloor/state=MOVING. Properties: ", properties)
				case "NewOrder":
					SetState(properties, ID, MOVING)
					SetDirn(properties, ID, GetNewDirection(m.Floor, GetLastFloor(properties, ID)))
					liftInstructionChan <- Instruction{"Set motor direction", GetNewDirection(m.Floor, GetLastFloor(properties, ID)), NOT_VALID, NOT_VALID}
					fmt.Println("Adm: Orders at floor ", m.Floor, " now belongs to me. Orders now: ", orders)
					fmt.Println("Adm: NewOrder kommer rundt. Properties: ", properties)
					stuckTimer = time.NewTimer(stuckTimeout)

				case "I'm stuck":
					SetState(properties, m.ID, STUCK)
					DeassignOrders(orders, m.ID)
					startTimerChan <- "Entered STUCK state"
					liftInstructionChan <- Instruction{"Entered STUCK state, stopping engine", DIRN_STOP, NOT_VALID, NOT_VALID}

				case "Idle":
					SetState(properties, m.ID, IDLE)
					fmt.Println("Adm: Idle kommer rundt, setter state=IDLE. Orders, properties: ", orders, properties)
				}

			default: //Any other lift
				switch m.Info {
				case "ButtonPressed":
					fmt.Println("Adm: Får tilbake fra network, annen heis, ButtonPressed")
					AddOrder(orders, m.Floor, m.ID, m.ButtonType)
					if m.ButtonType == BUTTON_CALL_UP || m.ButtonType == BUTTON_CALL_DOWN {
						liftInstructionChan <- Instruction{"Set light in button", m.ButtonType, m.Floor, ON}
						fmt.Println("Adm: Samme loop, state og orders: ", GetState(properties, ID), orders)
						if GetState(properties, ID) == IDLE {
							fmt.Println("Adm: State == IDLE når knapp trykket på, melding fra annen heis")
							findNewOrder(orders, ID, properties, aliveLifts, startTimerChan, liftInstructionChan, outgoingMessageChan)
						}
					}
					fmt.Println("Adm: Properties inne i samme case: ", properties)
				case "Stopped":
					fmt.Println("Adm: Får tilbake fra network, annen heis, Stopped")
					SetLastFloor(properties, m.ID, m.Floor)
					SetState(properties, m.ID, DOOR_OPEN)
					AssignOrders(orders, m.Floor, m.ID)
					CompleteOrders(orders, m.Floor, m.ID)
					liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_UP, m.Floor, OFF}
					liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_DOWN, m.Floor, OFF}
					fmt.Println("Adm: The ID of the lift that stopped, orders, properties: ", m.ID, orders, properties)
				case "DrovePast":
					fmt.Println("Adm: Får tilbake fra network, annen heis, DrovePast")
					SetLastFloor(properties, m.ID, m.Floor)
					SetState(properties, m.ID, MOVING)
					fmt.Println("Adm: Properties inne i samme case: ", properties)

				case "NewOrder":
					fmt.Println("Adm: Får tilbake fra network, annen heis, NewOrder")
					SetState(properties, m.ID, MOVING)
					SetDirn(properties, m.ID, GetNewDirection(m.Floor, GetLastFloor(properties, m.ID)))
					AssignOrders(orders, m.Floor, m.ID)
					fmt.Println("Adm: Orders at floor ", m.Floor, " now belongs to ", m.ID, " . Orders now: ", orders)
					fmt.Println("Adm: Properties inne i samme case: ", properties)

				case "I'm stuck":
					SetState(properties, m.ID, STUCK)
					DeassignOrders(orders, m.ID)
					if GetState(properties, ID) == IDLE {
						fmt.Println("Adm: State == IDLE når en annen er STUCK ")
						findNewOrder(orders, ID, properties, aliveLifts, startTimerChan, liftInstructionChan, outgoingMessageChan)
					}

				case "Idle":
					fmt.Println("Adm: Får tilbake fra network, annen heis, Idle")
					SetState(properties, m.ID, IDLE)
					fmt.Println("Adm: Orders, properties inne i samme case: ", orders, properties)
				}
			}

		case <-stuckTimer.C:
			// Sets Last floor to a value that is not equal 0 so the lift will stop no matter where it gets stuck.
			SetLastFloor(properties, ID, NOT_VALID)
			outgoingMessageChan <- Message{"I'm stuck", ID, GetLastFloor(properties, ID), NOT_VALID}

		case backupMsg := <-incomingBackupChan:
			fmt.Println("Adm: Fått inn melding fra incomingBackupChan, melding: ", backupMsg)
			if ID != backupMsg.SenderID {
				if !backupsAreIdentical(backupMsg, lastBackUpRecevied[backupMsg.SenderID]) {
					lastBackUpRecevied[backupMsg.SenderID] = backupMsg
					switch backupMsg.Info {
					case "IWasAlone":
						fmt.Println("Adm: Fått ny backup (I was alone). Backupmelding: ", backupMsg)
						fmt.Println("Adm: Orders before backupcommands: ", orders)
						// Legg inn alle INDRE ordre for backupMsg.SenderID
						CopyInnerOrders(orders, backupMsg.Orders)
						// Ta inn properties for backupMsg.SenderID
						SetSingleLiftProperties(properties, backupMsg.SenderID, backupMsg.Properties)

						fmt.Println("Adm: Orders after backupcommands: ", orders)

					case "IWasNotAlone":
						fmt.Println("Adm: Fått ny backup (I was NOT alone). Backupmelding: ", backupMsg)
						fmt.Println("Adm: Orders before backupcommands: ", orders)
						// Skriv over alt i orders minus egne indre ordre.
						OverwriteEverythingButInternalOrders(orders, backupMsg.Orders)

						// Behold egne properties, skriv over resten.
						SetPropertiesFromBackup(properties, ID, backupMsg.Properties)
						fmt.Println("Adm: Orders after backupcommands: ", orders)

					}
				}
			}

		case liftChange := <-aliveLiftChangeChan:
			switch liftChange.TypeOfChange {
			case "New":
				fmt.Println("Adm: Får inn New peerID. Det er: ", liftChange.IDofChangedLift)
				if len(aliveLifts) <= 1 {
					outgoingBackupChan <- BackUp{"IWasAlone", ID, orders, properties}
				} else {
					outgoingBackupChan <- BackUp{"IWasNotAlone", ID, orders, properties}
				}
				aliveLifts = append(aliveLifts, liftChange.IDofChangedLift)
				sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] })

			case "Lost":
				fmt.Println("Adm: Får inn Lost peer. Det er: ", liftChange.IDofChangedLift)
				for i, lostPeer := range aliveLifts {
					if lostPeer == liftChange.IDofChangedLift {
						aliveLifts = append(aliveLifts[:i], aliveLifts[i+1:]...)
						if lostPeer != ID {
							DeassignOrders(orders, lostPeer)
						}
						if GetState(properties, ID) == IDLE {
							fmt.Println("Adm: State == IDLE, en annen heis er død => kan være nye ordre")
							findNewOrder(orders, ID, properties, aliveLifts, startTimerChan, liftInstructionChan, outgoingMessageChan)
						}
						break
					}
				}
				if len(aliveLifts) == 0 { // You are alone, you are the one who lost your connection. Turn off the outer lights.
					for floor := 0; floor < N_FLOORS; floor++ {
						liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_UP, floor, OFF}
						liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_DOWN, floor, OFF}
					}
				}

			}

		}
	}
}

func findNewOrder(orders [][]int, ID int, properties []int, aliveLifts []int, startTimerChan chan<- string,
	liftInstructionChan chan<- Instruction, outgoingMessageChan chan<- Message) {
	fmt.Println("Adm: Inne i findNewOrder. Orders, properties: ", orders, properties)

	newDirection, destination := CalculateNextOrder(orders, ID, properties, aliveLifts)

	fmt.Println("Adm: Got new direction", newDirection, destination)
	if newDirection == DIRN_STOP {
		fmt.Println("Adm: I DIRN_STOP for findNewOrder")
		AssignOrders(orders, destination, ID)
		liftInstructionChan <- Instruction{"Set floor indicator light", NOT_VALID, GetLastFloor(properties, ID), ON}
		liftInstructionChan <- Instruction{"Open the door", DIRN_STOP, NOT_VALID, NOT_VALID}
		startTimerChan <- "Opening the door now"
		outgoingMessageChan <- Message{"Stopped", ID, GetLastFloor(properties, ID), NOT_VALID}
	} else if newDirection == DIRN_DOWN || newDirection == DIRN_UP {
		fmt.Println("Adm: I DIRN_DOWN/DIRN_UP for findNewOrder")
		AssignOrders(orders, destination, ID)
		outgoingMessageChan <- Message{"NewOrder", ID, destination, NOT_VALID}
	} else {
		fmt.Println("Adm: I IDLE for findNewOrder")
		outgoingMessageChan <- Message{"Idle", ID, destination, NOT_VALID}
	}
	fmt.Println("Adm: På vei ut av findNewOrder. Orders, properties: ", orders, properties)
}

func isButtonAlreadyRegistrered(orders [][]int, b Button, ID int) bool {
	if b.ButtonType == BUTTON_COMMAND {
		if orders[b.ButtonType+ID][b.Floor] == -1 {
			return false
		}
	} else {
		if orders[b.ButtonType][b.Floor] == -1 {
			return false
		}
	}
	return true
}

func backupsAreIdentical(backup1 BackUp, backup2 BackUp) bool {
	for i := range backup1.Orders {
		for j := range backup1.Orders[i] {
			if backup1.Orders[i][j] != backup2.Orders[i][j] {
				return false
			}
		}
	}
	for k := range backup1.Properties {
		if backup1.Properties[k] != backup2.Properties[k] {
			return false
		}
	}
	return true
}
