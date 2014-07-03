// Code generated by protoc-gen-go.
// source: exeggutor.proto
// DO NOT EDIT!

/*
Package protocol is a generated protocol buffer package.

It is generated from these files:
	exeggutor.proto

It has these top-level messages:
	StringKeyValue
	StringIntKeyValue
	DeployedAppComponent
	Application
	ScheduledApp
	HealthCheck
	ApplicationSLA
*/
package protocol

import proto "code.google.com/p/goprotobuf/proto"
import math "math"
import mesos "github.com/reverb/go-mesos/mesos"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

//
// AppStatus is used to indicate where an app is in a lifecycle on the cluster
type AppStatus int32

const (
	// AppStatus_ABSENT the application has no running instances
	AppStatus_ABSENT AppStatus = 1
	// AppStatus_DEPLOYING the application is currently being deployed
	AppStatus_DEPLOYING AppStatus = 2
	// AppStatus_STOPPED the application has been stopped
	AppStatus_STOPPED AppStatus = 3
	// AppStatus_STOPPING the application has is stopping, a command was issued to stop the app
	AppStatus_STOPPING AppStatus = 4
	// AppStatus_STARTING the application has been deployed and is currently starting up
	AppStatus_STARTING AppStatus = 5
	// AppStatus_STARTED the application is fully available for taking requests
	AppStatus_STARTED AppStatus = 6
	// AppStatus_VERY_BUSY the application is still up but timing out very often, best to avoid it for a while
	AppStatus_VERY_BUSY AppStatus = 7
	// AppStatus_UNHEALTHY the application has a running process but is otherwise broken, don't send requests here
	AppStatus_UNHEALTHY AppStatus = 8
)

var AppStatus_name = map[int32]string{
	1: "ABSENT",
	2: "DEPLOYING",
	3: "STOPPED",
	4: "STOPPING",
	5: "STARTING",
	6: "STARTED",
	7: "VERY_BUSY",
	8: "UNHEALTHY",
}
var AppStatus_value = map[string]int32{
	"ABSENT":    1,
	"DEPLOYING": 2,
	"STOPPED":   3,
	"STOPPING":  4,
	"STARTING":  5,
	"STARTED":   6,
	"VERY_BUSY": 7,
	"UNHEALTHY": 8,
}

func (x AppStatus) Enum() *AppStatus {
	p := new(AppStatus)
	*p = x
	return p
}
func (x AppStatus) String() string {
	return proto.EnumName(AppStatus_name, int32(x))
}
func (x *AppStatus) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(AppStatus_value, data, "AppStatus")
	if err != nil {
		return err
	}
	*x = AppStatus(value)
	return nil
}

//
// ComponentType is used to describe what type of service this is.
// This is used for determining montitoring strategy and so on.
// It might also influence the way an application is deployed
type ComponentType int32

const (
	// ComponentType_SERVICE A long-running service
	ComponentType_SERVICE ComponentType = 0
	// ComponentType_TASK A short one-off task
	ComponentType_TASK ComponentType = 1
	// ComponentType_CRON A task scheduled to repeat on a schedule or to be executed at a later, scheduled time
	ComponentType_CRON ComponentType = 2
	// ComponentType_SPARK_JOB A spark job
	ComponentType_SPARK_JOB ComponentType = 3
)

var ComponentType_name = map[int32]string{
	0: "SERVICE",
	1: "TASK",
	2: "CRON",
	3: "SPARK_JOB",
}
var ComponentType_value = map[string]int32{
	"SERVICE":   0,
	"TASK":      1,
	"CRON":      2,
	"SPARK_JOB": 3,
}

func (x ComponentType) Enum() *ComponentType {
	p := new(ComponentType)
	*p = x
	return p
}
func (x ComponentType) String() string {
	return proto.EnumName(ComponentType_name, int32(x))
}
func (x *ComponentType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(ComponentType_value, data, "ComponentType")
	if err != nil {
		return err
	}
	*x = ComponentType(value)
	return nil
}

//
// Distribution is used to decide how an application should be deployed
// This determines whether it needs extraction etc.
type Distribution int32

const (
	// Distributed as a package for the OS package manager (RPM, DEB, ...)
	// This implies that a docker container will be created ad-hoc to install this package.
	// It's probably better to get jenkins to build a proper docker container for your application.
	Distribution_PACKAGE Distribution = 0
	// Distributed as a Docker Container
	Distribution_DOCKER Distribution = 1
	// Ad-Hoc script, fat-jar, single-binary
	Distribution_SCRIPT Distribution = 2
	// Distributed as executable jar
	Distribution_FAT_JAR Distribution = 3
)

