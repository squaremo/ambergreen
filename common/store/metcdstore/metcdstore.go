package metcdstore

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/weaveworks/mesh/metcd"
	"golang.org/x/net/context"

	"github.com/weaveworks/flux/common/store"
)

// New returns a ServiceDefiner backed by a default metcd server.
func New(ctx context.Context, minPeerCount int, logger *log.Logger) store.ServiceDefiner {
	// TODO(pb): don't ignore lifecycle management
	terminatec := make(chan struct{})
	terminatedc := make(chan error)
	return &metcdStore{
		ctx:    ctx,
		server: metcd.NewDefaultServer(minPeerCount, terminatec, terminatedc, logger),
	}
}

type metcdStore struct {
	ctx    context.Context
	server metcd.Server
}

var (
	// ErrNotFound indicates an entity was not found.
	ErrNotFound = errors.New("not found")
)

func (s *metcdStore) CheckRegisteredService(serviceName string) error {
	resp, err := s.server.Range(s.ctx, &etcdserverpb.RangeRequest{
		Key: []byte(serviceRootKey(serviceName)),
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
