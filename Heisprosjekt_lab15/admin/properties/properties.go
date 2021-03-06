package properties

import (
	. "../../definitions"
)

func InitializeLiftProperties() []int {
	properties := make([]int, 3*MAX_N_LIFTS)
	for liftID := 0; liftID < MAX_N_LIFTS; liftID++ {
		properties[3*liftID] = NOT_VALID   // Last floor
		properties[3*liftID+1] = DIRN_DOWN // Direction
		properties[3*liftID+2] = INIT      // State
	}
	return properties
}

func SetLastFloor(properties []int, liftID, lastFloor int) {
	properties[3*liftID] = lastFloor
}

func SetDirection(properties []int, liftID, dirn int) {
	properties[3*liftID+1] = dirn
}

func SetState(properties []int, liftID, state int) {
	properties[3*liftID+2] = state
}

func GetLastFloor(properties []int, liftID int) int { // I calculate_order
	return properties[3*liftID]
}

func GetDirection(properties []int, liftID int) int { // I calculate_order
	return properties[3*liftID+1]
}

func GetState(properties []int, liftID int) int { // I admin, calculate_order
	return properties[3*liftID+2]
}

func SetSingleLiftProperties(properties []int, liftID int, backupProperties []int) {
	SetLastFloor(properties, liftID, GetLastFloor(backupProperties, liftID))
	SetDirection(properties, liftID, GetDirection(backupProperties, liftID))
	SetState(properties, liftID, GetState(backupProperties, liftID))
}

func SetOtherLiftsPropertiesFromBackup(properties []int, liftID int, backupProperties []int) {
	for elev := 0; elev < MAX_N_LIFTS; elev++ {
		if elev != liftID {
			SetLastFloor(properties, elev, GetLastFloor(backupProperties, elev))
			SetDirection(properties, elev, GetDirection(backupProperties, elev))
			SetState(properties, elev, GetState(backupProperties, elev))
		}
	}
}
