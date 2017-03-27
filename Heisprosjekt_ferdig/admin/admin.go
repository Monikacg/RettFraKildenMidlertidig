package admin

import (
	"time"
	"sort"

	. "../definitions"
	. "./calculateOrder"
	. "./properties"
	. "./orders"
)

func Admin(IDInput int, buttonPressedChan <-chan Button, floorSensorTriggeredChan <-chan int,
	liftInstructionChan chan<- Instruction, outgoingMessageChan chan<- Message, incomingMessageChan <-chan Message, outgoingBackupChan chan<- BackUp, incomingBackupChan <-chan BackUp,
	aliveLiftChangeChan <-chan ChangedLift, startTimerChan chan<- string, timeOutChan <-chan string) {

	const stuckTimeout = 10 * time.Second

	orders := InitializeOrders()
	properties := InitializeLiftProperties()
	ownID := IDInput

	var aliveLifts []int
	lastBackUpRecevied := make([]BackUp, MAX_N_LIFTS)
	for i := range lastBackUpRecevied {
		lastBackUpRecevied[i].Orders = InitializeOrders()
		lastBackUpRecevied[i].Properties = InitializeLiftProperties()
	}

	stuckTimer := time.NewTimer(stuckTimeout)

// Want the information we can get from the other lifts before we start moving.
searchingForInitialBackupLoop:
	for {
		select {

		case backup := <-incomingBackupChan:
			orders = backup.Orders
			properties = backup.Properties

			// No one else would have the right information about our properties at this point, so reset those:
			SetState(properties, ownID, INIT)
			SetDirection(properties, ownID, DIRN_DOWN)
			SetLastFloor(properties, ownID, NOT_VALID)
			break searchingForInitialBackupLoop

		case liftChange := <-aliveLiftChangeChan:
			switch liftChange.TypeOfChange {
			case "New":
				outgoingBackupChan <- BackUp{"I was isolated", ownID, orders, properties}
				aliveLifts = append(aliveLifts, liftChange.IDofChangedLift)
				sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] })

			case "Lost":
				for i, lostPeer := range aliveLifts {
					if lostPeer == liftChange.IDofChangedLift {
						aliveLifts = append(aliveLifts[:i], aliveLifts[i+1:]...)
						if lostPeer != ownID {
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
			SetLastFloor(properties, ownID, floor)
			liftInstructionChan <- Instruction{"Set floor indicator light", NOT_VALID, floor, ON}
			liftInstructionChan <- Instruction{"Open the door", DIRN_STOP, NOT_VALID, NOT_VALID}
			startTimerChan <- "Opening the door now"
			outgoingMessageChan <- Message{"Stopped at floor", ownID, floor, NOT_VALID}
			stuckTimer.Stop()
			break initLoop

		default:
			SetState(properties, ownID, MOVING)
			liftInstructionChan <- Instruction{"Set motor direction", DIRN_DOWN, NOT_VALID, NOT_VALID}
			stuckTimer = time.NewTimer(stuckTimeout)
			break initLoop
		}
	}


	for {
		select {
		case button := <-buttonPressedChan:
			if !isButtonAlreadyRegistrered(orders, button, ownID) {
				outgoingMessageChan <- Message{"Button pressed", ownID, button.Floor, button.ButtonType}
			}

		case floor := <-floorSensorTriggeredChan:
			switch GetState(properties, ownID) {
			case MOVING:
				if floor != GetLastFloor(properties, ownID) {
					SetLastFloor(properties, ownID, floor)
					liftInstructionChan <- Instruction{"Set floor indicator light", NOT_VALID, floor, ON}

					if ShouldStop(orders, properties, floor, ownID) == true {
						liftInstructionChan <- Instruction{"Open the door", DIRN_STOP, NOT_VALID, NOT_VALID}
						startTimerChan <- "Opening the door now"
						outgoingMessageChan <- Message{"Stopped at floor", ownID, floor, NOT_VALID}
					} else {
						outgoingMessageChan <- Message{"Drove past floor", ownID, floor, NOT_VALID}
					}
				}
			}

		case timeOut := <-timeOutChan:
			switch timeOut {
			case "Time to close the door":
				liftInstructionChan <- Instruction{"Close the door", NOT_VALID, NOT_VALID, NOT_VALID}

				// Turning off lights for this floor, as we complete the orders here.
				liftInstructionChan <- Instruction{"Set light in button", BUTTON_COMMAND, GetLastFloor(properties, ownID), OFF}
				liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_UP, GetLastFloor(properties, ownID), OFF}
				liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_DOWN, GetLastFloor(properties, ownID), OFF}

				findNewOrder(orders, ownID, properties, aliveLifts, startTimerChan, liftInstructionChan, outgoingMessageChan)

			case "Time to exit STUCK state and see if the engine is working":
				select {
				case floor := <-floorSensorTriggeredChan:
					SetLastFloor(properties, ownID, floor)
					liftInstructionChan <- Instruction{"Set floor indicator light", NOT_VALID, floor, ON}
					liftInstructionChan <- Instruction{"Open the door", DIRN_STOP, NOT_VALID, NOT_VALID}
					startTimerChan <- "Opening the door now"
					outgoingMessageChan <- Message{"Stopped at floor", ownID, floor, NOT_VALID}

				default:
					SetState(properties, ownID, MOVING)
					SetDirection(properties, ownID, DIRN_DOWN)
					liftInstructionChan <- Instruction{"Set motor direction", DIRN_DOWN, NOT_VALID, NOT_VALID}
					stuckTimer = time.NewTimer(stuckTimeout)
				}
			}

		case m := <-incomingMessageChan:
			switch m.ID {
			case ownID:
				switch m.Info {
				case "Button pressed":
					AddOrder(orders, m.Floor, m.ID, m.ButtonType)
					liftInstructionChan <- Instruction{"Set light in button", m.ButtonType, m.Floor, ON}

					if GetState(properties, ownID) == IDLE {
						findNewOrder(orders, ownID, properties, aliveLifts, startTimerChan, liftInstructionChan, outgoingMessageChan)
					}

				case "Stopped at floor":
					SetLastFloor(properties, m.ID, m.Floor)
					SetState(properties, m.ID, DOOR_OPEN)
					AssignOrders(orders, m.Floor, m.ID)
					CompleteOrders(orders, m.Floor, m.ID)

					if !AnyAssignedOrdersLeft(orders, ownID) {
						stuckTimer.Stop()
					} else {
						stuckTimer = time.NewTimer(stuckTimeout)
					}

				case "Drove past floor":
					SetLastFloor(properties, m.ID, m.Floor)
					SetState(properties, m.ID, MOVING)
					stuckTimer = time.NewTimer(stuckTimeout)

				case "Got assigned a new order":
					SetState(properties, m.ID, MOVING)
					SetDirection(properties, m.ID, GetNewDirection(m.Floor, GetLastFloor(properties, m.ID)))
					liftInstructionChan <- Instruction{"Set motor direction", GetNewDirection(m.Floor, GetLastFloor(properties, m.ID)), NOT_VALID, NOT_VALID}
					stuckTimer = time.NewTimer(stuckTimeout)

				case "I'm stuck":
					SetState(properties, m.ID, STUCK)
					DeassignOrders(orders, m.ID)
					startTimerChan <- "Entered STUCK state"
					liftInstructionChan <- Instruction{"Entered STUCK state, stopping engine", DIRN_STOP, NOT_VALID, NOT_VALID}

				case "Entered IDLE state":
					SetState(properties, m.ID, IDLE)
				}

			default: //Any other lift
				switch m.Info {
				case "Button pressed":
					AddOrder(orders, m.Floor, m.ID, m.ButtonType)
					if m.ButtonType == BUTTON_CALL_UP || m.ButtonType == BUTTON_CALL_DOWN {
						liftInstructionChan <- Instruction{"Set light in button", m.ButtonType, m.Floor, ON}

						if GetState(properties, ownID) == IDLE {
							findNewOrder(orders, ownID, properties, aliveLifts, startTimerChan, liftInstructionChan, outgoingMessageChan)
						}
					}

				case "Stopped at floor":
					SetLastFloor(properties, m.ID, m.Floor)
					SetState(properties, m.ID, DOOR_OPEN)
					AssignOrders(orders, m.Floor, m.ID)
					CompleteOrders(orders, m.Floor, m.ID)
					liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_UP, m.Floor, OFF}
					liftInstructionChan <- Instruction{"Set light in button", BUTTON_CALL_DOWN, m.Floor, OFF}

				case "Drove past floor":
					SetLastFloor(properties, m.ID, m.Floor)
					SetState(properties, m.ID, MOVING)

				case "Got assigned a new order":
					SetState(properties, m.ID, MOVING)
					SetDirection(properties, m.ID, GetNewDirection(m.Floor, GetLastFloor(properties, m.ID)))
					AssignOrders(orders, m.Floor, m.ID)

				case "I'm stuck":
					SetState(properties, m.ID, STUCK)
					DeassignOrders(orders, m.ID)

					if GetState(properties, ownID) == IDLE {
						findNewOrder(orders, ownID, properties, aliveLifts, startTimerChan, liftInstructionChan, outgoingMessageChan)
					}

				case "Entered IDLE state":
					SetState(properties, m.ID, IDLE)
				}
			}

		case <-stuckTimer.C:
			SetLastFloor(properties, ownID, NOT_VALID) // Sets Last floor to a value that is not equal 0 so the lift will stop no matter where it gets stuck.
			outgoingMessageChan <- Message{"I'm stuck", ownID, GetLastFloor(properties, ownID), NOT_VALID}

		case backupMsg := <-incomingBackupChan:
			if ownID != backupMsg.SenderID {
				if !backupsAreIdentical(backupMsg, lastBackUpRecevied[backupMsg.SenderID]) {
					lastBackUpRecevied[backupMsg.SenderID] = backupMsg
					switch backupMsg.Info {
					case "I was isolated":
						// ExtractInnerOrders will give orders the highest valued inner order it can find (assigned order > unassigned order > no order)
						ExtractInnerOrders(orders, backupMsg.Orders)
						SetSingleLiftProperties(properties, backupMsg.SenderID, backupMsg.Properties)

					case "I was part of a group":
						// CopyOrdersFromBackup will give orders the highest valued inner order it can find (assigned order > unassigned order > no order), and copy the outer orders.
						// Copying outer orders instead of taking the highest value since the ones in the group might have finished the order
						// while you were isolated. If you were part of the same group, you will have the same orders anyway.
						CopyOrdersFromBackup(orders, backupMsg.Orders)
						SetOtherLiftsPropertiesFromBackup(properties, ownID, backupMsg.Properties)
					}
				}
			}

		case liftChange := <-aliveLiftChangeChan:
			switch liftChange.TypeOfChange {
			case "New":
				if len(aliveLifts) <= 1 {
					outgoingBackupChan <- BackUp{"I was isolated", ownID, orders, properties}
				} else {
					outgoingBackupChan <- BackUp{"I was part of a group", ownID, orders, properties}
				}
				aliveLifts = append(aliveLifts, liftChange.IDofChangedLift)
				sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] })

			case "Lost":
				for i, lostPeer := range aliveLifts {
					if lostPeer == liftChange.IDofChangedLift {
						aliveLifts = append(aliveLifts[:i], aliveLifts[i+1:]...)
						if lostPeer != ownID {
							DeassignOrders(orders, lostPeer)
						}
						if GetState(properties, ownID) == IDLE {
							findNewOrder(orders, ownID, properties, aliveLifts, startTimerChan, liftInstructionChan, outgoingMessageChan)
						}
						break
					}
				}
				if len(aliveLifts) == 0 { // Lost connection.
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

	newDirection, destination := CalculateNextOrder(orders, ID, properties, aliveLifts)

	if newDirection == DIRN_STOP {
		AssignOrders(orders, destination, ID)
		liftInstructionChan <- Instruction{"Set floor indicator light", NOT_VALID, GetLastFloor(properties, ID), ON}
		liftInstructionChan <- Instruction{"Open the door", DIRN_STOP, NOT_VALID, NOT_VALID}
		startTimerChan <- "Opening the door now"
		outgoingMessageChan <- Message{"Stopped at floor", ID, GetLastFloor(properties, ID), NOT_VALID}

	} else if newDirection == DIRN_DOWN || newDirection == DIRN_UP {
		AssignOrders(orders, destination, ID)
		outgoingMessageChan <- Message{"Got assigned a new order", ID, destination, NOT_VALID}

	} else {
		outgoingMessageChan <- Message{"Entered IDLE state", ID, destination, NOT_VALID}
	}
}


func isButtonAlreadyRegistrered(orders [][]int, b Button, liftID int) bool {
	if b.ButtonType == BUTTON_COMMAND {
		if orders[b.ButtonType+liftID][b.Floor] == -1 {
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
