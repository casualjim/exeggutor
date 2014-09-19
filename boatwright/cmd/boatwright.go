package main

import (
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/reverb/exeggutor/boatwright"
	"github.com/reverb/exeggutor/boatwright/commands"
)

type opts struct {
}

func main() {

	initCmd := &commands.InitCommand{}

	ec := boatwright.Config{}
	initCmd.Execute(&ec)
	var options opts
	parser := flags.NewParser(&options, flags.Default)
	parser.AddCommand("tail-logs", "Tail the logs of services", "Tails the logs of all the services selected from the specified clusters", commands.NewTailLogsCommand(&ec))
	parser.AddCommand("search-docker", "Search the docker registry", "Searches the docker registry for the specified search string", commands.NewSearchDockerCommand(&ec))

	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}
