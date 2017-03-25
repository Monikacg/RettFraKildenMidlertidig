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

/* Kan kanskje trengs hvis ikke sort.Slice lightens up. Fungerte på sanntidssal, så kanskje greit?
^Bare problem med versjon: sort.Slice kommer bare inn i go 1.8.
type ActiveLift struct {
	LiftID int
}
*/

func Admin(IDInput int, buttonChan <-chan Button, floorSensorChan <-chan int,
	localOrderChan chan<- Order, adminTChan chan<- Udp, adminRChan <-chan Udp, backupTChan chan<- BackUp, backupRChan <-chan BackUp,
	peerChangeChan <-chan Peer, peerInitializeChan <-chan CurrPeers, startTimerChan chan<- string, timeOutChan <-chan string) {

	orders := InitializeOrders()
	properties := InitializeLiftProperties()
	ID := IDInput
	aliveLifts := make([]int, 0, MAX_N_LIFTS) // Kan tydeligvis bare skrive var aliveLifts []int og la golang fikse det selv...
	//aliveLifts = append(aliveLifts, ID)
	//For test
	//aliveLifts = append(aliveLifts, 1)

	//lastButtonPressed := Button{NOT_VALID, NOT_VALID} //Går uten dette
	//bi := 0

	//Spør nett om noen har orders og properties. If so, sett orders og properties lik de på nettet. If not, forsett med det samme.

	// Tror det er uavhengig av hvilken state det står i for det her: Vil bare vite om vi står i en etasje.

	//Kan bruke PEERS for å få inn ALLE

	SetDirn(properties, ID, DIRN_DOWN)

getAliveLiftsLoop:
	for {
		select {
		case totalPeers := <-peerInitializeChan: // Sett inn under også? for å være trygg?
			aliveLifts = totalPeers.Peers
			break getAliveLiftsLoop
		case <-time.After(2 * time.Second): // To be sure we continue
			break getAliveLiftsLoop
		}
	}

	inList := false
	fmt.Println("Adm: aliveLifts etter første loop: ", aliveLifts)
	for _, peer := range aliveLifts {
		if peer == ID {
			inList = true
		}
	}

	if !inList {
		aliveLifts = append(aliveLifts, ID)
		sort.Slice(aliveLifts, func(i, j int) bool { return aliveLifts[i] < aliveLifts[j] })
	}
	fmt.Println("Adm: aliveLifts etter test om egen id finnes der: ", aliveLifts)

searchingForBackupLoop:
	for {
		select {
		//case totalPeers := <-peerInitializeChan: // Sett inn under også? for å være trygg?
		//aliveLifts = totalPeers.Peers
		//Legg inn en måte å få inn andre som er på nett?
		case backup := <-backupRChan:
			orders = backup.Orders
			properties = backup.Properties
			break searchingForBackupLoop

		case peerMsg := <-peerChangeChan:
			switch peerMsg.Change {
			case "New": //Må sjekke om peer allerede er i aliveLifts
				fmt.Println("Adm: Får inn New peer. Det er: ", peerMsg.ChangedPeer)
				if len(aliveLifts) - 1 == 1 {
					backupTChan <- BackUp{"VarAlene", ID, orders, properties}
				} else {
					backupTChan <- BackUp{"IkkeAlene", ID, orders, properties}
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
					backupTChan <- BackUp{"VarAlene", ID, orders, properties}
				} else {
					backupTChan <- BackUp{"IkkeAlene", ID, orders, properties}
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
			case "IWasAlone":
				// Legg inn alle INDRE ordre for backupMsg.SenderID
				CopyInnerOrders(orders, ID, backupMsg.Orders, backupMsg.SenderID)
				// Ta inn properties for backupMsg.SenderID
				SetSingleLiftProperties(properties, backupMsg.SenderID, backupMsg.Properties)

			case "IWasNotAlone":
				// Skriv over alt i orders minus egne indre ordre.
				OverwriteEverythingButInternalOrders(orders, ID, backupMsg.Orders)

				// Behold egne properties, skriv over resten.
				SetPropertiesFromBackup(properties, ID, backupMsg.Properties)

			}

		case f := <-floorSensorChan:
			fmt.Println("Adm: initLoop, floor Sensor") // NOTE: NEEDS DIRN_DOWN SET FURTHER UP.
			SetLastFloor(properties, ID, f)
			localOrderChan <- Order{"FLOOR_LIGHT", NOT_VALID, f, ON}
			localOrderChan <- Order{"DIRN", DIRN_STOP, NOT_VALID, ON}
			startTimerChan <- "DOOR_OPEN"
			adminTChan <- Udp{ID, "Stopped", f, NOT_VALID}
			break initLoop

		case <-time.After(3 * time.Second):
			SetState(properties, ID, MOVING)
			localOrderChan <- Order{"DIRN", DIRN_DOWN, NOT_VALID, NOT_VALID}
			break initLoop
		}
	}
	// Exit init state.

	for {
		select {
		// Problem med å sende melding om button pressed ut på nettet og deretter melding fra findNewOrder?
		// evt legge ved hvilke ordre vi tar hver gang i findNewOrder-melding => alle andre kan oppdatere.
		// Husk "problem" med at assign bare tar de som allerede finnes, så
		// må ha en måte å slå sammen her.

		case b := <-buttonChan:

			adminTChan <- Udp{ID, "ButtonPressed", b.Floor, b.Button_dir}
			/*if b != lastButtonPressed {
				adminTChan <- Udp{ID, "ButtonPressed", b.Floor, b.Button_dir}
				bi++

				if bi >= 5 {
					lastButtonPressed = Button{NOT_VALID, NOT_VALID}
				}
			}*/

			//Tanke: Legg inn noe som gjør at det ikke legges til(sendes ut på NW) hvis allerede finnes i orders.
			/*if not in orders {
				adminTChan <- Udp{ID, "ButtonPressed", b.Floor, b.Button_dir}
			}*/

		case fs := <-floorSensorChan:
			switch GetState(properties, ID) {
			case DOOR_OPEN:
				//Intentionally blank, probably might as well just remove this case, right now for completeness
				// Just needs to break, which it will do without these. Remove in the end?
			case IDLE:
				// See DOOR_OPEN
			case MOVING:
				if fs != GetLastFloor(properties, ID) {
					SetLastFloor(properties, ID, fs)
					localOrderChan <- Order{"FLOOR_LIGHT", NOT_VALID, fs, ON}
					//fmt.Println("Adm: Verdier på vei inn i Should_stop: (orders, properties, fs, ID)")
					//fmt.Println("Adm: ", orders, properties, fs, ID)
					if ShouldStop(orders, properties, fs, ID) == true {
						//fmt.Println("Adm: Should_stop")
						localOrderChan <- Order{"DIRN", DIRN_STOP, NOT_VALID, ON}
						startTimerChan <- "DOOR_OPEN"
						adminTChan <- Udp{ID, "Stopped", fs, NOT_VALID}
					} else {
						//fmt.Println("Adm: Should_stop NOT")
						adminTChan <- Udp{ID, "DrovePast", fs, NOT_VALID} // ID, "kjørte forbi", etasje
						//fmt.Println("Adm: Under teit beskjed")
					}
				}
			}

		case <-timeOutChan:
			fmt.Println("Adm: Fikk timeout")
			localOrderChan <- Order{"DOOR", NOT_VALID, NOT_VALID, OFF}

			//TURN OFF LIGHTS!
			localOrderChan <- Order{"LIGHT", BUTTON_COMMAND, GetLastFloor(properties, ID), OFF}
			localOrderChan <- Order{"LIGHT", BUTTON_CALL_UP, GetLastFloor(properties, ID), OFF}
			localOrderChan <- Order{"LIGHT", BUTTON_CALL_DOWN, GetLastFloor(properties, ID), OFF}
			//Alternativt etter hver CompleteOrder, men der blir ikke helt bra med når ordre i samme etasje.

			findNewOrder(orders, ID, properties, aliveLifts, startTimerChan, localOrderChan, adminTChan)

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
					AssignOrders(orders, m.Floor, ID)
					CompleteOrder(orders, m.Floor, ID)
					fmt.Println("Adm: Orders at ", m.Floor, " when I get stopped back: ", orders)
					fmt.Println("Adm: Fått Stopped tilbake. Properties: ", properties)

				case "DrovePast":
					SetState(properties, m.ID, MOVING)
					SetLastFloor(properties, m.ID, m.Floor)
					fmt.Println("Adm: DrovePast kommer rundt, setter lastFloor/state=MOVING. Properties: ", properties)
				case "NewOrder":
					// Gjør alt før, er bare ack her. Skal det i det hele tatt komme tilbake hit?
					localOrderChan <- Order{"DIRN", GetNewDirection(m.Floor, GetLastFloor(properties, ID)), NOT_VALID, NOT_VALID}
					AssignOrders(orders, m.Floor, ID)
					SetState(properties, ID, MOVING)
					SetDirn(properties, ID, GetNewDirection(m.Floor, GetLastFloor(properties, ID)))
					fmt.Println("Adm: Orders at floor ", m.Floor, " now belongs to me. Orders now: ", orders)
					fmt.Println("Adm: NewOrder kommer rundt. Properties: ", properties)
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
					AssignOrders(orders, m.Floor, m.ID)
					CompleteOrder(orders, m.Floor, m.ID)
					localOrderChan <- Order{"LIGHT", BUTTON_CALL_UP, m.Floor, OFF}
					localOrderChan <- Order{"LIGHT", BUTTON_CALL_DOWN, m.Floor, OFF}
					SetState(properties, m.ID, DOOR_OPEN)
					SetLastFloor(properties, m.ID, m.Floor)
					fmt.Println("Adm: The ID of the lift that stopped, orders, properties: ", m.ID, orders, properties)
				case "DrovePast":
					fmt.Println("Adm: Får tilbake fra network, annen heis, DrovePast")
					SetLastFloor(properties, m.ID, m.Floor)
					SetState(properties, m.ID, MOVING)
					fmt.Println("Adm: Properties inne i samme case: ", properties)

				case "NewOrder":
					fmt.Println("Adm: Får tilbake fra network, annen heis, NewOrder")
					AssignOrders(orders, m.Floor, m.ID)
					SetState(properties, m.ID, MOVING)
					SetDirn(properties, m.ID, GetNewDirection(m.Floor, GetLastFloor(properties, m.ID)))
					fmt.Println("Adm: Orders at floor ", m.Floor, " now belongs to ", m.ID ," . Orders now: ", orders)
					fmt.Println("Adm: Properties inne i samme case: ", properties)

				case "Idle":
					fmt.Println("Adm: Får tilbake fra network, annen heis, Idle")
					SetState(properties, m.ID, IDLE)
					fmt.Println("Adm: Orders, properties inne i samme case: ", orders, properties)
				}
			}

		case backupMsg := <-backupRChan:
			fmt.Println("Adm: Fått inn melding fra backupRChan, melding: ", backupMsg)
			switch backupMsg.Info {
			case "IWasAlone":
				fmt.Println("Adm: Fått ny backup (I was alone). Backup melding: ")
				fmt.Println(backupMsg)
				fmt.Println("Adm: Orders before backupcommands: ", orders)
				// Legg inn alle INDRE ordre for backupMsg.SenderID
				CopyInnerOrders(orders, ID, backupMsg.Orders, backupMsg.SenderID)
				// Ta inn properties for backupMsg.SenderID
				SetSingleLiftProperties(properties, backupMsg.SenderID, backupMsg.Properties)

				fmt.Println("Adm: Orders after backupcommands: ", orders)

			case "IWasNotAlone":
				fmt.Println("Adm: Fått ny backup (I was NOT alone). Backup melding: ")
				fmt.Println(backupMsg)
				fmt.Println("Adm: Orders before backupcommands: ", orders)
				// Skriv over alt i orders minus egne indre ordre.
				OverwriteEverythingButInternalOrders(orders, ID, backupMsg.Orders)

				// Behold egne properties, skriv over resten.
				SetPropertiesFromBackup(properties, ID, backupMsg.Properties)
				fmt.Println("Adm: Orders after backupcommands: ", orders)

			}

		case peerMsg := <-peerChangeChan:
			switch peerMsg.Change {
			case "New":
				fmt.Println("Adm: Får inn New peerID. Det er: ", peerMsg.ChangedPeer)
				if len(aliveLifts) == 1 {
					backupTChan <- BackUp{"VarAlene", ID, orders, properties}
				} else {
					backupTChan <- BackUp{"IkkeAlene", ID, orders, properties}
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
		localOrderChan <- Order{"FLOOR_LIGHT", NOT_VALID, GetLastFloor(properties, ID), ON}
		localOrderChan <- Order{"DIRN", DIRN_STOP, NOT_VALID, ON}
		startTimerChan <- "DOOR_OPEN"
		adminTChan <- Udp{ID, "Stopped", GetLastFloor(properties, ID), NOT_VALID}
	} else if newDirn == DIRN_DOWN || newDirn == DIRN_UP {
		fmt.Println("Adm: I DIRN_DOWN/DIRN_UP for findNewOrder")
		adminTChan <- Udp{ID, "NewOrder", dest, NOT_VALID} // ID, "Moving, desting (new order)", etasje
	} else { // newDirn == -2 (NOT_VALID)
		fmt.Println("Adm: I IDLE for findNewOrder")
		adminTChan <- Udp{ID, "Idle", dest, NOT_VALID} // ID, "IDLE", etasje
	}
	fmt.Println("Adm: På vei ut av findNewOrder. Orders, properties: ", orders, properties)
}
