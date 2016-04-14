package metcdstore

import (
	"encoding/json"
	"errors"
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

func (s *metcdStore) GetHosts() ([]*store.Host, error) {
	return nil, errors.New("not implemented") // TODO(pb)
}

func (s *metcdStore) Heartbeat(identity string, ttl time.Duration, state *store.Host) error {
	return errors.New("not implemented") // TODO(pb)
}

func (s *metcdStore) DeregisterHost(identity string) error {
	return errors.New("not implemented") // TODO(pb)
}

func (s *metcdStore) WatchHosts(ctx context.Context, changes chan<- store.HostChange, errs daemon.ErrorSink) {
	return // TODO(pb)
}

func (s *metcdStore) Ping() error {
	return errors.New("not implemented") // TODO(pb)
}

func (s *metcdStore) CheckRegisteredService(serviceName string) error {
	key := []byte(serviceRootKey(serviceName))
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
		Key:   []byte(serviceKey(name)),
		Value: buf,
	})
	return err
}

func (s *metcdStore) RemoveService(serviceName string) error {
	key := []byte(serviceRootKey(serviceName))
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
		Key:   []byte(ruleKey(serviceName, ruleName)),
		Value: buf,
	})
	return err
}

func (s *metcdStore) RemoveContainerRule(serviceName string, ruleName string) error {
	_, err := s.server.DeleteRange(s.ctx, &etcdserverpb.DeleteRangeRequest{
		Key: []byte(ruleKey(serviceName, ruleName)),
	})
	return err // ignore number of deleted entries
}

func (s *metcdStore) AddInstance(serviceName, instanceName string, instance store.Instance) error {
	buf, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	_, err = s.server.Put(s.ctx, &etcdserverpb.PutRequest{
		Key:   []byte(instanceKey(serviceName, instanceName)),
		Value: buf,
	})
	return err
}

func (s *metcdStore) RemoveInstance(serviceName, instanceName string) error {
	resp, err := s.server.DeleteRange(s.ctx, &etcdserverpb.DeleteRangeRequest{
		Key: []byte(instanceKey(serviceName, instanceName)),
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
	return nil, errors.New("not implemented") // TODO(pb)
}

func (s *metcdStore) GetAllServices(opts store.QueryServiceOptions) ([]*store.ServiceInfo, error) {
	return nil, errors.New("not implemented") // TODO(pb)
}

func (s *metcdStore) WatchServices(ctx context.Context, resCh chan<- store.ServiceChange, errorSink daemon.ErrorSink, opts store.QueryServiceOptions) {
	return // TODO(pb)
}
