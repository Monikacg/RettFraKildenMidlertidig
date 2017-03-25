package calculate_order

import (
	//"fmt"

	. "../../definitions"
	. "../lift_properties"
)

func CalculateNextOrder(orders [][]int, ID int, properties []int, aliveLifts []int) (int, int) {
	var newDirn, dest int = NOT_VALID, NOT_VALID
	dest = findDestination(orders, ID, properties, aliveLifts) //get destination?
	newDirn = GetNewDirection(dest, GetLastFloor(properties, ID))
	return newDirn, dest
}

func GetNewDirection(dest int, currentFloor int) int {
	if dest == NOT_VALID {
		return NOT_VALID
	}
	if dest-currentFloor > 0 {
		return DIRN_UP
	} else if dest-currentFloor < 0 {
		return DIRN_DOWN
	} else {
		return DIRN_STOP
	}
}

func findDestination(orders [][]int, ID int, properties []int, aliveLifts []int) int {
	//fmt.Println("CalcO: findDest")
	dest, destExists := checkForValidDestination(orders, ID)
	//^does not care about where you are. Needs only 1 order in the system (alt, 1 floor). If not, will go from 0 to 3?
	if destExists {
		return dest
	}
	return newDestination(orders, ID, properties, aliveLifts)

}

func checkForValidDestination(orders [][]int, ID int) (int, bool) {
	//fmt.Println("CalcO: checkIfValidDest")
	for floor := 0; floor < N_FLOORS; floor++ {
		if orders[BUTTON_CALL_UP][floor] == ID+1 {
			return floor, true
		}
		if orders[BUTTON_CALL_DOWN][floor] == ID+1 {
			return floor, true
		}
		if orders[BUTTON_COMMAND+ID][floor] == ID+1 {
			return floor, true
		}
	}
	return NOT_VALID, false
}

func orderAtCurrentFloor(orders [][]int, properties []int, aliveLifts []int, ID int) bool {
	var liftsIdleOrWithOpenDoors []int
	var liftsAtThisFloor []int
	floor := GetLastFloor(properties, ID)

	// If there are anyone inside you should let them out no matter what.
	if orders[BUTTON_COMMAND+ID][floor] == 0 {
		return true
	}

	for _, lift := range aliveLifts {
		if GetState(properties, lift) == IDLE || GetState(properties, lift) == DOOR_OPEN {
			_, destExists := checkForValidDestination(orders, lift)
			if !destExists {
				liftsIdleOrWithOpenDoors = append(liftsIdleOrWithOpenDoors, lift)
			}
		}
	}

	if orders[BUTTON_CALL_UP][floor] == 0 || orders[BUTTON_CALL_DOWN][floor] == 0 {
		for _, lift := range liftsIdleOrWithOpenDoors {
			if GetLastFloor(properties, lift) == floor {
				// Someone inside this lift. This lift takes all the orders.
				if orders[BUTTON_COMMAND+lift][floor] == 0 {
					return false
				}
				liftsAtThisFloor = append(liftsAtThisFloor, lift)
			}
		}
	} else {
		return false
	}

	if len(liftsAtThisFloor) == 1 { // Only you at floor.
		return true
	} else if liftsAtThisFloor[0] == ID { // Using lowest ID to determine precedence
		return true
	}

	return false
}

