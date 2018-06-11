package node

import "time"

type Action int

const (
	ActionStop Action = iota
)

type Event struct {
	Action   Action
	Duration Duration
	Delay    Duration
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}