var Distribution_name = map[int32]string{
	0: "PACKAGE",
	1: "DOCKER",
	2: "SCRIPT",
	3: "FAT_JAR",
}
var Distribution_value = map[string]int32{
	"PACKAGE": 0,
	"DOCKER":  1,
	"SCRIPT":  2,
	"FAT_JAR": 3,
}

func (x Distribution) Enum() *Distribution {
	p := new(Distribution)
	*p = x
	return p
}
func (x Distribution) String() string {
	return proto.EnumName(Distribution_name, int32(x))
}
func (x *Distribution) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(Distribution_value, data, "Distribution")
	if err != nil {
		return err
	}
	*x = Distribution(value)
	return nil
}

//
// HealthCheckMode the strategy to use when checking for health.
// for the HTTP strategy we make a request
type HealthCheckMode int32

const (
	// For the HTTP strategy it will make a request and expect a 200 OK status
	HealthCheckMode_HTTP HealthCheckMode = 0
	// For the TCP strategy it will just try to connect to the port
	HealthCheckMode_TCP HealthCheckMode = 1
	// For the METRICS strategy it will use the HTTP strategy but additionally the response body will be validated that all components are running fine.
	HealthCheckMode_METRICS HealthCheckMode = 2
)

var HealthCheckMode_name = map[int32]string{
	0: "HTTP",
	1: "TCP",
	2: "METRICS",
}
var HealthCheckMode_value = map[string]int32{
	"HTTP":    0,
	"TCP":     1,
	"METRICS": 2,
}

func (x HealthCheckMode) Enum() *HealthCheckMode {
	p := new(HealthCheckMode)
	*p = x
	return p
}
func (x HealthCheckMode) String() string {
	return proto.EnumName(HealthCheckMode_name, int32(x))
}
func (x *HealthCheckMode) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(HealthCheckMode_value, data, "HealthCheckMode")
	if err != nil {
		return err
	}
	*x = HealthCheckMode(value)
	return nil
}