func newDestination(orders [][]int, ID int, properties []int, aliveLifts []int) int {
	//fmt.Println("CalcO: newDest")
	var newDest int = NOT_VALID
	var newDestExists, iAmClosest bool = false, false
	// Sjekk om skal av i samme etasje.
	switch GetState(properties, ID) { //Needs to know which elevators are alive
	case DOOR_OPEN:

		switch GetDirn(properties, ID) {
		case DIRN_UP:
			//fmt.Println("CalcO: newDest/DIRN_UP")
			/*
				if orderCurrentFloorRightDirection(orders, properties, ID) {
					return GetLastFloor(properties, ID)
				}
			*/
			if orderAtCurrentFloor(orders, properties, aliveLifts, ID) {
				return GetLastFloor(properties, ID)
			}

			newDest, newDestExists = orderAbove(orders, properties, ID)
			if newDestExists {
				return newDest
			}
			/*
				if orderCurrentFloorOppositeDirection(orders, properties, ID) {
					return GetLastFloor(properties, ID)
				}
			*/

			// None over, changing direction
			newDest, newDestExists = orderBelow(orders, properties, ID)
			if newDestExists {
				return newDest
			}
		case DIRN_DOWN:
			//fmt.Println("CalcO: newDest/DIRN_DOWN")
			/*
				if orderCurrentFloorRightDirection(orders, properties, ID) {
					return GetLastFloor(properties, ID)
				}
			*/
			if orderAtCurrentFloor(orders, properties, aliveLifts, ID) {
				return GetLastFloor(properties, ID)
			}

			newDest, newDestExists = orderBelow(orders, properties, ID)
			if newDestExists {
				return newDest
			}
			/*
				if orderCurrentFloorOppositeDirection(orders, properties, ID) {
					return GetLastFloor(properties, ID)
				}
			*/
			// None over, changing direction
			newDest, newDestExists = orderAbove(orders, properties, ID)
			if newDestExists {
				return newDest
			}
		}

	case IDLE:
		// NB! Dette kan føre til at flere tar samme. Bør endres?
		/*
			if orderCurrentFloorAny(orders, properties, ID) {
				return GetLastFloor(properties, ID)
			}
		*/
		if orderAtCurrentFloor(orders, properties, aliveLifts, ID) {
			return GetLastFloor(properties, ID)
		}

		newDest, iAmClosest = newAmIClosest(orders, properties, aliveLifts, ID)
		//Sjekker hvilke andre (som er i live) som er IDLE,
		//finner ut hvem som er nærmest. Lavest ID prioritet etter lavest avstand
		//Hvis en nærmere, tar nest nærmest frem til ingen igjen.
		// Counterpoint: Calles bare når 1 som finnes, så... Nei. Kan være en som mister connection
		// som gjør at du må ta en runde lengre unna.

		if iAmClosest {
			return newDest
		}
	}
	return NOT_VALID
}

func newAmIClosest(orders [][]int, properties []int, aliveLifts []int, ID int) (int, bool) {
	var closestLift, shortestDistance int = NOT_VALID, N_FLOORS + 2
	var closestLiftIndex int
	var floorsWithOutsideOrders []int
	var floorsWithOrdersThatHasntBeenTaken []int
	var liftsIdleOrWithOpenDoors []int
	var aliveAndMoving []int

	for _, lift := range aliveLifts {
		if GetState(properties, lift) == IDLE || GetState(properties, lift) == DOOR_OPEN {
			_, destExists := checkForValidDestination(orders, lift)
			if !destExists {
				liftsIdleOrWithOpenDoors = append(liftsIdleOrWithOpenDoors, lift)
			}
		}
	}

	// Some of these orders might already have been taken in a previous function,
	// meaning all floors that have a lift in liftsIdleOrWithOpenDoors at that floor means that
	// someone already has taken the order at that floor.
	for floor := 0; floor < N_FLOORS; floor++ {
		floorAdded := false
		if orders[BUTTON_CALL_UP][floor] == 0 {
			if !(orders[BUTTON_CALL_DOWN][floor] > 0) { // In case someone else has button down assigned.
				floorsWithOutsideOrders = append(floorsWithOutsideOrders, floor)
				floorAdded = true
			}
		}
		if orders[BUTTON_CALL_DOWN][floor] == 0 && !floorAdded {
			floorsWithOutsideOrders = append(floorsWithOutsideOrders, floor)
		}
	}

	for _, floor := range floorsWithOutsideOrders {
		anyLiftAtThisFloor := false
		for _, lift := range liftsIdleOrWithOpenDoors {
			if GetLastFloor(properties, lift) == floor {
				anyLiftAtThisFloor = true
			}
		}
		if !anyLiftAtThisFloor {
			floorsWithOrdersThatHasntBeenTaken = append(floorsWithOrdersThatHasntBeenTaken, floor)
		}
	}

	// If someone has an assigned order at this floor (would be part of aliveLifts, but not liftsIdleOrWithOpenDoors),
	// then that lift should take this order, and whoever called this function should stay away.

	for _, lift := range aliveLifts {
		_, destExists := checkForValidDestination(orders, lift)
		if !destExists {
			aliveAndMoving = append(aliveAndMoving, lift)
		}
	}

	for i, floor := range floorsWithOrdersThatHasntBeenTaken {
		for _, lift := range aliveAndMoving {
			if orders[BUTTON_COMMAND+lift][floor] > 0 {
				floorsWithOrdersThatHasntBeenTaken = append(floorsWithOrdersThatHasntBeenTaken[:i], floorsWithOrdersThatHasntBeenTaken[i+1:]...)
				i--
			}
		}
	}

	// If there are any left now, they will go to the closest lifts.
	for _, floor := range floorsWithOrdersThatHasntBeenTaken {
		closestLift, shortestDistance = NOT_VALID, N_FLOORS+2
		for j, lift := range liftsIdleOrWithOpenDoors {
			if abs(GetLastFloor(properties, lift)-floor) < shortestDistance {
				shortestDistance = abs(GetLastFloor(properties, lift) - floor)
				closestLift = lift
				closestLiftIndex = j
			}
		}
		if closestLift == ID {
			return floor, true
		}
		liftsIdleOrWithOpenDoors = append(liftsIdleOrWithOpenDoors[:closestLiftIndex], liftsIdleOrWithOpenDoors[closestLiftIndex+1:]...)
	}

	return NOT_VALID, false
}

