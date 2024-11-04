package bitarray

// BitArray is a simple implementation of the Bitarray interface.
type BitArray struct {
	data []uint8
}

// newBitArray creates a new BitArray with a given size.
func newBitArray(size int) *BitArray {
	return &BitArray{
		data: make([]uint8, (size+7)/8),
	}
}

// Set sets the bit at idx to 1.
func (b *BitArray) Set(idx int) {
	byteIndex := idx / 8
	bitIndex := uint(idx % 8) // Bit position in the byte
	b.data[byteIndex] |= 1 << bitIndex
}

// Get returns true if the bit at idx is 1.
func (b *BitArray) Get(idx int) bool {
	byteIndex := idx / 8
	bitIndex := uint(idx % 8) // Bit position in the byte
	return (b.data[byteIndex]>>bitIndex)&1 == 1
}

// GetByte returns a byte starting from the bit at index pos*8.
func (b *BitArray) GetByte(pos int) byte {
	return b.data[pos]
}

// swapBits swaps the bits at positions pos1 and pos2 in byte b.
func swapBits(b byte, pos1 uint, pos2 uint) byte {
	// Ensure positions are within the range [0,7]
	if pos1 > 7 || pos2 > 7 {
		// Positions out of range, return b unchanged or handle error
		return b
	}

	// Extract the bits at positions pos1 and pos2
	bit1 := (b >> pos1) & 1
	bit2 := (b >> pos2) & 1

	// If the bits are different, swap them
	if bit1 != bit2 {
		// Create a bitmask where bits at pos1 and pos2 are set
		mask := (uint8(1) << pos1) | (uint8(1) << pos2)
		// Toggle the bits at pos1 and pos2 using XOR
		b ^= mask
	}
	return b
}

// braillePattern returns the Unicode Braille Pattern character for a given byte mask.
func braillePattern(mask byte) rune {
	// Base Unicode point for Braille Patterns (all dots blank)
	base := rune(0x2800)
	mask = swapBits(mask, 3, 4)
	mask = swapBits(mask, 4, 5)
	mask = swapBits(mask, 5, 6)
	return base + rune(mask)
}

// Fill sets all bits up to the n-th bit to 1.
func (b *BitArray) Fill(n int) {
	if n <= 0 {
		return
	}
	totalBits := len(b.data) * 8
	if n > totalBits {
		n = totalBits
	}
	fullBytes := n / 8
	for i := 0; i < fullBytes; i++ {
		b.data[i] = 0xFF
	}
	remainingBits := n % 8
	if remainingBits != 0 && fullBytes < len(b.data) {
		mask := byte((1 << remainingBits) - 1)
		b.data[fullBytes] |= mask
	}
}

func (b *BitArray) String() string {
	res := make([]rune, 0)
	res = append(res, '[')
	for _, m := range b.data {
		res = append(res, braillePattern(m))
	}
	res = append(res, ']')
	return string(res)
}

func New(size int) *BitArray {
	return newBitArray(size)
}
