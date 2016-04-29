package metcdstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/weaveworks/mesh"
	"github.com/weaveworks/mesh/metcd"
	"golang.org/x/net/context"

	"github.com/weaveworks/flux/common/daemon"
	"github.com/weaveworks/flux/common/store"
)

// New returns a Store backed by a default metcd server.
func New(ctx context.Context, minPeerCount int, logger mesh.Logger) store.Store {
	// TODO(pb): don't ignore lifecycle management
	terminatec := make(chan struct{})
	terminatedc := make(chan error)
	return &metcdStore{
		ctx:    ctx,
		server: metcd.NewDefaultServer(minPeerCount, terminatec, terminatedc, logger),
		logger: logger,
	}
}

type metcdStore struct {
	ctx    context.Context
	server metcd.Server
	logger mesh.Logger
}

// ErrNotFound indicates an entity was not found.
var ErrNotFound = errors.New("not found")

// The store.Cluster methods provide an interface for clients to register (via
// Heartbeat) and deregister hosts. Hosts are logical collections of entities, a
// convenient abstraction for when a node dies and takes many things with it.

func (s *metcdStore) GetHosts() ([]*store.Host, error) {
	key := []byte(hostRoot)
	resp, err := s.server.Range(s.ctx, &etcdserverpb.RangeRequest{
		Key:      key,
		RangeEnd: metcd.PrefixRangeEnd(key),
	})
	if err != nil {
		return nil, err
	}
	hosts := make([]*store.Host, len(resp.Kvs))
	for i, kv := range resp.Kvs {
		var host store.Host
		if err = json.Unmarshal(kv.Value, &host); err != nil {
			return nil, err
		}
		hosts[i] = &host
	}
	return hosts, nil
}

func (s *metcdStore) Heartbeat(ttl time.Duration) error {
	// TODO(pb): metcd must support LeaseServer
	// TODO(pb): see Heartbeat in etcdstore
	return errors.New("not implemented")
}

func (s *metcdStore) EndSession() error {
	return errors.New("not implemented")
}

func (s *metcdStore) RegisterHost(identity string, details *store.Host) error {
	_, err := s.server.Put(s.ctx, &etcdserverpb.PutRequest{
		Key: hostKey(identity),
		// Value: sessionHost{...} // TODO(pb)
	})
	return err
}

func (s *metcdStore) DeregisterHost(identity string) error {
	_, err := s.server.DeleteRange(s.ctx, &etcdserverpb.DeleteRangeRequest{
		Key: hostKey(identity),
	})
	return err
}

func (s *metcdStore) WatchHosts(ctx context.Context, changes chan<- store.HostChange, errs daemon.ErrorSink) {
	// TODO(pb): metcd must support WatchServer
}

func (s *metcdStore) Ping() error {
	return errors.New("not implemented") // TODO(pb)
}

func (s *metcdStore) CheckRegisteredService(serviceName string) error {
	key := serviceRootKey(serviceName)
	resp, err := s.server.Range(s.ctx, &etcdserverpb.RangeRequest{
		Key:      key,
		RangeEnd: metcd.PrefixRangeEnd(key),
	})
	if err != nil {
		return err
	}
	if len(resp.Kvs) <= 0 {
		return ErrNotFound
	}
	return nil
}

func (s *metcdStore) AddService(name string, service store.Service) error {
	buf, err := json.Marshal(service)
	if err != nil {
		return err
	}
	_, err = s.server.Put(s.ctx, &etcdserverpb.PutRequest{
		Key:   serviceKey(name),
		Value: buf,
	})
	return err
}

func (s *metcdStore) RemoveService(serviceName string) error {
	key := serviceRootKey(serviceName)
	resp, err := s.server.DeleteRange(s.ctx, &etcdserverpb.DeleteRangeRequest{
		Key:      key,
		RangeEnd: metcd.PrefixRangeEnd(key),
	})
	if err != nil {
		return err
	}
	if resp.Deleted <= 0 {
		return ErrNotFound
	}
	return nil
}

func (s *metcdStore) RemoveAllServices() error {
	_, err := s.server.DeleteRange(s.ctx, &etcdserverpb.DeleteRangeRequest{
		Key:      []byte(serviceRoot),
		RangeEnd: metcd.PrefixRangeEnd([]byte(serviceRoot)),
	})
	return err
}

func (s *metcdStore) SetContainerRule(serviceName string, ruleName string, spec store.ContainerRule) error {
	buf, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	_, err = s.server.Put(s.ctx, &etcdserverpb.PutRequest{
		Key:   ruleKey(serviceName, ruleName),
		Value: buf,
	})
	return err
}

func (s *metcdStore) RemoveContainerRule(serviceName string, ruleName string) error {
	_, err := s.server.DeleteRange(s.ctx, &etcdserverpb.DeleteRangeRequest{
		Key: ruleKey(serviceName, ruleName),
	})
	return err // ignore number of deleted entries
}

func (s *metcdStore) AddInstance(serviceName, instanceName string, instance store.Instance) error {
	buf, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	_, err = s.server.Put(s.ctx, &etcdserverpb.PutRequest{
		Key:   instanceKey(serviceName, instanceName),
		Value: buf,
	})
	return err
}

func (s *metcdStore) RemoveInstance(serviceName, instanceName string) error {
	resp, err := s.server.DeleteRange(s.ctx, &etcdserverpb.DeleteRangeRequest{
		Key: instanceKey(serviceName, instanceName),
	})
	if err != nil {
		return err
	}
	if resp.Deleted <= 0 {
		return ErrNotFound
	}
	return nil
}

func (s *metcdStore) GetService(serviceName string, opts store.QueryServiceOptions) (*store.ServiceInfo, error) {
	key := serviceRootKey(serviceName)
	resp, err := s.server.Range(s.ctx, &etcdserverpb.RangeRequest{
		Key:      key,
		RangeEnd: metcd.PrefixRangeEnd(key),
	})
	if err != nil {
		return nil, err
	}
	p := newServiceParser(resp, opts)
	if !p.next() {
		return nil, fmt.Errorf("failed to parse a service: %v", p.err())
	}
	_, service := p.service()
	return &service, nil
}

func (s *metcdStore) GetAllServices(opts store.QueryServiceOptions) (map[string]*store.ServiceInfo, error) {
	key := []byte(serviceRoot)
	resp, err := s.server.Range(s.ctx, &etcdserverpb.RangeRequest{
		Key:      key,
		RangeEnd: metcd.PrefixRangeEnd(key),
	})
	if err != nil {
		return nil, err
	}
	services := map[string]*store.ServiceInfo{}
	p := newServiceParser(resp, opts)
	for p.next() {
		name, service := p.service()
		services[name] = &service // TODO(pb)
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return services, nil
}

func (s *metcdStore) WatchServices(ctx context.Context, resCh chan<- store.ServiceChange, errorSink daemon.ErrorSink, opts store.QueryServiceOptions) {
	return // TODO(pb)
}
