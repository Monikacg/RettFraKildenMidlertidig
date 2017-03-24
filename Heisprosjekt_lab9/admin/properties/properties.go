package properties

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

func SetLastFloor(properties []int, liftID, lastFloor int) {
	properties[3*liftID] = lastFloor
}

func SetDirection(properties []int, liftID, direction int) {
	properties[3*liftID+1] = direction
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

// SetSingleLiftProperties uses liftID as the ID of the lift  that it will extract the properties from from the backupProperties
func SetSingleLiftProperties(properties []int, liftID int, backupProperties []int) {
	SetLastFloor(properties, liftID, GetLastFloor(backupProperties, liftID))
	SetDirection(properties, liftID, GetDirection(backupProperties, liftID))
	SetState(properties, liftID, GetState(backupProperties, liftID))
}

// SetPropertiesFromBackup uses liftID as the ID of the lift that will take in the new information from the backup
func SetPropertiesFromBackup(properties []int, liftID int, backupProperties []int) {
	for elev := 0; elev < MAX_N_LIFTS; elev++ {
		if elev != liftID {
			SetLastFloor(properties, elev, GetLastFloor(backupProperties, elev))
			SetDirection(properties, elev, GetDirection(backupProperties, elev))
			SetState(properties, elev, GetState(backupProperties, elev))
		}
	}
}
