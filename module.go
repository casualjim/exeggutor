package exeggutor

// Module An interface to describe things that have a start and stop method
type Module interface {
	Start() error
	Stop() error
}

// IDGenerator an abstraction for a pluggable id generator
type IDGenerator interface {
	Next() (string, error)
}
