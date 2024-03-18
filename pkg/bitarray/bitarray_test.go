package bitarray

import (
	"fmt"
	"testing"
)

func TestBitArray_SetGet(t *testing.T) {
	// Test setting and getting bits
	tests := []struct {
		setIdx   int
		getIdx   int
		expected bool
	}{
		{1, 1, true},
		{3, 3, true},
		{4, 3, false},
	}

	for _, tt := range tests {
		size := 64
		ba := newBitArray(size)
		ba.Set(tt.setIdx)
		if ba.Get(tt.getIdx) != tt.expected {
			t.Errorf("After setting bit at index %d, expected %t, got %t", tt.setIdx, tt.expected, ba.Get(tt.getIdx))
		}
	}

}

func TestBitArray_GetByte(t *testing.T) {
	size := 64
	ba := newBitArray(size)
	ba.Set(3)
	ba.Set(4)
	expectedByte := byte(0b00011000)

	if byteVal := ba.GetByte(0); byteVal != expectedByte {
		t.Errorf("Expected byte %08b, got %08b", expectedByte, byteVal)
	}
	ba.Set(0)
	expectedByte = byte(0b00011001)
	if byteVal := ba.GetByte(0); byteVal != expectedByte {
		t.Errorf("Expected byte %08b, got %08b", expectedByte, byteVal)
	}
	ba.Set(8)
	expectedByte = byte(0b00000001)
	if byteVal := ba.GetByte(1); byteVal != expectedByte {
		t.Errorf("Expected byte %08b, got %08b", expectedByte, byteVal)
	}
}

func TestBitArray_String(t *testing.T) {
	ba := New(64)
	for idx := 0; idx < 64; idx++ {
		ba.Set(idx)
		fmt.Printf("bitarray: %s\n", ba)
	}
}
