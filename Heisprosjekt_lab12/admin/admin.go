package admin

import (
	"fmt"
	"time"

	"sort"

	. "../definitions"
	. "./calculate_order"
	. "./lift_properties"
	. "./order_matrix"
)

func Admin(IDInput int, buttonChan <-chan Button, floorSensorChan <-chan int,
	localOrderChan chan<- Order, adminTChan chan<- Udp, adminRChan <-chan Udp, backupTChan chan<- BackUp, backupRChan <-chan BackUp,
	peerChangeChan <-chan Peer, peerInitializeChan <-chan CurrPeers, startTimerChan chan<- string, timeOutChan <-chan string) {

	var stuckTimer *time.Timer
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
				backupTChan <- BackUp{"IWasAlone", ID, orders, properties}
				aliveLifts = append(aliveLifts, peerMsg.ChangedPeer)
				sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] }) //Bare problem på mac?

			case "Lost":
				for i, lostPeer := range aliveLifts {
					if lostPeer == peerMsg.ChangedPeer {
						aliveLifts = append(aliveLifts[:i], aliveLifts[i+1:]...)
						if lostPeer != ID {
							DeassignOuterOrders(orders, lostPeer)
						}
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

		case f := <-floorSensorChan:
			fmt.Println("Adm: initLoop, floor Sensor") // NOTE: NEEDS DIRN_DOWN SET FURTHER UP.
			SetLastFloor(properties, ID, f)
			localOrderChan <- Order{"FLOOR_LIGHT", NOT_VALID, f, ON}
			localOrderChan <- Order{"DIRN", DIRN_STOP, NOT_VALID, ON}
			startTimerChan <- "Opening the door now"
			adminTChan <- Udp{ID, "Stopped", f, NOT_VALID}
			break initLoop

		default:
			SetState(properties, ID, MOVING)
			localOrderChan <- Order{"DIRN", DIRN_DOWN, NOT_VALID, NOT_VALID}
			break initLoop
		}
	}

	for {
		select {

		case b := <-buttonChan:

			if !isButtonAlreadyRegistrered(orders, b, ID) {
				adminTChan <- Udp{ID, "ButtonPressed", b.Floor, b.Button_dir}
			}

		case fs := <-floorSensorChan:
			switch GetState(properties, ID) {
			case MOVING:
				if fs != GetLastFloor(properties, ID) {
					SetLastFloor(properties, ID, fs)
					localOrderChan <- Order{"FLOOR_LIGHT", NOT_VALID, fs, ON}
					//fmt.Println("Adm: Verdier på vei inn i Should_stop: (orders, properties, fs, ID)")
					//fmt.Println("Adm: ", orders, properties, fs, ID)
					if ShouldStop(orders, properties, fs, ID) == true {
						//fmt.Println("Adm: Should_stop")
						localOrderChan <- Order{"DIRN", DIRN_STOP, NOT_VALID, ON}
						startTimerChan <- "Opening the door now"
						adminTChan <- Udp{ID, "Stopped", fs, NOT_VALID}
					} else {
						//fmt.Println("Adm: Should_stop NOT")
						adminTChan <- Udp{ID, "DrovePast", fs, NOT_VALID} // ID, "kjørte forbi", etasje
						//fmt.Println("Adm: Under teit beskjed")
					}
				}
			}

		case timeOut := <-timeOutChan:
			fmt.Println("Adm: Fikk timeout")
			switch timeOut {
			case "Time to close the door":
				localOrderChan <- Order{"DOOR", NOT_VALID, NOT_VALID, OFF}

				//TURN OFF LIGHTS!
				localOrderChan <- Order{"LIGHT", BUTTON_COMMAND, GetLastFloor(properties, ID), OFF}
				localOrderChan <- Order{"LIGHT", BUTTON_CALL_UP, GetLastFloor(properties, ID), OFF}
				localOrderChan <- Order{"LIGHT", BUTTON_CALL_DOWN, GetLastFloor(properties, ID), OFF}
				//Alternativt etter hver CompleteOrder, men der blir ikke helt bra med når ordre i samme etasje.
				findNewOrder(orders, ID, properties, aliveLifts, startTimerChan, localOrderChan, adminTChan)

			case "Time to exit STUCK state and see if the engine is working":
				select {
				case f := <-floorSensorChan:
					fmt.Println("Adm: Stuck exited, floor Sensor")
					SetLastFloor(properties, ID, f)
					localOrderChan <- Order{"FLOOR_LIGHT", NOT_VALID, f, ON}
					localOrderChan <- Order{"DIRN", DIRN_STOP, NOT_VALID, ON}
					startTimerChan <- "Opening the door now"
					adminTChan <- Udp{ID, "Stopped", f, NOT_VALID}

				default:
					SetState(properties, ID, MOVING)
					SetDirn(properties, ID, DIRN_DOWN)
					localOrderChan <- Order{"DIRN", DIRN_DOWN, NOT_VALID, NOT_VALID}
					stuckTimer = time.NewTimer(stuckTimeout)
				}
			}

		case m := <-adminRChan:
			fmt.Println("Adm: Fått inn melding fra adminRChan, melding: ", m)
			switch m.ID {
			case ID:
				//Alt for egen heis
				switch m.Type {
				case "ButtonPressed":
					fmt.Println("Adm: Får tilbake fra network, ButtonPressed")
					AddOrder(orders, m.Floor, m.ID, m.ExtraInfo)
					localOrderChan <- Order{"LIGHT", m.ExtraInfo, m.Floor, ON}
					fmt.Println("Adm: Samme loop, state og orders: ", GetState(properties, ID), orders)
					if GetState(properties, ID) == IDLE {
						fmt.Println("Adm: State == IDLE når knapp trykket på")
						findNewOrder(orders, ID, properties, aliveLifts, startTimerChan, localOrderChan, adminTChan)
					}
					fmt.Println("Adm: Properties inne i samme case: ", properties)
				case "Stopped":
					SetLastFloor(properties, ID, m.Floor)
					SetState(properties, ID, DOOR_OPEN)
					AssignOrders(orders, m.Floor, ID) // også nederst nå.
					CompleteOrder(orders, m.Floor, ID)
					fmt.Println("Adm: Orders at ", m.Floor, " when I get stopped back: ", orders)
					fmt.Println("Adm: Fått Stopped tilbake. Properties: ", properties)
					if AnyAssignedOrdersLeft(orders, m.ID) {
						stuckTimer = time.NewTimer(stuckTimeout)
					} else {
						stuckTimer.Stop()
					}

				case "DrovePast":
					SetLastFloor(properties, m.ID, m.Floor)
					SetState(properties, m.ID, MOVING)
					fmt.Println("Adm: DrovePast kommer rundt, setter lastFloor/state=MOVING. Properties: ", properties)
				case "NewOrder":
					// Gjør alt før, er bare ack her. Skal det i det hele tatt komme tilbake hit?
					SetState(properties, ID, MOVING)
					SetDirn(properties, ID, GetNewDirection(m.Floor, GetLastFloor(properties, ID)))
					localOrderChan <- Order{"DIRN", GetNewDirection(m.Floor, GetLastFloor(properties, ID)), NOT_VALID, NOT_VALID}
					// AssignOrders(orders, m.Floor, ID) MOVED
					fmt.Println("Adm: Orders at floor ", m.Floor, " now belongs to me. Orders now: ", orders)
					fmt.Println("Adm: NewOrder kommer rundt. Properties: ", properties)
					stuckTimer = time.NewTimer(stuckTimeout)

				case "I'm stuck":
					SetState(properties, m.ID, STUCK)
					DeassignOuterOrders(orders, m.ID)
					startTimerChan <- "Entered STUCK state"

				case "Idle":
					// Samme som over. Nada.
					SetState(properties, m.ID, IDLE)
					fmt.Println("Adm: Idle kommer rundt, setter state=IDLE. Orders, properties: ", orders, properties)
				}

			default: //Any other lift
				switch m.Type {
				case "ButtonPressed":
					fmt.Println("Adm: Får tilbake fra network, annen heis, ButtonPressed")
					AddOrder(orders, m.Floor, m.ID, m.ExtraInfo)
					if m.ExtraInfo == BUTTON_CALL_UP || m.ExtraInfo == BUTTON_CALL_DOWN {
						localOrderChan <- Order{"LIGHT", m.ExtraInfo, m.Floor, ON}
						fmt.Println("Adm: Samme loop, state og orders: ", GetState(properties, ID), orders)
						if GetState(properties, ID) == IDLE {
							fmt.Println("Adm: State == IDLE når knapp trykket på, melding fra annen heis")
							findNewOrder(orders, ID, properties, aliveLifts, startTimerChan, localOrderChan, adminTChan)
						}
					}
					fmt.Println("Adm: Properties inne i samme case: ", properties)
				case "Stopped":
					fmt.Println("Adm: Får tilbake fra network, annen heis, Stopped")
					SetLastFloor(properties, m.ID, m.Floor)
					SetState(properties, m.ID, DOOR_OPEN)
					AssignOrders(orders, m.Floor, m.ID)
					CompleteOrder(orders, m.Floor, m.ID)
					localOrderChan <- Order{"LIGHT", BUTTON_CALL_UP, m.Floor, OFF}
					localOrderChan <- Order{"LIGHT", BUTTON_CALL_DOWN, m.Floor, OFF}
					fmt.Println("Adm: The ID of the lift that stopped, orders, properties: ", m.ID, orders, properties)
				case "DrovePast":
					fmt.Println("Adm: Får tilbake fra network, annen heis, DrovePast")
					SetLastFloor(properties, m.ID, m.Floor)
					SetState(properties, m.ID, MOVING)
					fmt.Println("Adm: Properties inne i samme case: ", properties)

				case "NewOrder":
					fmt.Println("Adm: Får tilbake fra network, annen heis, NewOrder")
					SetState(properties, m.ID, MOVING)
					// Skal vi sette retning hvis retning er DIRN_STOP?
					SetDirn(properties, m.ID, GetNewDirection(m.Floor, GetLastFloor(properties, m.ID)))
					AssignOrders(orders, m.Floor, m.ID)
					fmt.Println("Adm: Orders at floor ", m.Floor, " now belongs to ", m.ID, " . Orders now: ", orders)
					fmt.Println("Adm: Properties inne i samme case: ", properties)

				case "I'm stuck":
					DeassignOuterOrders(orders, m.ID)
					SetState(properties, m.ID, STUCK)
					if GetState(properties, ID) == IDLE {
						fmt.Println("Adm: State == IDLE når en annen er STUCK ")
						findNewOrder(orders, ID, properties, aliveLifts, startTimerChan, localOrderChan, adminTChan)
					}

				case "Idle":
					fmt.Println("Adm: Får tilbake fra network, annen heis, Idle")
					SetState(properties, m.ID, IDLE)
					fmt.Println("Adm: Orders, properties inne i samme case: ", orders, properties)
				}
			}

		case <-stuckTimer.C:
			adminTChan <- Udp{ID, "I'm stuck", GetLastFloor(properties, ID), NOT_VALID}

		case backupMsg := <-backupRChan:
			fmt.Println("Adm: Fått inn melding fra backupRChan, melding: ", backupMsg)
			if ID != backupMsg.SenderID {
				if !backupsAreIdentical(backupMsg, lastBackUpRecevied[backupMsg.SenderID]) {
					lastBackUpRecevied[backupMsg.SenderID] = backupMsg
					switch backupMsg.Info {
					case "IWasAlone":
						fmt.Println("Adm: Fått ny backup (I was alone). Backupmelding: ", backupMsg)
						fmt.Println("Adm: Orders before backupcommands: ", orders)
						// Legg inn alle INDRE ordre for backupMsg.SenderID
						CopyInnerOrders(orders, ID, backupMsg.Orders, backupMsg.SenderID)
						// Ta inn properties for backupMsg.SenderID
						SetSingleLiftProperties(properties, backupMsg.SenderID, backupMsg.Properties)

						fmt.Println("Adm: Orders after backupcommands: ", orders)

					case "IWasNotAlone":
						fmt.Println("Adm: Fått ny backup (I was NOT alone). Backupmelding: ", backupMsg)
						fmt.Println("Adm: Orders before backupcommands: ", orders)
						// Skriv over alt i orders minus egne indre ordre.
						OverwriteEverythingButInternalOrders(orders, ID, backupMsg.Orders)

						// Behold egne properties, skriv over resten.
						SetPropertiesFromBackup(properties, ID, backupMsg.Properties)
						fmt.Println("Adm: Orders after backupcommands: ", orders)

					}
				}
			}

		case peerMsg := <-peerChangeChan:
			switch peerMsg.Change {
			case "New":
				fmt.Println("Adm: Får inn New peerID. Det er: ", peerMsg.ChangedPeer)
				if len(aliveLifts) <= 1 {
					backupTChan <- BackUp{"IWasAlone", ID, orders, properties}
				} else {
					backupTChan <- BackUp{"IWasNotAlone", ID, orders, properties}
				}
				aliveLifts = append(aliveLifts, peerMsg.ChangedPeer)
				sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] })

			case "Lost":
				fmt.Println("Adm: Får inn Lost peer. Det er: ", peerMsg.ChangedPeer)
				for i, lostPeer := range aliveLifts {
					if lostPeer == peerMsg.ChangedPeer {
						aliveLifts = append(aliveLifts[:i], aliveLifts[i+1:]...)
						if lostPeer != ID {
							DeassignOuterOrders(orders, lostPeer)
						}
						if GetState(properties, ID) == IDLE {
							fmt.Println("Adm: State == IDLE, en annen heis er død => kan være nye ordre")
							findNewOrder(orders, ID, properties, aliveLifts, startTimerChan, localOrderChan, adminTChan)
						}
						break
					}
				}
			}

		}
	}
}

