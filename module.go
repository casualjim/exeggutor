package exeggutor

// Module An interface to describe things that have a start and stop method
type Module interface {
	Start()
	Stop()
}
