package parsers

// Parser is the base interface for all parsers
type Parser interface {
	// GetType returns the type of parser
	GetType() string
}

// StringParser is the interface for parsers that process complete strings
type StringParser interface {
	Parser
	// Parse processes a complete string and returns structured data
	Parse(input string) (interface{}, error)
}

// StreamParser is the interface for parsers that process streaming content
type StreamParser interface {
	Parser
	// ProcessChunk processes a chunk of streaming data
	ProcessChunk(chunk []byte) error
	// Finalize processes any remaining content and finalizes the state
	Finalize() error
}