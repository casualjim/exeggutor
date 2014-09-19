// Package boatwright provides ...
package boatwright

// Config holds the config for all the environments
type Config struct {
	Dev  *EnvConfig
	Prod *EnvConfig
	SSH  SshConfig
}

// HttpConfig holds the configuration for talking to a remote host
type HttpConfig struct {
	URL      string
	User     string
	Password string `yaml:"pass"`
}

// SshConfig holds the configuration for establishing ssh connections
type SshConfig struct {
	User    string
	KeyFile string `yaml:"private_key"`
}

// EnvConfig holds the configuration for the other jobs
type EnvConfig struct {
	Caprica        HttpConfig
	LogPaths       map[string]string
	DockerRegistry HttpConfig `yaml:"docker_registry"`
}

// InventoryItem is an item that is retrieved from an inventory server
type InventoryItem struct {
	PublicHost  string
	PrivateHost string
	Name        string
	Cluster     string
}

// Inventory is a service that serves inventories.
type Inventory interface {
	TailCommand(serviceName string) string
	FetchInventory(clusterNames, serviceNames string) ([]InventoryItem, error)
	LogPath(serviceName string) string
}

// RemoteEvent is a structure that represents an event received from a remote host.
type RemoteEvent struct {
	Host InventoryItem
	Line []byte
}

// DockerConfig holds the information necessary to talk to work with docker.
type DockerConfig struct {
	Registry string
}