func findNewOrder(orders [][]int, ID int, properties []int, aliveLifts []int, startTimerChan chan<- string,
	localOrderChan chan<- Order, adminTChan chan<- Udp) {
	fmt.Println("Adm: Inne i findNewOrder. Orders, properties: ", orders, properties)

	newDirn, dest := CalculateNextOrder(orders, ID, properties, aliveLifts)

	// Default dest and newDirn returned has to be undefined (-2,-2)
	fmt.Println("Adm: Got new direction", newDirn, dest)
	if newDirn == DIRN_STOP {
		fmt.Println("Adm: I DIRN_STOP for findNewOrder")
		AssignOrders(orders, dest, ID)
		localOrderChan <- Order{"FLOOR_LIGHT", NOT_VALID, GetLastFloor(properties, ID), ON}
		localOrderChan <- Order{"DIRN", DIRN_STOP, NOT_VALID, ON}
		startTimerChan <- "Opening the door now"
		adminTChan <- Udp{ID, "Stopped", GetLastFloor(properties, ID), NOT_VALID}
	} else if newDirn == DIRN_DOWN || newDirn == DIRN_UP {
		fmt.Println("Adm: I DIRN_DOWN/DIRN_UP for findNewOrder")
		AssignOrders(orders, dest, ID)
		adminTChan <- Udp{ID, "NewOrder", dest, NOT_VALID} // ID, "Moving, desting (new order)", etasje
	} else { // newDirn == -2 (NOT_VALID)
		fmt.Println("Adm: I IDLE for findNewOrder")
		adminTChan <- Udp{ID, "Idle", dest, NOT_VALID} // ID, "IDLE", etasje
	}
	fmt.Println("Adm: På vei ut av findNewOrder. Orders, properties: ", orders, properties)
}

func isButtonAlreadyRegistrered(orders [][]int, b Button, ID int) bool {
	if b.Button_dir == BUTTON_COMMAND {
		if orders[b.Button_dir+ID][b.Floor] == -1 {
			return false
		}
	} else {
		if orders[b.Button_dir][b.Floor] == -1 {
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
