package lift_control

import (
	"testing"

	. "./../definitions"
)

func TestFn(t *testing.T) {
	button_chan := make(chan Button, 100)
	floor_sensor_chan := make(chan int, 100)

	local_order_chan := make(chan Order, 100)

	Lift_control_init(button_chan, floor_sensor_chan, local_order_chan)
}
