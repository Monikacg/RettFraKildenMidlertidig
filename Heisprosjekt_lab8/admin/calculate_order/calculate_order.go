package calculate_order

import (
	. "../../definitions"
	. "../properties"
)

/*
CalculateNextOrder uses the orders table to find which destination to send the lift to next.
To see how the orders table is built up, check the comment at the top in order.go.
*/
func CalculateNextOrder(orders [][]int, properties []int, aliveLifts []int, ID int) (int, int) {
	var direction, destination int                                    //= NOT_VALID, NOT_VALID               // Not necessary?
	destination = findDestination(orders, properties, aliveLifts, ID) //get destination?
	direction = GetNewDirection(destination, GetLastFloor(properties, ID))
	return direction, destination
}

func GetNewDirection(destination int, currentFloor int) int {
	if destination == NOT_VALID {
		return NOT_VALID
	}
	if destination-currentFloor > 0 {
		return DIRN_UP
	} else if destination-currentFloor < 0 {
		return DIRN_DOWN
	} else {
		return DIRN_STOP
	}
}

func findDestination(orders [][]int, properties []int, aliveLifts []int, ID int) int {
	//fmt.Println("CalcO: findDest")
	destination, destinationExists := checkIfDestinationAlreadyExists(orders, ID)
	//^does not care about where you are. Needs only 1 order in the system (alt, 1 floor). If not, will go from 0 to 3?
	if destinationExists {
		return destination
	}
	return findNewDestination(orders, properties, aliveLifts, ID)

}

