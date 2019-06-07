package main

import "testing"

func TestInitW1(t *testing.T){
	initGPIO()
	initW1()
	t.Error(
		"Test","failed",
	)
}
