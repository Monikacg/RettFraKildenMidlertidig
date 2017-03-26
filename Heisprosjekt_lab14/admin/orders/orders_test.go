package orders

import (
	"fmt"
	"testing"

	. "./../../definitions"
)

func TestFn(t *testing.T) {
	// Create order matrix test
	orders := InitializeOrders()
	fmt.Println("Create order matrix test: ")
	fmt.Println(orders)

	// Add order test: adds orders on all possible.
	//func AddOrder(orders [][]int, floor, lift, button_call int)
	for i := 0; i < N_FLOORS; i++ {
		for j := 0; j < MAX_N_LIFTS; j++ {
			for k := 0; k < 3; k++ {
				AddOrder(orders, i, j, k)
			}
		}
	}
	fmt.Println("Add order test: ")
	fmt.Println(orders)

	/*
	  // Delete order test
	  //func delete_order(orders [][]int, floor, lift, button_call int)
	  for i := 0; i < N_FLOORS; i++ {
	    for j := 0; j < MAX_N_LIFTS; j++ {
	      for k := 0; k < 3; k++ {
	        Delete_order(orders, 1, 2, 0)
	      }
	    }
	  }
	  fmt.Println("Delete order test: ")
	  fmt.Println(orders)
	*/

	// Assign order test
	/*func delete_order(orders [][]int, floor, lift, button_call int)
	for i := 0; i < N_FLOORS; i++ {
		for j := 0; j < MAX_N_LIFTS; j++ {
			for k := 0; k < 3; k++ {
				AssignOrder(orders, i, j, k)
			}
		}
	}
	fmt.Println("Assign order test: ")
	fmt.Println(orders)
	*/
	/*
	  // Assign orders test
	  //func delete_order(orders [][]int, floor, lift, button_call int)
	  for i := 0; i < N_FLOORS; i++ {
	    for j := 0; j < MAX_N_LIFTS; j++ {
	      AssignOrders(orders, i, j)
	    }
	  }
	  fmt.Println("Assign orders test: ")
	  fmt.Println(orders)
	*/

	/*
	  // Deassign order test
	  // func deassignOrders(orders [][]int, lift int)
	  for i := 0; i < N_FLOORS; i++ {
	    for j := 0; j < MAX_N_LIFTS; j++ {
	      for k := 0; k < 3; k++ {
	        DeassignOrders(orders, 0)
	      }
	    }
	  }
	  fmt.Println("Deassign order test: ")
	  fmt.Println(orders)
	*/

	// Complete order test
	// func CompleteOrder(orders [][]int, floor, lift int)
	for i := 0; i < N_FLOORS; i++ {
		for j := 0; j < MAX_N_LIFTS; j++ {
			for k := 0; k < 3; k++ {
				CompleteOrder(orders, 1, 2)
			}
		}
	}
	fmt.Println("Complete order test: ")
	fmt.Println(orders)

}
