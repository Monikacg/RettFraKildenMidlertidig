package lift_properties

import (
	"fmt"
	"testing"

	. "./../../definitions"
)

func TestFn(t *testing.T) {
	fmt.Println("Create lift properties list: ")
	properties := InitializeLiftProperties()
	fmt.Println(properties)

	dirn := DIRN_STOP
	state := DOOR_OPEN
	lastFloor := 1

	fmt.Println("Set last floor = 1 for all lifts: ")
	for lift := 0; lift < MAX_N_LIFTS; lift++ {
		SetLastFloor(properties, lift, lastFloor)
	}
	fmt.Println(properties)

	fmt.Println("Set dirn = DIRN_STOP (0) for all lifts: ")
	for lift := 0; lift < MAX_N_LIFTS; lift++ {
		SetDirn(properties, lift, dirn)
	}
	fmt.Println(properties)

	fmt.Println("Set state = DOOR_OPEN (2) for all lifts: ")
	for lift := 0; lift < MAX_N_LIFTS; lift++ {
		SetState(properties, lift, state)
	}
	fmt.Println(properties)

}