// StringKeyValue represents a pair of 2 strings used as a replacement for maps
type StringKeyValue struct {
	Key              *string `protobuf:"bytes,1,req,name=key" json:"key,omitempty"`
	Value            *string `protobuf:"bytes,2,req,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *StringKeyValue) Reset()         { *m = StringKeyValue{} }
func (m *StringKeyValue) String() string { return proto.CompactTextString(m) }
func (*StringKeyValue) ProtoMessage()    {}

func (m *StringKeyValue) GetKey() string {
	if m != nil && m.Key != nil {
		return *m.Key
	}
	return ""
}

func (m *StringKeyValue) GetValue() string {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return ""
}

// StringIntKeyValue represents a pair of string to int, used as a replacement for maps
type StringIntKeyValue struct {
	// Key the key of this pair
	Key *string `protobuf:"bytes,1,req,name=key" json:"key,omitempty"`
	// Value the value of this pair
	Value            *int32 `protobuf:"varint,2,req,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *StringIntKeyValue) Reset()         { *m = StringIntKeyValue{} }
func (m *StringIntKeyValue) String() string { return proto.CompactTextString(m) }
func (*StringIntKeyValue) ProtoMessage()    {}

func (m *StringIntKeyValue) GetKey() string {
	if m != nil && m.Key != nil {
		return *m.Key
	}
	return ""
}

func (m *StringIntKeyValue) GetValue() int32 {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return 0
}

//
// DeployedAppComponent is a part of a deployed application
// It links component definition to a mesos task info and an app status
// This keeps track of the state an application is actually in.
// So in an API this could be used to return the info we need for
// displaying what has been deployed and how many instances of it and so forth.
type DeployedAppComponent struct {
	// the application name for the deployed application
	AppName *string `protobuf:"bytes,1,req,name=app_name" json:"app_name,omitempty"`
	// the application component that is deployed
	Component *Application `protobuf:"bytes,2,req,name=component" json:"component,omitempty"`
	// the task id that represents this component in the cluster
	Task *mesos.TaskInfo `protobuf:"bytes,3,req,name=task" json:"task,omitempty"`
	// the status this deployed application is in
	Status *AppStatus `protobuf:"varint,4,req,name=status,enum=protocol.AppStatus,def=1" json:"status,omitempty"`
	// the slave id to which this app is deployed
	Slave            *mesos.SlaveID `protobuf:"bytes,20,opt,name=slave" json:"slave,omitempty"`
	XXX_unrecognized []byte         `json:"-"`
}

func (m *DeployedAppComponent) Reset()         { *m = DeployedAppComponent{} }
func (m *DeployedAppComponent) String() string { return proto.CompactTextString(m) }
func (*DeployedAppComponent) ProtoMessage()    {}

const Default_DeployedAppComponent_Status AppStatus = AppStatus_ABSENT

func (m *DeployedAppComponent) GetAppName() string {
	if m != nil && m.AppName != nil {
		return *m.AppName
	}
	return ""
}

func (m *DeployedAppComponent) GetComponent() *Application {
	if m != nil {
		return m.Component
	}
	return nil
}

func (m *DeployedAppComponent) GetTask() *mesos.TaskInfo {
	if m != nil {
		return m.Task
	}
	return nil
}

func (m *DeployedAppComponent) GetStatus() AppStatus {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return Default_DeployedAppComponent_Status
}

func (m *DeployedAppComponent) GetSlave() *mesos.SlaveID {
	if m != nil {
		return m.Slave
	}
	return nil
}

//
// Application is a part of what makes up a single application.
// It describes the packaging and distribution model of the component
// It also describes the requirements for the component in terms of disk space, cpu and memory
// Furthermore it contains the configuration for the environment and scheme to port mapping
// It also has a status field to track the deployment status of this component
type Application struct {
	// the name for this component
	Name *string `protobuf:"bytes,1,req,name=name" json:"name,omitempty"`
	// the amount of cpu cores this component requires
	Cpus *float32 `protobuf:"fixed32,2,req,name=cpus" json:"cpus,omitempty"`
	// the amount of memory this component requires
	Mem *float32 `protobuf:"fixed32,3,req,name=mem" json:"mem,omitempty"`
	// the amount of disk space this component requires
	DiskSpace *int64 `protobuf:"varint,4,req,name=disk_space" json:"disk_space,omitempty"`
	// the url to use when deploying the application
	DistUrl *string `protobuf:"bytes,5,req,name=dist_url" json:"dist_url,omitempty"`
	// the command to run when this application is deployed
	Command *string `protobuf:"bytes,6,req,name=command" json:"command,omitempty"`
	// the environment variables passed to the application
	Env []*StringKeyValue `protobuf:"bytes,7,rep,name=env" json:"env,omitempty"`
	// the scheme/port mapping for this component
	Ports []*StringIntKeyValue `protobuf:"bytes,8,rep,name=ports" json:"ports,omitempty"`
	// the version for this component
	Version *string `protobuf:"bytes,9,req,name=version" json:"version,omitempty"`
	// the application this component belongs to
	AppName *string `protobuf:"bytes,10,req,name=app_name" json:"app_name,omitempty"`
	// the distribution model used for this component
	Distribution *Distribution `protobuf:"varint,11,req,name=distribution,enum=protocol.Distribution,def=1" json:"distribution,omitempty"`
	// the modality as to how to treat this component
	ComponentType *ComponentType `protobuf:"varint,12,req,name=component_type,enum=protocol.ComponentType,def=0" json:"component_type,omitempty"`
	// where to expect logs to appear
	LogDir *string `protobuf:"bytes,30,opt,name=log_dir" json:"log_dir,omitempty"`
	// where to expect work to appear
	WorkDir *string `protobuf:"bytes,31,opt,name=work_dir" json:"work_dir,omitempty"`
	// where to expect the configuration to be
	ConfDir *string `protobuf:"bytes,32,opt,name=conf_dir" json:"conf_dir,omitempty"`
	// the application SLA to use for this component
	Sla              *ApplicationSLA `protobuf:"bytes,33,opt,name=sla" json:"sla,omitempty"`
	XXX_unrecognized []byte          `json:"-"`
}

func (m *Application) Reset()         { *m = Application{} }
func (m *Application) String() string { return proto.CompactTextString(m) }
func (*Application) ProtoMessage()    {}

const Default_Application_Distribution Distribution = Distribution_DOCKER
const Default_Application_ComponentType ComponentType = ComponentType_SERVICE

func (m *Application) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *Application) GetCpus() float32 {
	if m != nil && m.Cpus != nil {
		return *m.Cpus
	}
	return 0
}

