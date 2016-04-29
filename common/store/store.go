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

type ServiceInfo struct {
	Service
	Instances      map[string]Instance
	ContainerRules map[string]ContainerRule
}

type RuntimeStore interface {
	Store
	StartFunc() daemon.StartFunc
}

type Store interface {
	Pinger
	Cluster
	HostDefiner
	HostQueryer
	ServiceDefiner
	ServiceQueryer
	InstanceDefiner
}

type Pinger interface {
	Ping() error
}

type Cluster interface {
	Heartbeat(ttl time.Duration) error
	EndSession() error
}

type HostDefiner interface {
	RegisterHost(identity string, details *Host) error
	DeregisterHost(identity string) error
}

type HostQueryer interface {
	GetHosts() ([]*Host, error)
	WatchHosts(ctx context.Context, resCh chan<- HostChange, errorSink daemon.ErrorSink)
}

type ServiceDefiner interface {
	CheckRegisteredService(serviceName string) error
	AddService(name string, service Service) error
	RemoveService(serviceName string) error
	RemoveAllServices() error

	SetContainerRule(serviceName string, ruleName string, spec ContainerRule) error
	RemoveContainerRule(serviceName string, ruleName string) error
}

type ServiceQueryer interface {
	GetService(serviceName string, opts QueryServiceOptions) (*ServiceInfo, error)
	GetAllServices(opts QueryServiceOptions) (map[string]*ServiceInfo, error)
	WatchServices(ctx context.Context, resCh chan<- ServiceChange, errorSink daemon.ErrorSink, opts QueryServiceOptions)
}

type InstanceDefiner interface {
	AddInstance(serviceName, instanceName string, details Instance) error
	RemoveInstance(serviceName, instanceName string) error
}

// CompositeStore implements Store, and allows different concrete
// implementations to service different parts of the interface.
type CompositeStore struct {
	Pinger
	Cluster
	HostDefiner
	HostQueryer
	ServiceDefiner
	ServiceQueryer
	InstanceDefiner
}

var _ Store = CompositeStore{}
