package githttp

import (
	"fmt"
)

// An event (triggered on push/pull)
type Event struct {
	Type EventType `json:"type"`

	////
	// Set for pushes and pulls
	////

	// SHA of commit
	Commit string `json:"commit"`

	// Path to bare repo
	Dir string

	////
	// Set for pushes or tagging
	////
	Tag    string `json:"tag,omitempty"`
	Last   string `json:"last,omitempty"`
	Branch string `json:"branch,omitempty"`
}

type EventType int

// Possible event types
const (
	TAG = iota + 1
	PUSH
	FETCH
)

func (e EventType) String() string {
	switch e {
	case TAG:
		return "tag"
	case PUSH:
		return "push"
	case FETCH:
		return "fetch"
	}
	return "unknown"
}

func (e EventType) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, e)), nil
}

func (e EventType) UnmarshalJSON(data []byte) error {
	str := string(data[:])
	switch str {
	case "tag":
		e = TAG
	case "push":
		e = PUSH
	case "fetch":
		e = FETCH
	default:
		return fmt.Errorf("'%s' is not a known git event type")
	}
	return nil
}
