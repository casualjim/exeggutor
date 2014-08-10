package caprica

import (
	"strconv"
	"time"
)

type EpochTime time.Time

func (t *EpochTime) UnmarshalText(b []byte) error {
	result, err := strconv.ParseInt(string(b), 0, 64)
	if err != nil {
		return err
	}
	*t = EpochTime(time.Unix(result/1000, 0))
	return nil
}

func (t *EpochTime) UnmarshalJSON(b []byte) error {
	result, err := strconv.ParseInt(string(b), 0, 64)
	if err != nil {
		return err
	}
	*t = EpochTime(time.Unix(result/1000, 0))
	return nil
}

// CapricaResponse represents a response received from caprica
type CapricaResponse struct {
	ErrorCode    int       `json:"errorCode"`
	ErrorMessage string    `json:"errorMessage"`
	Success      bool      `json:"success"`
	Timestamp    EpochTime `json:"ts"`
}

// A response that lists caprica services
type CapricaInstancesResponse struct {
	CapricaResponse
	Result []CapricaService `json:"result"`
}

// CapricaService is a service found in a caprica
type CapricaService struct {
	BuildNumber     string    `json:"buildNumber"`
	Cluster         string    `json:"cluster"`
	CustomTags      []string  `json:"customTags"`
	Instance        *Instance `json:"instance"`
	Platform        string    `json:"platform"`
	ServiceFunction string    `json:"serviceFunction,omitempty"`
	Name            string    `json:"serviceName"`
	Zone            string    `json:"zone"`
	CreationDate    string    `json:"creationDate"`
}

// IamInstanceProfile representes the instance profile information
type IamInstanceProfile struct {
	ARN string `json:"arn"`
	ID  string `json:"id"`
}

// Instance encapsulates a running instance in EC2.
//
// See http://goo.gl/OCH8a for more details.
type Instance struct {
	InstanceID         string              `json:"instanceId"`
	InstanceType       string              `json:"instanceType"`
	ImageID            string              `json:"imageId"`
	PrivateDNSName     string              `json:"privateDnsName"`
	DNSName            string              `json:"publicDnsName"`
	KeyName            string              `json:"keyName"`
	AMILaunchIndex     int                 `json:"amiLaunchIndex"`
	State              *InstanceState      `json:"state"`
	Tags               []Tag               `json:"tags"`
	VpcID              string              `json:"vpcId"`
	SubnetID           string              `json:"subnetId"`
	IamInstanceProfile *IamInstanceProfile `json:"iamInstanceProfile"`
	PrivateIPAddress   string              `json:"privateIpAddress"`
	PublicIPAddress    string              `json:"publicIpAddress"`
	Architecture       string              `json:"architecture"`
	LaunchTime         EpochTime           `json:"launchTime"`
	SecurityGroups     []SecurityGroup     `json:"securityGroups"`
	RootDeviceName     string              `json:"rootDeviceName"`
	RootDeviceType     string              `json:"rootDeviceType"`
	EBSOptimized       bool                `json:"ebsOptimized"`
	VirtualizationType string              `json:"virtualizationType"`
}

// InstanceState encapsulates the state of an instance in EC2.
//
// See http://goo.gl/y3ZBq for more details.
type InstanceState struct {
	Code int    `json:"code"` // Watch out, bits 15-8 have unpublished meaning.
	Name string `json:"name"`
}

// Tag represents key-value metadata used to classify and organize
// EC2 instances.
//
// See http://goo.gl/bncl3 for more details
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SecurityGroup represents an EC2 security group.
// If SecurityGroup is used as a parameter, then one of Id or Name
// may be empty. If both are set, then Id is used.
type SecurityGroup struct {
	ID          string `json:"groupId"`
	Name        string `json:"groupName"`
	Description string `json:"groupDescription,omitempty"`
	VpcID       string `json:"vpcId,omitempty"`
}