/*
Sjekke ytre knapper for ordre, indre i alle IDLE
Vil bare vær 1 knapp trykket

MÅ TESTES GRUNDIG
*/

// Gå igjennom igjen. Ikke alltid bare én order som finnes, kan også være ytre ordre en
// lost heis har gitt opp som kan tas.
/*
func amIClosestToNewOrder(orders [][]int, properties []int, aliveLifts []int, ID int) (int, bool) {
	//fmt.Println("CalcO: amIClosest")
	var closestLift, newDest, shortestDistance int = NOT_VALID, NOT_VALID, N_FLOORS + 1
	var lastFloors []int
	var aliveIdleLifts []int
	var floorsWithOutsideOrders []int

	for floor := 0; floor < N_FLOORS; floor++ {
		floorAddedBefore := false
		if orders[BUTTON_CALL_UP][floor] == 0 {
			floorsWithOutsideOrders = append(floorsWithOutsideOrders, floor)
			//newDest = floor // Needs to go
			floorAddedBefore = true
		}
		if orders[BUTTON_CALL_DOWN][floor] == 0 && !floorAddedBefore {
			floorsWithOutsideOrders = append(floorsWithOutsideOrders, floor)
			//newDest = floor
		}
	}

	//Place all Idle lifts in a slice and iterate over them instead of aliveLifts
	for _, lift := range aliveLifts {
		if GetState(properties, lift) == IDLE {
			aliveIdleLifts = append(aliveIdleLifts, lift)
		}
	}

	for i, lift := range aliveIdleLifts {
		lastFloors = append(lastFloors, GetLastFloor(properties, lift))
		for floor := 0; floor < N_FLOORS; floor++ {
			if orders[BUTTON_COMMAND+lift][floor] == 0 { // newDest må bort
				if lift == ID {
					return floor, true
				}
				aliveIdleLifts = append(aliveIdleLifts[:i], aliveIdleLifts[i+1:]...)
				for j, f := range floorsWithOutsideOrders {
					if f == floor {
						floorsWithOutsideOrders = append(floorsWithOutsideOrders[:j], floorsWithOutsideOrders[j+1:]...)
						break
					}
				}
			}
		}
	}

	for (len(floorsWithOutsideOrders) > 0) && (len(aliveIdleLifts) > 0) {
		newDest, floorsWithOutsideOrders = floorsWithOutsideOrders[len(floorsWithOutsideOrders)-1], floorsWithOutsideOrders[:len(floorsWithOutsideOrders)-1]

		// Gives priority to lowest ID. REQUIRES SAME ORDER aliveLifts IN ALL
		// (SORT FROM LOWEST TO HIGHEST?)
		for _, lift := range aliveIdleLifts {
			if abs(GetLastFloor(properties, lift)-newDest) < shortestDistance {
				shortestDistance = abs(GetLastFloor(properties, lift) - newDest)
				closestLift = lift
			}
		}
		if closestLift == ID {
			return newDest, true
		}
		for i, lift := range aliveIdleLifts {
			if lift == closestLift {
				aliveIdleLifts = append(aliveIdleLifts[:i], aliveIdleLifts[i+1:]...)
				break
			}
		}
	}

	return NOT_VALID, false
}
*/

func abs(value int) int {
	if value < 0 {
		return value * (-1)
	}
	return value
}

// NB! Nå gir den prioritet til de som går ned i høyere etasje over å
// gå ned og hente ny. Endre hvis FAT krever annet.
func orderAbove(orders [][]int, properties []int, ID int) (int, bool) {
	//fmt.Println("CalcO: orderAbove")
	floor_start := GetLastFloor(properties, ID) + 1
	if floor_start >= N_FLOORS {
		return NOT_VALID, false
	}

	for floor := floor_start; floor < N_FLOORS; floor++ {
		if orders[BUTTON_COMMAND+ID][floor] == 0 {
			return floor, true
		}
		if orders[BUTTON_CALL_UP][floor] == 0 {
			return floor, true
		}
	}
	for floor := floor_start; floor < N_FLOORS; floor++ {
		if orders[BUTTON_CALL_DOWN][floor] == 0 {
			return floor, true
		}
	}
	return NOT_VALID, false
}

