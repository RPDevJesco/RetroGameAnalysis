package connection

// Driver interface for emulator communication
type Driver interface {
	// Connect establishes connection to the emulator
	Connect() error

	// ReadMemoryBlocks reads multiple memory blocks
	ReadMemoryBlocks(blocks []MemoryBlock) (map[uint32][]byte, error)

	// WriteBytes writes data to a specific memory address
	WriteBytes(address uint32, data []byte) error

	// Close closes the connection
	Close() error
}

// MemoryBlock represents a contiguous block of memory
type MemoryBlock struct {
	Name  string
	Start uint32
	End   uint32
}
