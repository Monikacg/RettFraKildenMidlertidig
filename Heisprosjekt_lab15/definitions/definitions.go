package definitions

/* NB! These constants are defined to fit with the functions in elev.h.
The constants have not been directly imported from C simply because we
didn't find a way to when writing this. One obvious improvement of the
code would be to import (some) of these constants in order to fix a
weakness in the system.
*/

const (
	// Number of floors/buttons
	N_FLOORS    = 4
	N_BUTTONS   = 3
	MAX_N_LIFTS = 3

	// Lift states
	INIT      = -1
	IDLE      = 0
	MOVING    = 1
	DOOR_OPEN = 2
	STUCK     = 3

	// Button calls
	BUTTON_CALL_UP   = 0
	BUTTON_CALL_DOWN = 1
	BUTTON_COMMAND   = 2

	// Motor directions
	DIRN_DOWN = -1
	DIRN_STOP = 0
	DIRN_UP   = 1

	NOT_VALID = -2

	ON  = 1
	OFF = 0
)

type Button struct {
	Floor      int
	ButtonType int
}

type Instruction struct {
	Category string
	Order    int
	Floor    int
	Value    int
}

type Message struct {
	Info       string
	ID         int
	Floor      int
	ButtonType int
}

type Broadcast struct {
	ThisIsAnAck    bool
	AckersID       int
	SequenceStart  int
	SequenceNumber int
	Message
}

type Ack struct {
	SequenceNumber int
	Message
	Ackers []int
}

type ChangedLift struct {
	TypeOfChange      string
	IDofChangedLift int
}

type Lifts struct {
	AliveLifts []int
}

type BackUp struct {
	Info       string
	SenderID   int
	Orders     [][]int
	Properties []int
}