func orderBelow(orders [][]int, properties []int, ID int) (int, bool) {
	//fmt.Println("CalcO: orderBelow")
	floor_start := GetLastFloor(properties, ID) - 1
	if floor_start < 0 {
		return NOT_VALID, false
	}
	for floor := floor_start; floor >= 0; floor-- {
		//fmt.Println("CalcO: orderBelow/floorloop: floor_start, floor: ", floor_start, floor)
		if orders[BUTTON_COMMAND+ID][floor] == 0 {
			return floor, true
		}
		if orders[BUTTON_CALL_DOWN][floor] == 0 {
			return floor, true
		}
	}
	for floor := floor_start; floor >= 0; floor-- {
		if orders[BUTTON_CALL_UP][floor] == 0 {
			return floor, true
		}
	}
	return NOT_VALID, false
}

/*
//Endre navn sikkert
func orderCurrentFloorRightDirection(orders [][]int, properties []int, ID int) bool {
	//fmt.Println("CalcO: orderCurrentFloorMoving")
	floor := GetLastFloor(properties, ID)

	switch GetDirn(properties, ID) {
	case DIRN_UP:
		if orders[BUTTON_COMMAND+ID][floor] == 0 {
			return true
		}
		if orders[BUTTON_CALL_UP][floor] == 0 {
			return true
		}
	case DIRN_DOWN:
		if orders[BUTTON_COMMAND+ID][floor] == 0 {
			return true
		}
		if orders[BUTTON_CALL_DOWN][floor] == 0 {
			return true
		}
	}
	return false
}

func orderCurrentFloorOppositeDirection(orders [][]int, properties []int, ID int) bool {
	//fmt.Println("CalcO: orderCurrentFloorWrongWay")
	floor := GetLastFloor(properties, ID)

	switch GetDirn(properties, ID) {
	case DIRN_UP:
		if orders[BUTTON_CALL_DOWN][floor] == 0 {
			return true
		}

	case DIRN_DOWN:
		if orders[BUTTON_CALL_UP][floor] == 0 {
			return true
		}
	}
	return false
}

func orderCurrentFloorAny(orders [][]int, properties []int, ID int) bool {
	//fmt.Println("CalcO: orderCurrentFloorIdle")
	floor := GetLastFloor(properties, ID)

	if orders[BUTTON_COMMAND+ID][floor] == 0 {
		return true
	}
	if orders[BUTTON_CALL_UP][floor] == 0 {
		return true
	}
	if orders[BUTTON_CALL_DOWN][floor] == 0 {
		return true
	}
	return false
}

*/

func ShouldStop(orders [][]int, properties []int, floor int, ID int) bool {
	//fmt.Println("CalcO: Inne i ShouldStop, floor", floor)

	//Test at fungerer
	if floor == 0 && GetDirn(properties, ID) == DIRN_DOWN {
		return true
	}
	if floor == (N_FLOORS-1) && GetDirn(properties, ID) == DIRN_UP {
		return true
	}
	if assignedOrderExists(orders, floor, ID) {
		return true
	}
	if unassignedOrderExists(orders, properties, floor, ID) { // En vi skal stoppe på. Feasible unassigned?
		return true
	}
	//fmt.Println("CalcO: Ender med false i ShouldStop, floor", floor)
	return false
}

func assignedOrderExists(orders [][]int, floor int, ID int) bool {
	if orders[BUTTON_CALL_UP][floor] == ID+1 {
		return true
	}
	if orders[BUTTON_CALL_DOWN][floor] == ID+1 {
		return true
	}
	if orders[BUTTON_COMMAND+ID][floor] == ID+1 {
		return true
	}
	return false
}

// Vurder om bør bruk listenotasjon istedenfor å ta inn fra lift_properties
func unassignedOrderExists(orders [][]int, properties []int, floor int, ID int) bool {
	switch GetDirn(properties, ID) {
	case DIRN_UP:
		if orders[BUTTON_CALL_UP][floor] == 0 {
			return true
		}
		if orders[BUTTON_COMMAND+ID][floor] == 0 {
			return true
		}
		/*
			if floor == N_FLOORS { // Trengs den her egentlig? Vil du egentlig kjøre til 4., komme inn i funksjonen her
				if orders[BUTTON_CALL_DOWN][floor] == 0 { // og likevel komme ned hit? Står inntil videre
					return true
				}
			}
		*/
	case DIRN_DOWN:
		if orders[BUTTON_CALL_DOWN][floor] == 0 {
			return true
		}
		if orders[BUTTON_COMMAND+ID][floor] == 0 {
			return true
		}
		/*
			if floor == 0 { // Se over
				if orders[BUTTON_CALL_UP][floor] == 0 {
					return true
				}
			}
		*/
	}
	return false
}
