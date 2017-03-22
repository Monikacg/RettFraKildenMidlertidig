package orders

import (
	. "../../definitions"
)

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

func DeassignOuterOrders(orders [][]int, liftID int) {
	for floor := 0; floor < N_FLOORS; floor++ {
		if orders[BUTTON_CALL_UP][floor] == liftID+1 {
			orders[BUTTON_CALL_UP][floor] = 0
		}
		if orders[BUTTON_CALL_DOWN][floor] == liftID+1 {
			orders[BUTTON_CALL_DOWN][floor] = 0
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

func OverwriteInnerOrders(target [][]int, targetLiftID int, source [][]int, sourceLiftID int) {
	for floor := 0; floor < N_FLOORS; floor++ {
		target[BUTTON_COMMAND+targetLiftID][floor] = source[BUTTON_COMMAND+sourceLiftID][floor]
	}
}

func OverwriteEverythingButInternalOrders(orders [][]int, liftID int, backupOrders [][]int) {
	for floor := 0; floor < N_FLOORS; floor++ {
		orders[BUTTON_CALL_UP][floor] = backupOrders[BUTTON_CALL_UP][floor]
		orders[BUTTON_CALL_DOWN][floor] = backupOrders[BUTTON_CALL_DOWN][floor]
		for elev := 0; elev < MAX_N_LIFTS; elev++ {
			if elev != liftID {
				orders[BUTTON_COMMAND+liftID][floor] = backupOrders[BUTTON_CALL_DOWN][floor]
			} else { // Checks own inner orders in both own table in received backup. Taking any order that exists.
				if orders[BUTTON_COMMAND+liftID][floor] == liftID+1 || backupOrders[BUTTON_CALL_DOWN][floor] == liftID+1 {
					orders[BUTTON_COMMAND+liftID][floor] = liftID + 1
				} else if orders[BUTTON_COMMAND+liftID][floor] == 0 || backupOrders[BUTTON_CALL_DOWN][floor] == 0 {
					orders[BUTTON_COMMAND+liftID][floor] = 0
				} else {
					orders[BUTTON_COMMAND+liftID][floor] = -1
				}
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
