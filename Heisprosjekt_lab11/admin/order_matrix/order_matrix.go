package order_matrix

import (
	//"fmt"
	. "../../definitions"
)

func InitializeOrders() [][]int {

	orders := make([][]int, 2+MAX_N_LIFTS)
	for i := 0; i < 2+MAX_N_LIFTS; i++ {
		orders[i] = make([]int, N_FLOORS)
		for j := 0; j < N_FLOORS; j++ {
			orders[i][j] = -1
		}
	}
	return orders // The 4 number is N_FLOORS from elev.h, can't import. Find a way to do it to improve.
}

func AddOrder(orders [][]int, floor, lift, typeOfButton int) {
	switch typeOfButton {
	case BUTTON_CALL_UP:
		if orders[BUTTON_CALL_UP][floor] == -1 {
			orders[BUTTON_CALL_UP][floor] = 0 // Index [0][]
		}
	case BUTTON_CALL_DOWN:
		if orders[BUTTON_CALL_DOWN][floor] == -1 {
			orders[BUTTON_CALL_DOWN][floor] = 0 // Index [1][]
		}
	case BUTTON_COMMAND:
		if orders[BUTTON_COMMAND+lift][floor] == -1 {
			orders[BUTTON_COMMAND+lift][floor] = 0 // Index[2+lift][]
		}
	}
}

/* Tror ikke denne er i bruk
//Bør endres, tror noe er feil (button_call)
func Delete_order(orders [][]int, floor, lift, button_call int) {
	switch button_call {
	case BUTTON_CALL_UP:
		orders[BUTTON_CALL_UP][floor] = -1 // Index [0][]
	case BUTTON_CALL_DOWN:
		orders[BUTTON_CALL_DOWN][floor] = -1 // Index [1][]
	case BUTTON_COMMAND:
		orders[BUTTON_COMMAND+lift][floor] = -1 // Index[2+lift][]
	}
}
*/

func AssignOrders(orders [][]int, floor, lift int) {
	// NB! BØR LEGGE TIL RETURVERDI SOM INDIKERER OM VI FIKK ASSIGNA
	// 15.03: Trengs det virkelig? Går videre med det samme uansett.
	if orders[BUTTON_CALL_UP][floor] == 0 {
		orders[BUTTON_CALL_UP][floor] = lift + 1 // Index [0][]
	}
	if orders[BUTTON_CALL_DOWN][floor] == 0 {
		orders[BUTTON_CALL_DOWN][floor] = lift + 1 // Index [1][]
	}
	if orders[BUTTON_COMMAND+lift][floor] == 0 {
		orders[BUTTON_COMMAND+lift][floor] = lift + 1 // Index[2+lift][]
	}
}

func DeassignOuterOrders(orders [][]int, lift int) { // Hvis mister nett -> noen andre skal ta over.
	for floor := 0; floor < N_FLOORS; floor++ { //SETT INN N_FLOORS
		if orders[BUTTON_CALL_UP][floor] == lift+1 {
			orders[BUTTON_CALL_UP][floor] = 0
		}
		if orders[BUTTON_CALL_DOWN][floor] == lift+1 {
			orders[BUTTON_CALL_DOWN][floor] = 0
		}
		if orders[BUTTON_COMMAND+lift][floor] == lift+1 {
			orders[BUTTON_COMMAND+lift][floor] = 0
		}
	}
}

func CompleteOrder(orders [][]int, floor, liftID int) {
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

/* Old version
func CompleteOrder(orders [][]int, floor, lift int) { // Akkurat nå har med alle som er utenfor
	orders[BUTTON_CALL_UP][floor] = -1      // Index [0][]
	orders[BUTTON_CALL_DOWN][floor] = -1    // Index [1][]
	orders[BUTTON_COMMAND+lift][floor] = -1 // Index[2+lift][]
}
*/
func CopyInnerOrders(target [][]int, targetLift int, source [][]int, sourceLift int) {
	for floor := 0; floor < N_FLOORS; floor++ {
		target[BUTTON_COMMAND+targetLift][floor] = source[BUTTON_COMMAND+sourceLift][floor]
	}
}

func OverwriteEverythingButInternalOrders(orders [][]int, liftID int, backupOrders [][]int) {
	for floor := 0; floor < N_FLOORS; floor++ {
		orders[BUTTON_CALL_UP][floor] = backupOrders[BUTTON_CALL_UP][floor]
		orders[BUTTON_CALL_DOWN][floor] = backupOrders[BUTTON_CALL_DOWN][floor]
		for elev := 0; elev < MAX_N_LIFTS; elev++ {
			if elev != liftID {
				orders[BUTTON_COMMAND+liftID][floor] = backupOrders[BUTTON_COMMAND+elev][floor]
			} else { // Checks own inner orders in both own table in received backup. Taking any order that exists.
				if orders[BUTTON_COMMAND+liftID][floor] == liftID+1 || backupOrders[BUTTON_COMMAND+elev][floor] == liftID+1 {
					orders[BUTTON_COMMAND+liftID][floor] = liftID + 1
				} else if orders[BUTTON_COMMAND+liftID][floor] == 0 || backupOrders[BUTTON_COMMAND+elev][floor] == 0 {
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

// Trengs en funksjon som tar vekk én assigned order? Trur det! Reavaluate orders hver gang når etasje/knappetrykk.
