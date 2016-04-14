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

// New returns a ServiceDefiner backed by a default metcd server.
func New(ctx context.Context, minPeerCount int, logger mesh.Logger) store.ServiceDefiner {
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

// TODO(pb): remove eventually; just for development
var (
	_ store.Cluster         = &metcdStore{}
	_ store.Pinger          = &metcdStore{}
	_ store.ServiceDefiner  = &metcdStore{}
	_ store.InstanceDefiner = &metcdStore{}
	_ store.ServiceQueryer  = &metcdStore{}
	_ store.Store           = &metcdStore{}
)

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
	prefix := []byte(serviceRootKey(serviceName))
	resp, err := s.server.Range(s.ctx, &etcdserverpb.RangeRequest{
		Key:      prefix,
		RangeEnd: metcd.PrefixRangeEnd(prefix),
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
	_, err := s.server.DeleteRange(s.ctx, &etcdserverpb.DeleteRangeRequest{
		Key: []byte(serviceRootKey(serviceName)),
	})
	return err // ignore number of deleted entries
}

func (s *metcdStore) RemoveAllServices() error {
	_, err := s.server.DeleteRange(s.ctx, &etcdserverpb.DeleteRangeRequest{
		Key: []byte(serviceRoot),
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

func (s *metcdStore) AddInstance(serviceName, instanceName string, details store.Instance) error {
	return errors.New("not implemented") // TODO(pb)
}

func (s *metcdStore) RemoveInstance(serviceName, instanceName string) error {
	return errors.New("not implemented") // TODO(pb)
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