func (m *Application) GetMem() float32 {
	if m != nil && m.Mem != nil {
		return *m.Mem
	}
	return 0
}

func (m *Application) GetDiskSpace() int64 {
	if m != nil && m.DiskSpace != nil {
		return *m.DiskSpace
	}
	return 0
}

func (m *Application) GetDistUrl() string {
	if m != nil && m.DistUrl != nil {
		return *m.DistUrl
	}
	return ""
}

func (m *Application) GetCommand() string {
	if m != nil && m.Command != nil {
		return *m.Command
	}
	return ""
}

func (m *Application) GetEnv() []*StringKeyValue {
	if m != nil {
		return m.Env
	}
	return nil
}

func (m *Application) GetPorts() []*StringIntKeyValue {
	if m != nil {
		return m.Ports
	}
	return nil
}

func (m *Application) GetVersion() string {
	if m != nil && m.Version != nil {
		return *m.Version
	}
	return ""
}

func (m *Application) GetAppName() string {
	if m != nil && m.AppName != nil {
		return *m.AppName
	}
	return ""
}

func (m *Application) GetDistribution() Distribution {
	if m != nil && m.Distribution != nil {
		return *m.Distribution
	}
	return Default_Application_Distribution
}

func (m *Application) GetComponentType() ComponentType {
	if m != nil && m.ComponentType != nil {
		return *m.ComponentType
	}
	return Default_Application_ComponentType
}

func (m *Application) GetLogDir() string {
	if m != nil && m.LogDir != nil {
		return *m.LogDir
	}
	return ""
}

func (m *Application) GetWorkDir() string {
	if m != nil && m.WorkDir != nil {
		return *m.WorkDir
	}
	return ""
}

func (m *Application) GetConfDir() string {
	if m != nil && m.ConfDir != nil {
		return *m.ConfDir
	}
	return ""
}

func (m *Application) GetSla() *ApplicationSLA {
	if m != nil {
		return m.Sla
	}
	return nil
}

