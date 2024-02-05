package tasklimiters

type Status int16

const (
	Started Status = iota + 1 // Started
	Running                   // Running
	Stopped                   // Stopped
)
