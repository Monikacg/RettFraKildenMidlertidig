package calculate_order

import (
	"fmt"
	"testing"

	. "../orders"
	. "../properties"
	. "./../../definitions"
)

func TestFn(t *testing.T) {
	orders := InitializeOrders()
	properties := InitializeLiftProperties()
	var aliveLifts []int
	var newDirn, newDest int
	//aliveLifts := make([]int, MAX_N_LIFTS)
	ID := 0
	ID1 := 1
	ID2 := 2

	aliveLifts = append(aliveLifts, ID, ID1, ID2)

	/* Test av ShouldStop complete - It works as it should.


	AddOrder(orders, 1, ID, BUTTON_CALL_UP)
	AssignOrders(orders, 1, ID)
	SetLastFloor(properties, ID, 1)
	SetDirection(properties, ID, DIRN_DOWN)
	SetState(properties, ID, MOVING)
	fmt.Println("Properties: ", properties)

	if ShouldStop(orders, properties, 1, ID) {
		fmt.Println("Should really stop")
	}
	*/

	/*
		Possible newDest
		DIRN_DOWN = -1
		DIRN_STOP = 0
		DIRN_UP   = 1

		Floors normal
	*/

	/*
		// The rest of the file: Test of CalculateNextOrder.
		// First: Assigned orders.
		SetLastFloor(properties, ID, 1)
		SetDirection(properties, ID, DIRN_UP)
		SetState(properties, ID, IDLE)
		fmt.Println("Properties: ", properties)

		AddOrder(orders, 1, ID, BUTTON_CALL_UP)
		AssignOrders(orders, 1, ID)
		newDirn, newDest = CalculateNextOrder(orders, ID, properties, aliveLifts)
		fmt.Println("Alone: At floor 1/order at same floor, should be 0 1 (stop, 1)")
		fmt.Println(newDirn, newDest)
		orders = InitializeOrders()

		AddOrder(orders, 2, ID, BUTTON_CALL_UP)
		AssignOrders(orders, 2, ID)
		newDirn, newDest = CalculateNextOrder(orders, ID, properties, aliveLifts)
		fmt.Println("Alone: At floor 1/order at floor 2, should be 1 2 (up, 2)")
		fmt.Println(newDirn, newDest)
		orders = InitializeOrders()

		AddOrder(orders, 0, ID, BUTTON_CALL_UP)
		AssignOrders(orders, 0, ID)
		newDirn, newDest = CalculateNextOrder(orders, ID, properties, aliveLifts)
		fmt.Println("Alone: At floor 1/order at floor 0, should be -1 0 (down, 0)")
		fmt.Println(newDirn, newDest)
		orders = InitializeOrders()

		//Konklusjon: Fungerer bra for assigned orders, alle typer knapper, alle states.
	*/

	/*
		// Tester to i IDLE i første etasje (floor 0), en idle i floor 1. CalculateNextOrder for alle 3
		SetLastFloor(properties, ID, 1)
		SetDirection(properties, ID, DIRN_DOWN)
		SetState(properties, ID, IDLE)

		SetLastFloor(properties, ID1, 0)
		SetDirection(properties, ID1, DIRN_DOWN)
		SetState(properties, ID1, IDLE)

		SetLastFloor(properties, ID2, 0)
		SetDirection(properties, ID2, DIRN_DOWN)
		SetState(properties, ID2, IDLE)
		fmt.Println("Properties: ", properties)

		AddOrder(orders, 2, ID, BUTTON_CALL_UP)
		newDirn, newDest = CalculateNextOrder(orders, ID, properties, aliveLifts)
		fmt.Println("Three: This elevator is closest, should be 1 2 (up, 2)")
		fmt.Println(newDirn, newDest)
		newDirn, newDest = CalculateNextOrder(orders, ID1, properties, aliveLifts)
		fmt.Println("Three: This elevator is one of the ones further away, should be -2 -2 (NOT_VALID, NOT_VALID)")
		fmt.Println(newDirn, newDest)
		newDirn, newDest = CalculateNextOrder(orders, ID2, properties, aliveLifts)
		fmt.Println("Three: This elevator is one of the ones further away, should be -2 -2 (NOT_VALID, NOT_VALID)")
		fmt.Println(newDirn, newDest)

		// Perfect

	*/

	// Samme eksempel som over, men elevator som står i etasjen nærmest er nå MOVING på vei ned. Nå skal en av de
	// lengst ned ta ordren, men ikke den andre (skal være den med lavest ID, men vet ikke om det er implementert.)
	orders = InitializeOrders()
	SetLastFloor(properties, ID, 1)
	SetDirection(properties, ID, DIRN_DOWN)
	SetState(properties, ID, MOVING)

	SetLastFloor(properties, ID1, 0)
	SetDirection(properties, ID1, DIRN_DOWN)
	SetState(properties, ID1, IDLE)

	SetLastFloor(properties, ID2, 0)
	SetDirection(properties, ID2, DIRN_DOWN)
	SetState(properties, ID2, IDLE)
	fmt.Println("Properties: ", properties)

	AddOrder(orders, 2, ID, BUTTON_CALL_UP)
	newDirn, newDest = CalculateNextOrder(orders, ID, properties, aliveLifts)
	fmt.Println("Three: This elevator is closest, but moving down. Should not call this function, so no idea. It ends up with -2, -2, so that's great.")
	fmt.Println(newDirn, newDest)
	newDirn, newDest = CalculateNextOrder(orders, ID1, properties, aliveLifts)
	fmt.Println("Three: This elevator should take the order, should be 1 2 (up, 2)")
	fmt.Println(newDirn, newDest)
	newDirn, newDest = CalculateNextOrder(orders, ID2, properties, aliveLifts)
	fmt.Println("Three: This elevator is one of the ones further away, should be -2 -2 (NOT_VALID, NOT_VALID)")
	fmt.Println(newDirn, newDest)

	// Perfect

}
