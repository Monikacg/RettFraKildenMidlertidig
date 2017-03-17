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

type Button struct { // Brukes på button_outside_chan. Kanskje endre til å bruke DIRN_UP og DIRN_DOWN?
	Floor      int // Kan kutte ned til 1 button channel.
	Button_dir int
}

type Order struct { // Bru// NB! GÅ gjennom og endre til stor bokstav i koden også.
	//kes på local_order_chan
	Category string // "LIGHT"/"DIR"
	Order    int    // DIRN_DOWN/UP/STOP, BUTTON_CALL_UP/DOWN/COMMAND
	Floor    int    //0-3 (0-N_FLOORS)
	Value    int    // ON/OFF for lys, settes bare for "LIGHT"
} // Floor trengs ikke på doorlight, value trengs ikke på retn.

type Udp struct {
	ID        int
	Type      string
	Floor     int
	ExtraInfo int // If Type == "BUTTON": type of BUTTON
	// if Type == " NewOrder": new Destination
}

type OverNetwork struct {
	ThisIsAnAck bool
	AckersID    int
	Message     Udp
}

type Ack struct {
	Message Udp
	Ackers  []int
}

type Peer struct {
	Change      string
	ChangedPeer int
}

type CurrPeers struct {
	Peers []int
}

type BackUp struct {
	Info       string
	SenderID   int
	Orders     [][]int
	Properties []int
}
