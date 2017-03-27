package orders

import (
	. "../../definitions"
)

// Orders is a submodule of admin.  It contains functions to manipulate the orders table (adding orders, assigning++)
// The orders table is built up, from the left, by one column for outer orders going up,
// one for outer orders going down, then one column for each lifts inner orders (MAX_N_LIFTS).
// Each column is the same length as the number of floors (N_FLOORS).

// -1 indicates no order
// 0 indicates an unassigned order
// Numbers higher than 0 indicates an assigned order. An order is assigned to one specific lift by
// their ID+1. For example, Lift with ID = 2 has an assigned order if the element in the orders table is equal to 3 (2+1).
// This requires all lifts to have unique IDs, and it requires the ID to be less than MAX_N_LIFTS.

func InitializeOrders() [][]int {

	orders := make([][]int, 2+MAX_N_LIFTS)
	for liftID := 0; liftID < 2+MAX_N_LIFTS; liftID++ {
		orders[liftID] = make([]int, N_FLOORS)
		for floor := 0; floor < N_FLOORS; floor++ {
			orders[liftID][floor] = -1
		}
	}
	return orders
}

func AddOrder(orders [][]int, floor, liftID, typeOfButton int) {
	switch typeOfButton {
	case BUTTON_CALL_UP:
		if orders[BUTTON_CALL_UP][floor] == -1 {
			orders[BUTTON_CALL_UP][floor] = 0
		}
	case BUTTON_CALL_DOWN:
		if orders[BUTTON_CALL_DOWN][floor] == -1 {
			orders[BUTTON_CALL_DOWN][floor] = 0
		}
	case BUTTON_COMMAND:
		if orders[BUTTON_COMMAND+liftID][floor] == -1 {
			orders[BUTTON_COMMAND+liftID][floor] = 0
		}
	}
}

func AssignOrders(orders [][]int, floor, liftID int) {
	if orders[BUTTON_CALL_UP][floor] == 0 {
		orders[BUTTON_CALL_UP][floor] = liftID + 1
	}
	if orders[BUTTON_CALL_DOWN][floor] == 0 {
		orders[BUTTON_CALL_DOWN][floor] = liftID + 1
	}
	if orders[BUTTON_COMMAND+liftID][floor] == 0 {
		orders[BUTTON_COMMAND+liftID][floor] = liftID + 1
	}
}

func DeassignOrders(orders [][]int, liftID int) {
	for floor := 0; floor < N_FLOORS; floor++ {
		if orders[BUTTON_CALL_UP][floor] == liftID+1 {
			orders[BUTTON_CALL_UP][floor] = 0
		}
		if orders[BUTTON_CALL_DOWN][floor] == liftID+1 {
			orders[BUTTON_CALL_DOWN][floor] = 0
		}
		if orders[BUTTON_COMMAND+liftID][floor] == liftID+1 {
			orders[BUTTON_COMMAND+liftID][floor] = 0
		}
	}
}

func CompleteOrders(orders [][]int, floor, liftID int) {
	if orders[BUTTON_CALL_UP][floor] == liftID+1 {
		orders[BUTTON_CALL_UP][floor] = -1
	}
	if orders[BUTTON_CALL_DOWN][floor] == liftID+1 {
		orders[BUTTON_CALL_DOWN][floor] = -1
	}
	if orders[BUTTON_COMMAND+liftID][floor] == liftID+1 {
		orders[BUTTON_COMMAND+liftID][floor] = -1
	}
}

func ExtractInnerOrders(target [][]int, source [][]int) {
	for floor := 0; floor < N_FLOORS; floor++ {
		for elev := 0; elev < MAX_N_LIFTS; elev++ {
			if target[BUTTON_COMMAND+elev][floor] == elev+1 || source[BUTTON_COMMAND+elev][floor] == elev+1 {
				target[BUTTON_COMMAND+elev][floor] = elev + 1
			} else if target[BUTTON_COMMAND+elev][floor] == 0 || source[BUTTON_COMMAND+elev][floor] == 0 {
				target[BUTTON_COMMAND+elev][floor] = 0
			} else {
				target[BUTTON_COMMAND+elev][floor] = -1
			}
		}
	}
}

func CopyOrdersFromBackup(orders [][]int, backupOrders [][]int) {
	for floor := 0; floor < N_FLOORS; floor++ {
		orders[BUTTON_CALL_UP][floor] = backupOrders[BUTTON_CALL_UP][floor]
		orders[BUTTON_CALL_DOWN][floor] = backupOrders[BUTTON_CALL_DOWN][floor]
		for elev := 0; elev < MAX_N_LIFTS; elev++ {
			if orders[BUTTON_COMMAND+elev][floor] == elev+1 || backupOrders[BUTTON_COMMAND+elev][floor] == elev+1 {
				orders[BUTTON_COMMAND+elev][floor] = elev + 1
			} else if orders[BUTTON_COMMAND+elev][floor] == 0 || backupOrders[BUTTON_COMMAND+elev][floor] == 0 {
				orders[BUTTON_COMMAND+elev][floor] = 0
			} else {
				orders[BUTTON_COMMAND+elev][floor] = -1
			}
		}
	}
}

func AnyAssignedOrdersLeft(orders [][]int, liftID int) bool {
	for floor := 0; floor < N_FLOORS; floor++ {
		if orders[BUTTON_CALL_UP][floor] == liftID+1 {
			return true
		}
		if orders[BUTTON_CALL_DOWN][floor] == liftID+1 {
			return true
		}
		if orders[BUTTON_COMMAND+liftID][floor] == liftID+1 {
			return true
		}
	}
	return false
}
