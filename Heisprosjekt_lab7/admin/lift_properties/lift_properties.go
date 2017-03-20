package lift_properties

import (
	. "../../definitions"
)

func InitializeLiftProperties() []int {
	properties := make([]int, 3*MAX_N_LIFTS)
	for i := 0; i < MAX_N_LIFTS; i++ {
		properties[3*i] = NOT_VALID   // Last floor
		properties[3*i+1] = DIRN_DOWN // Direction
		properties[3*i+2] = INIT      // State
	}
	return properties
}

func SetLastFloor(properties []int, lift, lastFloor int) {
	properties[3*lift] = lastFloor
}

func SetDirn(properties []int, lift, dirn int) {
	properties[3*lift+1] = dirn
}

func SetState(properties []int, lift, state int) {
	properties[3*lift+2] = state
}

func GetLastFloor(properties []int, lift int) int { // I calculate_order
	return properties[3*lift]
}

func GetDirn(properties []int, lift int) int { // I calculate_order
	return properties[3*lift+1]
}

func GetState(properties []int, lift int) int { // I admin, calculate_order
	return properties[3*lift+2]
}

func SetSingleLiftProperties(properties []int, lift int, backupProperties []int) {
	SetLastFloor(properties, lift, GetLastFloor(backupProperties, lift))
	SetDirn(properties, lift, GetDirn(backupProperties, lift))
	SetState(properties, lift, GetState(backupProperties, lift))
}

func SetPropertiesFromBackup(properties []int, lift int, backupProperties []int) {
	for elev := 0; elev < MAX_N_LIFTS; elev++ {
		if elev != lift {
			SetLastFloor(properties, elev, GetLastFloor(backupProperties, elev))
			SetDirn(properties, elev, GetDirn(backupProperties, elev))
			SetState(properties, elev, GetState(backupProperties, elev))
		}
	}
}
