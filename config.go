package exeggutor

type Config struct {
	ZookeeperUrl    string `json:"zookeeper,omitempty" long:"zk" description:"The uri for zookeeper in the form of zk://localhost:2181/root"`
	MesosMaster     string `json:"mesos,omitempty" long:"mesos" description:"The uri for the mesos master"`
	DataDirectory   string `json:"dataDirectory,omitempty" long:"data_dir" description:"The base path for storing the data" default:"./data"`
	LogDirectory    string `json:"logDirectory,omitempty" long:"log_dir" description:"The directory to store log files in" default:"./logs"`
	WorkDirectory   string `json:"workDirectory,omitempty" long:"work_dir" description:"The directory to use when doing temporary work" default:"/tmp/agora-wrk-$RANDOM"`
	ConfigDirectory string `json:"confDirectory,omitempty" long:"conf" description:"The directory where to find the config files" default:"./etc"`
	Port            string `json:"port,omitempty" long:"port" description:"The port to listen on for web requests" default:"8000"`
	Interface       string `json:"interface,omitempty" long:"listen" description:"The interface to use to listen for web requests" default:"0.0.0.0"`
}
