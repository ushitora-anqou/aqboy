package cpu

import (
	"testing"
)

func TestAdd8(t *testing.T) {
	table := [][4]uint8{
		{0, 0, 0, 0},
		{128, 127, 255, 0},
		{128, 128, 0, 1},
	}

	for _, entry := range table {
		lhs := entry[0]
		rhs := entry[1]
		expectedVal := entry[2]
		expectedCarry := entry[3] != 0
		val, carry := add8(lhs, rhs, false)
		if !(val == expectedVal && carry == expectedCarry) {
			t.Fatalf("add8: (got: %v, %v) (expected: %v, %v) = %v + %v", val, carry, expectedVal, expectedCarry, lhs, rhs)
		}
	}
}

func TestAdd4(t *testing.T) {
	table := [][4]uint8{
		{0, 0, 0, 0},
		{8, 7, 15, 0},
		{8, 8, 0, 1},
	}

	for _, entry := range table {
		lhs := entry[0]
		rhs := entry[1]
		expectedVal := entry[2]
		expectedCarry := entry[3] != 0
		val, carry := add4(lhs, rhs, false)
		if !(val == expectedVal && carry == expectedCarry) {
			t.Fatalf("add8: (got: %v, %v) (expected: %v, %v) = %v + %v", val, carry, expectedVal, expectedCarry, lhs, rhs)
		}
	}
}

func TestSub8(t *testing.T) {
	table := [][4]uint8{
		{0, 0, 0, 0},
		{0, 1, 255, 1},
	}

	for _, entry := range table {
		lhs := entry[0]
		rhs := entry[1]
		expectedVal := entry[2]
		expectedCarry := entry[3] != 0
		val, carry := sub8(lhs, rhs, false)
		if !(val == expectedVal && carry == expectedCarry) {
			t.Fatalf("add8: (got: %v, %v) (expected: %v, %v) = %v + %v", val, carry, expectedVal, expectedCarry, lhs, rhs)
		}
	}
}

func TestSub4(t *testing.T) {
	table := [][4]uint8{
		{0, 0, 0, 0},
		{0, 1, 15, 1},
	}

	for _, entry := range table {
		lhs := entry[0]
		rhs := entry[1]
		expectedVal := entry[2]
		expectedCarry := entry[3] != 0
		val, carry := sub4(lhs, rhs, false)
		if !(val == expectedVal && carry == expectedCarry) {
			t.Fatalf("add8: (got: %v, %v) (expected: %v, %v) = %v + %v", val, carry, expectedVal, expectedCarry, lhs, rhs)
		}
	}
}
