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

func SetDirn(properties []int, liftID, dirn int) {
	properties[3*liftID+1] = dirn
}

func SetState(properties []int, liftID, state int) {
	properties[3*liftID+2] = state
}

func GetLastFloor(properties []int, liftID int) int { // I calculate_order
	return properties[3*liftID]
}

func GetDirn(properties []int, liftID int) int { // I calculate_order
	return properties[3*liftID+1]
}

func GetState(properties []int, liftID int) int { // I admin, calculate_order
	return properties[3*liftID+2]
}

func SetSingleLiftProperties(properties []int, liftID int, backupProperties []int) {
	SetLastFloor(properties, liftID, GetLastFloor(backupProperties, liftID))
	SetDirn(properties, liftID, GetDirn(backupProperties, liftID))
	SetState(properties, liftID, GetState(backupProperties, liftID))
}

func SetPropertiesFromBackup(properties []int, liftID int, backupProperties []int) {
	for elev := 0; elev < MAX_N_LIFTS; elev++ {
		if elev != liftID {
			SetLastFloor(properties, elev, GetLastFloor(backupProperties, elev))
			SetDirn(properties, elev, GetDirn(backupProperties, elev))
			SetState(properties, elev, GetState(backupProperties, elev))
		}
	}
}
