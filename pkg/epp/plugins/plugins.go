package plugins

import (
	"errors"
)

const (
	separator = "/"
)

type TypedName struct {
	// Type returns the type of a plugin.
	Type string
	// Name returns the name of a plugin instance.
	Name string
}

// String returns the type and name rendered as "<name>/<type>".
func (tn TypedName) String() string {
	return tn.Name + separator + tn.Type
}

type Plugin interface {
	// TypedName returns the type and name tuple of this plugin instance.
	TypedName() TypedName
}

var (
	// ErrNotFound is the not found error message.
	ErrNotFound = errors.New("not found")
)

// StateKey is the type of keys stored in PluginState.
type StateKey string

// StateData is a generic type for arbitrary data stored in PluginState.
type StateData interface {
	// Clone is an interface to make a copy of StateData.
	Clone() StateData
}
