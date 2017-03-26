package calculate_order

import (

	//"fmt"

	. "../../definitions"
	. "../properties"
)


/*
This module is an undermodule of admin. Three functions are used directly by admin:
- CalculateNextOrder
- GetNewDirection
- ShouldStop
*/


func CalculateNextOrder(orders [][]int, ID int, properties []int, aliveLifts []int) (int, int) {
	var newDirn, destination int = NOT_VALID, NOT_VALID
	destination = getDestination(orders, ID, properties, aliveLifts)
	newDirn = GetNewDirection(destination, GetLastFloor(properties, ID))
	return newDirn, destination
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

func getDestination(orders [][]int, ID int, properties []int, aliveLifts []int) int {
	//fmt.Println("CalcO: findDest")
	destination, destinationExists := checkForValidDestination(orders, ID)
	//^does not care about where you are. Needs only 1 order in the system (alt, 1 floor). If not, will go from 0 to 3?
	if destinationExists {
		return destination
	}
	return findNewDestination(orders, ID, properties, aliveLifts)

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
			_, destinationExists := checkForValidDestination(orders, lift)
			if !destinationExists {
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

func findNewDestination(orders [][]int, ID int, properties []int, aliveLifts []int) int {
	var newDestination int = NOT_VALID
	var newDestinationExists, iAmClosest bool = false, false
	switch GetState(properties, ID) {
	case DOOR_OPEN:

		switch GetDirection(properties, ID) {
		case DIRN_UP:

			if orderAtCurrentFloor(orders, properties, aliveLifts, ID) {
				return GetLastFloor(properties, ID)
			}

			newDestination, newDestinationExists = orderAbove(orders, properties, ID)
			if newDestinationExists {
				return newDestination
			}

			// None in prefered direction, changing direction
			newDestination, newDestinationExists = orderBelow(orders, properties, ID)
			if newDestinationExists {
				return newDestination
			}
		case DIRN_DOWN:

			if orderAtCurrentFloor(orders, properties, aliveLifts, ID) {
				return GetLastFloor(properties, ID)
			}

			newDestination, newDestinationExists = orderBelow(orders, properties, ID)
			if newDestinationExists {
				return newDestination
			}

			// None in prefered direction, changing direction
			newDestination, newDestinationExists = orderAbove(orders, properties, ID)
			if newDestinationExists {
				return newDestination
			}
		}

	case IDLE:

		if orderAtCurrentFloor(orders, properties, aliveLifts, ID) {
			return GetLastFloor(properties, ID)
		}

		newDestination, iAmClosest = amIClosestToOrder(orders, properties, aliveLifts, ID)


		if iAmClosest {
			return newDestination
		}
	}
	return NOT_VALID
}


func amIClosestToOrder(orders [][]int, properties []int, aliveLifts []int, ID int) (int, bool) {
	var closestLift, shortestDistance int = NOT_VALID, N_FLOORS + 2
	var closestLiftIndex int
	var floorsWithOutsideOrders []int
	var floorsWithOrdersThatHasntBeenTaken []int
	var liftsIdleOrWithOpenDoors []int
	var liftsIdle []int
	var aliveAndMoving []int

	for _, lift := range aliveLifts {
		if GetState(properties, lift) == IDLE || GetState(properties, lift) == DOOR_OPEN {
			_, destinationExists := checkForValidDestination(orders, lift)
			if !destinationExists {
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

	// Floors where there are lifts are removed since we know the outer orders there are taken care of.
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

	// Slices in go requires some way to keep track of the index you are deleting if you are deleting pieces of the one you are iterating over.
	liftDelCount := 0
	for floor := 0; floor < N_FLOORS; floor++ {
		for i, lift := range liftsIdleOrWithOpenDoors {
			if orders[BUTTON_COMMAND+lift][floor] == 0 {
				if lift == ID {
					return floor, true
				}
				for j, f := range floorsWithOrdersThatHasntBeenTaken {
					if f == floor {
						floorsWithOrdersThatHasntBeenTaken = append(floorsWithOrdersThatHasntBeenTaken[:j], floorsWithOrdersThatHasntBeenTaken[j+1:]...)
						break
					}
				}
				liftsIdleOrWithOpenDoors = append(liftsIdleOrWithOpenDoors[:i], liftsIdleOrWithOpenDoors[i+1:]...)
				i--
				liftDelCount++

				if len(liftsIdleOrWithOpenDoors) == i+liftDelCount-1 {
					break
				}
			}
		}
	}

	// If someone has an assigned order at this floor (would be part of aliveLifts, but not liftsIdleOrWithOpenDoors),
	// then that lift should take this order, and whoever called this function should stay away.

	for _, lift := range aliveLifts {
		_, destinationExists := checkForValidDestination(orders, lift)
		if !destinationExists {
			aliveAndMoving = append(aliveAndMoving, lift)
		}
	}

	floorDelCount := 0
	for i, floor := range floorsWithOrdersThatHasntBeenTaken {
		for _, lift := range aliveAndMoving {
			if orders[BUTTON_COMMAND+lift][floor] > 0 {
				floorsWithOrdersThatHasntBeenTaken = append(floorsWithOrdersThatHasntBeenTaken[:i], floorsWithOrdersThatHasntBeenTaken[i+1:]...)
				i--
			}
		}
		if len(floorsWithOrdersThatHasntBeenTaken) == i+floorDelCount-1 {
			break
		}
	}

	//fmt.Println("CO: FLoorsthathasn't been taken: ", floorsWithOrdersThatHasntBeenTaken)

	for _, lift := range liftsIdleOrWithOpenDoors {
		if GetState(properties, lift) == IDLE {
			liftsIdle = append(liftsIdle, lift)
		}
	}

	// If there are any left now, they will go to the closest lifts in IDLE state (so no order gets stuck waiting for a lift with an open door).
	for _, floor := range floorsWithOrdersThatHasntBeenTaken {
		closestLift, shortestDistance = NOT_VALID, N_FLOORS+2
		for j, lift := range liftsIdle {
			if abs(GetLastFloor(properties, lift)-floor) < shortestDistance {
				shortestDistance = abs(GetLastFloor(properties, lift) - floor)
				closestLift = lift
				closestLiftIndex = j
			}
		}
		if closestLift == ID {
			return floor, true
		}
		liftsIdle = append(liftsIdle[:closestLiftIndex], liftsIdle[closestLiftIndex+1:]...)
	}

	return NOT_VALID, false
}



func abs(value int) int {
	if value < 0 {
		return value * (-1)
	}
	return value
}


func orderAbove(orders [][]int, properties []int, liftID int) (int, bool) {
	floor_start := GetLastFloor(properties, liftID) + 1
	if floor_start >= N_FLOORS {
		return NOT_VALID, false
	}

	for floor := floor_start; floor < N_FLOORS; floor++ {
		if orders[BUTTON_COMMAND+liftID][floor] == 0 {
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

func orderBelow(orders [][]int, properties []int, liftID int) (int, bool) {
	floor_start := GetLastFloor(properties, liftID) - 1
	if floor_start < 0 {
		return NOT_VALID, false
	}
	for floor := floor_start; floor >= 0; floor-- {
		if orders[BUTTON_COMMAND+liftID][floor] == 0 {
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


func ShouldStop(orders [][]int, properties []int, floor int, liftID int) bool {

	if floor == 0 && GetDirection(properties, liftID) == DIRN_DOWN {
		return true
	}
	if floor == (N_FLOORS-1) && GetDirection(properties, liftID) == DIRN_UP {
		return true
	}
	if assignedOrderExists(orders, floor, liftID) {
		return true
	}
	if unassignedOrderExists(orders, properties, floor, liftID) {
		return true
	}
	return false
}

func assignedOrderExists(orders [][]int, floor int, liftID int) bool {
	if orders[BUTTON_CALL_UP][floor] == liftID+1 {
		return true
	}
	if orders[BUTTON_CALL_DOWN][floor] == liftID+1 {
		return true
	}
	if orders[BUTTON_COMMAND+liftID][floor] == liftID+1 {
		return true
	}
	return false
}

func unassignedOrderExists(orders [][]int, properties []int, floor int, liftID int) bool {
	switch GetDirection(properties, liftID) {
	case DIRN_UP:
		if orders[BUTTON_CALL_UP][floor] == 0 {
			return true
		}
		if orders[BUTTON_COMMAND+liftID][floor] == 0 {
			return true
		}

	case DIRN_DOWN:
		if orders[BUTTON_CALL_DOWN][floor] == 0 {
			return true
		}
		if orders[BUTTON_COMMAND+liftID][floor] == 0 {
			return true
		}

	}
	return false
}
