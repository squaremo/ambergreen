package metcdstore

import (
	"errors"
	"log"

	wackygrpc "github.com/coreos/etcd/Godeps/_workspace/src/google.golang.org/grpc"
	"github.com/weaveworks/flux/common/data"
	"github.com/weaveworks/flux/common/store"
	"github.com/weaveworks/mesh/metcd"
)

func New(minPeerCount int, logger *log.Logger) store.ServiceDefiner {
	return &metcdStore{
		server: metcd.NewDefaultServer(minPeerCount, logger),
	}
}

type metcdStore struct {
	server *wackygrpc.Server
}

func (s *metcdStore) CheckRegisteredService(serviceName string) error {
	return errors.New("not implemented")
}

func (s *metcdStore) AddService(name string, service data.Service) error {
	return errors.New("not implemented")
}

func (s *metcdStore) RemoveService(serviceName string) error {
	return errors.New("not implemented")
}

func (s *metcdStore) RemoveAllServices() error {
	return errors.New("not implemented")
}

func (s *metcdStore) SetContainerRule(serviceName string, ruleName string, spec data.ContainerRule) error {
	return errors.New("not implemented")
}

func (s *metcdStore) RemoveContainerRule(serviceName string, ruleName string) error {
	return errors.New("not implemented")
}
