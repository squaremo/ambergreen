package store

import (
	"time"

	"golang.org/x/net/context"

	"github.com/weaveworks/flux/common/daemon"
)

type QueryServiceOptions struct {
	WithInstances      bool
	WithContainerRules bool
}

type InstanceInfo struct {
	Name string `json:"name"`
	Instance
}

type ContainerRuleInfo struct {
	Name string `json:"name"`
	ContainerRule
}

type ServiceInfo struct {
	Name string `json:"name"`
	Service
	Instances      []InstanceInfo      `json:"instances,omitempty"`
	ContainerRules []ContainerRuleInfo `json:"groups,omitempty"`
}

type Store interface {
	Cluster
	Pinger
	ServiceDefiner
	InstanceDefiner
	ServiceQueryer
}

type Cluster interface {
	GetHosts() ([]*Host, error)
	Heartbeat(identity string, ttl time.Duration, state *Host) error
	DeregisterHost(identity string) error
	WatchHosts(ctx context.Context, changes chan<- HostChange, errs daemon.ErrorSink)
}

type Pinger interface {
	Ping() error
}

type ServiceDefiner interface {
	CheckRegisteredService(serviceName string) error
	AddService(name string, service Service) error
	RemoveService(serviceName string) error
	RemoveAllServices() error

	SetContainerRule(serviceName string, ruleName string, spec ContainerRule) error
	RemoveContainerRule(serviceName string, ruleName string) error
}

type InstanceDefiner interface {
	AddInstance(serviceName, instanceName string, details Instance) error
	RemoveInstance(serviceName, instanceName string) error
}

type ServiceQueryer interface {
	GetService(serviceName string, opts QueryServiceOptions) (*ServiceInfo, error)
	GetAllServices(opts QueryServiceOptions) ([]*ServiceInfo, error)
	WatchServices(ctx context.Context, resCh chan<- ServiceChange, errorSink daemon.ErrorSink, opts QueryServiceOptions)
}

// CompositeStore implements Store, and allows different concrete
// implementations to service different parts of the interface.
type CompositeStore struct {
	Cluster
	Pinger
	ServiceDefiner
	InstanceDefiner
	ServiceQueryer
}

var _ Store = CompositeStore{}