func checkIfDestinationAlreadyExists(orders [][]int, ID int) (int, bool) {
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

func findNewDestination(orders [][]int, properties []int, aliveLifts []int, ID int) int {
	//fmt.Println("CalcO: newDest")
	var newDestination int                    //= NOT_VALID
	var newDestinationExists, iAmClosest bool //= false, false
	// Sjekk om skal av i samme etasje.
	switch GetState(properties, ID) { //Needs to know which elevators are alive
	case DOOR_OPEN:

		switch GetDirection(properties, ID) {
		case DIRN_UP:
			//fmt.Println("CalcO: newDest/DIRN_UP")
			if orderCurrentFloorRightDirection(orders, properties, ID) {
				return GetLastFloor(properties, ID)
			}
			newDestination, newDestinationExists = orderAbove(orders, properties, ID)
			if newDestinationExists {
				return newDestination
			}

			if orderCurrentFloorOppositeDirection(orders, properties, ID) {
				return GetLastFloor(properties, ID)
			}

			// None over, changing direction
			newDestination, newDestinationExists = orderBelow(orders, properties, ID)
			if newDestinationExists {
				return newDestination
			}
		case DIRN_DOWN:
			//fmt.Println("CalcO: newDestination/DIRN_DOWN")
			if orderCurrentFloorRightDirection(orders, properties, ID) {
				return GetLastFloor(properties, ID)
			}
			newDestination, newDestinationExists = orderBelow(orders, properties, ID)
			if newDestinationExists {
				return newDestination
			}

			if orderCurrentFloorOppositeDirection(orders, properties, ID) {
				return GetLastFloor(properties, ID)
			}
			// None over, changing direction
			newDestination, newDestinationExists = orderAbove(orders, properties, ID)
			if newDestinationExists {
				return newDestination
			}
		}

	case IDLE:
		// NB! Dette kan føre til at flere tar samme. Bør endres? Kan det gjøres uten den første?
		//if orderCurrentFloorAny(orders, properties, ID) {
		//return GetLastFloor(properties, ID)
		//}
		newDestination, iAmClosest = amIClosestToNewOrder(orders, properties, aliveLifts, ID)

		if iAmClosest {
			return newDestination
		}
	}
	return NOT_VALID
}

// Gå igjennom igjen. Ikke alltid bare én order som finnes, kan også være ytre ordre en
// lost heis har gitt opp som kan tas.
func amIClosestToNewOrder(orders [][]int, properties []int, aliveLifts []int, ID int) (int, bool) {
	//fmt.Println("CalcO: amIClosest")
	var closestLift, newDestination, shortestDistance int = NOT_VALID, NOT_VALID, N_FLOORS + 2
	var lastFloors []int
	var aliveIdleLifts []int
	var floorsWithOutsideOrders []int

	for floor := 0; floor < N_FLOORS; floor++ {
		floorAddedBefore := false
		if orders[BUTTON_CALL_UP][floor] == 0 {
			floorsWithOutsideOrders = append(floorsWithOutsideOrders, floor)
			floorAddedBefore = true
		}
		if orders[BUTTON_CALL_DOWN][floor] == 0 && !floorAddedBefore {
			floorsWithOutsideOrders = append(floorsWithOutsideOrders, floor)
		}
	}

	//Place all Idle lifts in a slice and iterate over them instead of aliveLifts
	for _, lift := range aliveLifts {
		if GetState(properties, lift) == IDLE {
			aliveIdleLifts = append(aliveIdleLifts, lift)
		}
	}

	var indicesOfLiftsWithNewOrders []int
	for i, lift := range aliveIdleLifts {
		lastFloors = append(lastFloors, GetLastFloor(properties, lift))
		for floor := 0; floor < N_FLOORS; floor++ {
			if orders[BUTTON_COMMAND+lift][floor] == 0 {
				if lift == ID {
					return floor, true
				}
				indicesOfLiftsWithNewOrders = append(indicesOfLiftsWithNewOrders, i)
				//aliveIdleLifts = append(aliveIdleLifts[:i], aliveIdleLifts[i+1:]...)
				for j, f := range floorsWithOutsideOrders {
					if f == floor {
						// The Idle lift on position i takes the outside orders on this floor
						floorsWithOutsideOrders = append(floorsWithOutsideOrders[:j], floorsWithOutsideOrders[j+1:]...)
						break
					}
				}
			}
		}
	}

	for k, i := range indicesOfLiftsWithNewOrders {
		aliveIdleLifts = append(aliveIdleLifts[:i-k], aliveIdleLifts[i-k+1:]...)
	}

	for (len(floorsWithOutsideOrders) > 0) && (len(aliveIdleLifts) > 0) {
		// Pops the last element from floorsWithOutsideOrders to the variable newDestination
		newDestination, floorsWithOutsideOrders = floorsWithOutsideOrders[len(floorsWithOutsideOrders)-1], floorsWithOutsideOrders[:len(floorsWithOutsideOrders)-1]

		for _, lift := range aliveIdleLifts {
			if abs(GetLastFloor(properties, lift)-newDestination) < shortestDistance {
				shortestDistance = abs(GetLastFloor(properties, lift) - newDestination)
				closestLift = lift
			}
		}
		if closestLift == ID {
			return newDestination, true
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
	floorStart := GetLastFloor(properties, ID) + 1
	if floorStart >= N_FLOORS {
		return NOT_VALID, false
	}

	for floor := floorStart; floor < N_FLOORS; floor++ {
		if orders[BUTTON_COMMAND+ID][floor] == 0 {
			return floor, true
		}
		if orders[BUTTON_CALL_UP][floor] == 0 {
			return floor, true
		}
	}

	// If we don't find any orders above going up, we try to find orders above going down.
	for floor := floorStart; floor < N_FLOORS; floor++ {
		if orders[BUTTON_CALL_DOWN][floor] == 0 {
			return floor, true
		}
	}
	return NOT_VALID, false
}

func orderBelow(orders [][]int, properties []int, ID int) (int, bool) {
	//fmt.Println("CalcO: orderBelow")
	floorStart := GetLastFloor(properties, ID) - 1
	if floorStart < 0 {
		return NOT_VALID, false
	}
	for floor := floorStart; floor >= 0; floor-- {
		//fmt.Println("CalcO: orderBelow/floorloop: floorStart, floor: ", floorStart, floor)
		if orders[BUTTON_COMMAND+ID][floor] == 0 {
			return floor, true
		}
		if orders[BUTTON_CALL_DOWN][floor] == 0 {
			return floor, true
		}
	}
	// If we don't find any orders below going down, we try to find orders below going up.
	for floor := floorStart; floor >= 0; floor-- {
		if orders[BUTTON_CALL_UP][floor] == 0 {
			return floor, true
		}
	}
	return NOT_VALID, false
}

//Endre navn sikkert
func orderCurrentFloorRightDirection(orders [][]int, properties []int, ID int) bool {
	//fmt.Println("CalcO: orderCurrentFloorMoving")
	floor := GetLastFloor(properties, ID)

	switch GetDirection(properties, ID) {
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

	switch GetDirection(properties, ID) {
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

func ShouldStop(orders [][]int, properties []int, floor int, ID int) bool {
	// NOTE: Hvis noen andre har tatt en ordre i den etasjen du spør om ShouldStop i, men det finnes en åpen ordre i den etasjen likevel:
	//  -> IKKE ta den ordren, la den andre gjøken ta den når den har kommet frem.

	//fmt.Println("CalcO: Inne i ShouldStop, floor", floor)

	//Test at fungerer
	if floor == 0 && GetDirection(properties, ID) == DIRN_DOWN {
		return true
	}
	if floor == (N_FLOORS-1) && GetDirection(properties, ID) == DIRN_UP {
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

// unassignedOrderExists only returns true if there is a free order at floor AND no other lift has assigned order at floor.
func unassignedOrderExists(orders [][]int, properties []int, floor int, ID int) bool {
	switch GetDirection(properties, ID) {
	case DIRN_UP:
		if orders[BUTTON_CALL_UP][floor] == 0 {
			for lift := 0; lift < MAX_N_LIFTS; lift++ {
				if orders[BUTTON_COMMAND+lift][floor] > 0 { // This would mean some other lift is on the way
					return false
				}
			}
			return true
		}
		if orders[BUTTON_COMMAND+ID][floor] == 0 {
			for lift := 0; lift < MAX_N_LIFTS; lift++ {
				if orders[BUTTON_COMMAND+lift][floor] > 0 {
					return false
				}
			}
			return true
		}

	case DIRN_DOWN:
		if orders[BUTTON_CALL_DOWN][floor] == 0 {
			for lift := 0; lift < MAX_N_LIFTS; lift++ {
				if orders[BUTTON_COMMAND+lift][floor] > 0 {
					return false
				}
			}
			return true
		}
		if orders[BUTTON_COMMAND+ID][floor] == 0 {
			for lift := 0; lift < MAX_N_LIFTS; lift++ {
				if orders[BUTTON_COMMAND+lift][floor] > 0 {
					return false
				}
			}
			return true
		}

	}
	return false
}
