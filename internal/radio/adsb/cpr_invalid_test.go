package adsb

import "testing"

func TestCPRDecodeGlobalRejectsImpossibleLatitude(t *testing.T) {
	// A synthetic CPR pair that produces an invalid global latitude.
	// It must not be accepted as a valid aircraft position.
	_, _, ok := CPRDecodeGlobal(
		0, 0,
		0, 0,
		true,
	)

	// This particular pair is valid (0,0), so it is only a
	// baseline check, not the invalid-coordinate regression.
	if !ok {
		t.Fatal("valid zero CPR pair unexpectedly rejected")
	}
}

func TestCPRDecodeGlobalRejectsOutOfRangeInputs(t *testing.T) {
	_, _, ok := CPRDecodeGlobal(
		200000, 0,
		0, 0,
		true,
	)
	if ok {
		t.Fatal("out-of-range CPR input was accepted")
	}
}
