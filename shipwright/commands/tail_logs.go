package commands

import (
	"fmt"
	"os"
	"os/signal"
	"sync"

	"github.com/reverb/exeggutor/shipwright"
	"github.com/reverb/exeggutor/shipwright/client/caprica"
	"github.com/reverb/exeggutor/shipwright/client/ssh"
)

// TailLogsCommand is a command to tail logs from servers.
type TailLogsCommand struct {
	ClusterName string `short:"c" long:"clusters" description:"The clusters to select services from" required:"true"`
	ServiceName string `short:"s" long:"services" description:"The services to tail logs for" required:"true"`
	config      *shipwright.Config
}

func NewTailLogsCommand(config *shipwright.Config) *TailLogsCommand {
	return &TailLogsCommand{config: config}
}

func (t *TailLogsCommand) Execute(args []string) error {
	inventory := caprica.NewInventory(t.config)
	items, err := inventory.FetchInventory(t.ClusterName, t.ServiceName)
	if err != nil {
		fmt.Errorf("Couldn't fetch the inventory for %s in %s because %v", t.ServiceName, t.ClusterName, err)
		return err
	}

	stream := make(chan string, 100)
	var closing []chan<- chan struct{}

	for _, item := range items {
		client := ssh.New(t.config)
		err := client.Connect(&item)
		if err != nil {
			fmt.Errorf("Failed to connect, because: %v", err)
			break
		}
		s, err := client.RunStreaming(inventory.TailCommand(item.Name))
		if err != nil {
			fmt.Errorf("Failed to get log stream, because %v", err)
			break
		}
		c := make(chan chan struct{})
		closing = append(closing, c)
		go func() {
			for {
				select {
				case evt := <-s:
					stream <- fmt.Sprintf("%s[%s][%s] %s", evt.Host.Cluster, evt.Host.Name, evt.Host.PublicHost, string(evt.Line))
				case ch := <-c:
					client.Disconnect()
					close(s)
					ch <- struct{}{}
					return
				default:
				}
			}
		}()
	}

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt)
	wg := &sync.WaitGroup{}
	wg.Add(len(closing))
	go func() {
		for {
			select {
			case ln := <-stream:
				fmt.Println(ln)
			case <-sigCh:
				for _, closeCh := range closing {
					r := make(chan struct{})
					closeCh <- r
					go func() {
						<-r
						wg.Add(-1)
					}()
				}
			}
		}
	}()

	wg.Wait()
	return nil
}
