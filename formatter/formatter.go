package formatter

import (
	"errors"
	"fmt"
)

// Error define
var (
	ErrFormatterNotFound = errors.New("formatter not found")
)

var (
	formatters = make(map[string]func() Formatter)
)

// Formatter
type Formatter interface {
	Initialize(data []byte) error
	Format(val interface{}) ([]byte, error)
}

// RegisterFormatter register a formatter
func RegisterFormatter(name string, create func() Formatter) {
	formatters[name] = create
}

// NewFormatter creates a formatter by name
func NewFormatter(name string) (Formatter, error) {
	create := formatters[name]
	if create == nil {
		return nil, fmt.Errorf("%v %w", name, ErrFormatterNotFound)
	}
	return create(), nil
}

// AllFormatter returns all name of registered formatter
func AllFormatter() []string {
	names := make([]string, 0, len(formatters))
	for n, _ := range formatters {
		names = append(names, n)
	}
	return names
}
