package executor

import "time"

// Options represents configurable options for the AI executor
type Options struct {
	DefaultTimeout   time.Duration
	DefaultModel     string
	DefaultMaxTokens int
	DefaultTemperature float64
	MaxBufferSize   int
}

// DefaultOptions returns a new Options struct with default values
func DefaultOptions() *Options {
	return &Options{
		DefaultTimeout:   time.Second * 60, // 1 minute timeout
		DefaultModel:     "gemini-2.0-flash", // Default model
		DefaultMaxTokens: 4000,
		DefaultTemperature: 0.3,
		MaxBufferSize:   1024 * 1024, // 1MB buffer size
	}
}

// WithTimeout sets the default timeout
func (o *Options) WithTimeout(timeout time.Duration) *Options {
	o.DefaultTimeout = timeout
	return o
}

// WithModel sets the default model
func (o *Options) WithModel(model string) *Options {
	o.DefaultModel = model
	return o
}

// WithMaxTokens sets the default max tokens
func (o *Options) WithMaxTokens(maxTokens int) *Options {
	o.DefaultMaxTokens = maxTokens
	return o
}

// WithTemperature sets the default temperature
func (o *Options) WithTemperature(temperature float64) *Options {
	o.DefaultTemperature = temperature
	return o
}

// WithMaxBufferSize sets the max buffer size
func (o *Options) WithMaxBufferSize(size int) *Options {
	o.MaxBufferSize = size
	return o
}