//
// ScheduledAppComponent a structure to describe an application
// component that has been scheduled for deployment.
type ScheduledApp struct {
	// Id the id of the scheduled component (for retrieval from persistence medium for example)
	Id *string `protobuf:"bytes,1,req,name=id" json:"id,omitempty"`
	// Name the name of the component
	Name *string `protobuf:"bytes,2,req,name=name" json:"name,omitempty"`
	// AppName the name of the app this component belongs to
	AppName *string `protobuf:"bytes,3,req,name=app_name" json:"app_name,omitempty"`
	// Component the full component that has been scheduled
	App *Application `protobuf:"bytes,4,req,name=app" json:"app,omitempty"`
	// Position the full position of this item in the queue
	Position *int32 `protobuf:"varint,5,req,name=position" json:"position,omitempty"`
	// Since the timestamp in nanoseconds when this item was added to the queue
	Since            *int64 `protobuf:"varint,6,req,name=since" json:"since,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *ScheduledApp) Reset()         { *m = ScheduledApp{} }
func (m *ScheduledApp) String() string { return proto.CompactTextString(m) }
func (*ScheduledApp) ProtoMessage()    {}

func (m *ScheduledApp) GetId() string {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return ""
}

func (m *ScheduledApp) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *ScheduledApp) GetAppName() string {
	if m != nil && m.AppName != nil {
		return *m.AppName
	}
	return ""
}

func (m *ScheduledApp) GetApp() *Application {
	if m != nil {
		return m.App
	}
	return nil
}

func (m *ScheduledApp) GetPosition() int32 {
	if m != nil && m.Position != nil {
		return *m.Position
	}
	return 0
}

func (m *ScheduledApp) GetSince() int64 {
	if m != nil && m.Since != nil {
		return *m.Since
	}
	return 0
}

//
// HealthCheck describes a health check for an application.
// For the TCP strategy it will just try to connect to the port
// For the HTTP strategy it will make a request and expect a 200 OK status
// For the METRICS strategy it will use the HTTP strategy but additionally
// the response body will be validated that all components are running fine.
type HealthCheck struct {
	// The strategy to use for the health check
	Mode *HealthCheckMode `protobuf:"varint,1,req,name=mode,enum=protocol.HealthCheckMode,def=0" json:"mode,omitempty"`
	// The delay in milliseconds to delay the initial health check after entering running state
	RampUp *int64 `protobuf:"varint,2,req,name=ramp_up" json:"ramp_up,omitempty"`
	// The interval in milliseconds at which to perform health checks
	IntervalMillis *int64 `protobuf:"varint,3,req,name=interval_millis" json:"interval_millis,omitempty"`
	// How quick should a health check return to be successful (in millis)
	Timeout *int64 `protobuf:"varint,5,req,name=timeout" json:"timeout,omitempty"`
	// when this is a http health check it will use this path to make the request
	Path *string `protobuf:"bytes,20,opt,name=path,def=/api/api-docs" json:"path,omitempty"`
	// for a http health check it will use this, other possible value is http
	Scheme           *string `protobuf:"bytes,21,opt,name=scheme,def=http" json:"scheme,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *HealthCheck) Reset()         { *m = HealthCheck{} }
func (m *HealthCheck) String() string { return proto.CompactTextString(m) }
func (*HealthCheck) ProtoMessage()    {}

const Default_HealthCheck_Mode HealthCheckMode = HealthCheckMode_HTTP
const Default_HealthCheck_Path string = "/api/api-docs"
const Default_HealthCheck_Scheme string = "http"

func (m *HealthCheck) GetMode() HealthCheckMode {
	if m != nil && m.Mode != nil {
		return *m.Mode
	}
	return Default_HealthCheck_Mode
}

func (m *HealthCheck) GetRampUp() int64 {
	if m != nil && m.RampUp != nil {
		return *m.RampUp
	}
	return 0
}

func (m *HealthCheck) GetIntervalMillis() int64 {
	if m != nil && m.IntervalMillis != nil {
		return *m.IntervalMillis
	}
	return 0
}

func (m *HealthCheck) GetTimeout() int64 {
	if m != nil && m.Timeout != nil {
		return *m.Timeout
	}
	return 0
}

func (m *HealthCheck) GetPath() string {
	if m != nil && m.Path != nil {
		return *m.Path
	}
	return Default_HealthCheck_Path
}

func (m *HealthCheck) GetScheme() string {
	if m != nil && m.Scheme != nil {
		return *m.Scheme
	}
	return Default_HealthCheck_Scheme
}

//
// ApplicationSLA an application SLA describes what makes a service healthy
// It is used to enforce how many instance of an application should be running
// it also defines the health check used and how many health checks should fail
//
type ApplicationSLA struct {
	// The minimum number of instances that needs to be deployed
	MinInstances *int32 `protobuf:"varint,1,req,name=min_instances,def=1" json:"min_instances,omitempty"`
	// The maximum number of instances that is deployed
	MaxInstances *int32 `protobuf:"varint,2,req,name=max_instances,def=1" json:"max_instances,omitempty"`
	// The health check to use
	HealthCheck *HealthCheck `protobuf:"bytes,3,req,name=health_check" json:"health_check,omitempty"`
	// The amount of health checks that have to fail sequentially to be considered unhealthy
	UnhealthyAt      *int32 `protobuf:"varint,4,req,name=unhealthy_at" json:"unhealthy_at,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *ApplicationSLA) Reset()         { *m = ApplicationSLA{} }
func (m *ApplicationSLA) String() string { return proto.CompactTextString(m) }
func (*ApplicationSLA) ProtoMessage()    {}

const Default_ApplicationSLA_MinInstances int32 = 1
const Default_ApplicationSLA_MaxInstances int32 = 1

func (m *ApplicationSLA) GetMinInstances() int32 {
	if m != nil && m.MinInstances != nil {
		return *m.MinInstances
	}
	return Default_ApplicationSLA_MinInstances
}

func (m *ApplicationSLA) GetMaxInstances() int32 {
	if m != nil && m.MaxInstances != nil {
		return *m.MaxInstances
	}
	return Default_ApplicationSLA_MaxInstances
}

func (m *ApplicationSLA) GetHealthCheck() *HealthCheck {
	if m != nil {
		return m.HealthCheck
	}
	return nil
}

func (m *ApplicationSLA) GetUnhealthyAt() int32 {
	if m != nil && m.UnhealthyAt != nil {
		return *m.UnhealthyAt
	}
	return 0
}

func init() {
	proto.RegisterEnum("protocol.AppStatus", AppStatus_name, AppStatus_value)
	proto.RegisterEnum("protocol.ComponentType", ComponentType_name, ComponentType_value)
	proto.RegisterEnum("protocol.Distribution", Distribution_name, Distribution_value)
	proto.RegisterEnum("protocol.HealthCheckMode", HealthCheckMode_name, HealthCheckMode_value)
}
