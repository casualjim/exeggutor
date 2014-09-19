package commands

import "github.com/reverb/exeggutor/boatwright"

// SearchDockerCommand is a command to search the private docker registry
type SearchDockerCommand struct {
	Section string
	config  *boatwright.Config
}

// NewTailLogsCommand creates a new instace of the tail logs command initialized with the global config
func NewSearchDockerCommand(config *boatwright.Config) *TailLogsCommand {
	return &TailLogsCommand{config: config}
}

func (s *SearchDockerCommand) Execute(args []string) error {
	return nil
}
