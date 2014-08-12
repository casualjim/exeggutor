package caprica

import (
	"fmt"

	"github.com/reverb/exeggutor/boatwright"
)

// CapricaInventory is an inventory implementation based on caprica
type CapricaInventory struct {
	config     *boatwright.Config
	devClient  *CapricaClient
	prodClient *CapricaClient
}

func NewInventory(config *boatwright.Config) boatwright.Inventory {
	devClient := New(&config.Dev.Caprica)
	prodClient := New(&config.Prod.Caprica)
	return &CapricaInventory{
		config:     config,
		devClient:  devClient,
		prodClient: prodClient,
	}
}

// TailCommand returns the command to tail a file on a remote host
func (i *CapricaInventory) TailCommand(serviceName string) string {
	return "tail -F " + i.LogPath(serviceName)
}

func (i *CapricaInventory) LogPath(serviceName string) string {
	return fmt.Sprintf("/var/wordnik/%s/current/logs/stdout.txt", serviceName)
}

// FetchInventoryForService fetches the inventory for the specified clusters and services
func (i *CapricaInventory) FetchInventory(clusterNames, serviceNames string) ([]boatwright.InventoryItem, error) {
	svcs, err := i.prodClient.GetInstances(clusterNames, serviceNames)
	if err != nil {
		return nil, err
	}
	svcs2, err := i.devClient.GetInstances(clusterNames, serviceNames)
	if err != nil {
		return nil, err
	}
	svcs = append(svcs, svcs2...)

	var result []boatwright.InventoryItem
	for _, svc := range svcs {
		if svc.Instance != nil && svc.Instance.State != nil && svc.Instance.State.Name == "running" {
			result = append(result, boatwright.InventoryItem{
				PublicHost:  svc.Instance.PublicIPAddress,
				PrivateHost: svc.Instance.PrivateIPAddress,
				Name:        svc.Name,
				Cluster:     svc.Cluster,
			})
		}
	}
	return result, nil
}
