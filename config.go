package exeggutor

import (
	"net/url"
	"strconv"
	"strings"
)

// Config the main configuration object to use in the application
type Config struct {
	ZookeeperURL    string             `json:"zookeeper,omitempty" long:"zk" description:"The uri for zookeeper in the form of zk://localhost:2181/root"`
	MesosMaster     string             `json:"mesos,omitempty" long:"mesos" description:"The uri for the mesos master"`
	DataDirectory   string             `json:"dataDirectory,omitempty" long:"data_dir" description:"The base path for storing the data" default:"./data"`
	StaticFiles     string             `json:"staticFiles,omitempty" long:"public" description:"The directory to find the static files for this app" default:"./static/build"`
	WorkDirectory   string             `json:"workDirectory,omitempty" long:"work_dir" description:"The directory to use when doing temporary work" default:"/tmp/agora-wrk-$RANDOM"`
	ConfigDirectory string             `json:"confDirectory,omitempty" long:"conf" description:"The directory where to find the config files" default:"./etc"`
	Port            int                `json:"port,omitempty" long:"port" description:"The port to listen on for web requests" default:"8000"`
	Interface       string             `json:"interface,omitempty" long:"listen" description:"The interface to use to listen for web requests" default:"0.0.0.0"`
	Mode            string             `json:"mode,omitempty" long:"mode" description:"The mode in which to run this application (dev, prod, stage, jenkins)" default:"development"`
	FrameworkInfo   *FrameworkConfig   `json:"framework,omitempty"`
	DockerIndex     *DockerIndexConfig `json:"dockerIndex,omitempty"`
	Logging         *LoggingConfig     `json:"logging,omitempty"`
}

// DockerIndexConfig contains the configuration properties for a docker index
type DockerIndexConfig struct {
	Host       string `json:"host,omitempty" long:"docker_host" description:"The host or domain name for the docker registry"`
	Port       int    `json:"port,omitempty" long:"docker_port" description:"The port for the docker registry" default:"5000"`
	Scheme     string `json:"scheme,omitempty" long:"docker_scheme" description:"The scheme to use when calling docker registry" default:"http"`
	APIVersion string `json:"api_version,omitempty" long:"docker_api_version" description:"The docker registry api version" default:"v1"`
	User       string `json:"user,omitempty" long:"docker_user" description:"The user to authenticate with at the docker registry" default:""`
	Pass       string `json:"pass,omitempty" long:"docker_pass" description:"The password to authenticate with at the docker registry" default:""`
}

// ToURL generates a url from the properties of the docker index config
func (d *DockerIndexConfig) ToURL() *url.URL {
	res := &url.URL{
		Scheme: d.Scheme,
		Host:   strings.Join([]string{d.Host, strconv.Itoa(d.Port)}, ":"),
		Path:   strings.Join([]string{"/", d.APIVersion}, ""),
	}
	if d.User != "" {
		res.User = url.UserPassword(d.User, d.Pass)
	}
	return res
}

// ToProtoURL generates a url from the properties of the docker index config
func (d *DockerIndexConfig) ToProtoURL() *url.URL {
	return &url.URL{
		Scheme: "docker",
		Host:   d.Host,
		Path:   strings.Join([]string{"/", d.APIVersion}, ""),
	}
}

// FrameworkConfig framework config contains configuration specific to mesos.
// It has things like a name of the framework and user to use when running applications
// on mesos
type FrameworkConfig struct {
	User                   string `json:"user,omitempty" long:"framework_user" description:"The user under which this framework should authenticate"`
	Name                   string `json:"name,omitempty" long:"framework_name" description:"The name of this framework" default:"Agora"`
	HealthCheckConcurrency int    `json:"healthCheckConcurrency" long:"health_check_concurrency" description:"The number of health check workers" default:"5"`
}

// LoggingConfig contains the configuration for the logging
// It configures levels and possibly appenders
type LoggingConfig struct {
	Level        string `json:"level,omitempty" long:"log_level" description:"The level at which to log" default:"DEBUG"`
	Colorize     bool   `json:"colorize,omitempty" long:"log_colorize" description:"Use colors in logs" default:"true"`
	Pattern      string `json:"pattern,omitempty" long:"log_pattern" description:"The pattern to use for logging" default:"%{level} %{message}"`
	LogDirectory string `json:"logDirectory,omitempty" long:"log_dir" description:"The directory to store log files in" default:"./logs"`
}
