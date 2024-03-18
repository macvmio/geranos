package bitarray

// Bitarray interface as per your request
type Bitarray interface {
	Set(idx int)
	Get(idx int) bool
	GetByte(pos int) byte
}

// bitArray is a simple implementation of the Bitarray interface.
type bitArray struct {
	data []uint8
}

// newBitArray creates a new bitArray with a given size.
func newBitArray(size int) *bitArray {
	return &bitArray{
		data: make([]uint8, (size+7)/8),
	}
}

// Set sets the bit at idx to 1.
func (b *bitArray) Set(idx int) {
	byteIndex := idx / 8
	bitIndex := uint(idx % 8) // Bit position in the byte
	b.data[byteIndex] |= 1 << bitIndex
}

// Get returns true if the bit at idx is 1.
func (b *bitArray) Get(idx int) bool {
	byteIndex := idx / 8
	bitIndex := uint(idx % 8) // Bit position in the byte
	return (b.data[byteIndex]>>bitIndex)&1 == 1
}

// GetByte returns a byte starting from the bit at index pos*8.
func (b *bitArray) GetByte(pos int) byte {
	return b.data[pos]
}

// braillePattern returns the Unicode Braille Pattern character for a given byte mask.
func braillePattern(mask byte) rune {
	// Base Unicode point for Braille Patterns (all dots blank)
	base := rune(0x2800)
	return base + rune(mask)
}

func (b *bitArray) String() string {
	res := make([]rune, 0)
	res = append(res, '[')
	for _, m := range b.data {
		res = append(res, braillePattern(m))
	}
	res = append(res, ']')
	return string(res)
}

func New(size int) Bitarray {
	return newBitArray(size)
}
