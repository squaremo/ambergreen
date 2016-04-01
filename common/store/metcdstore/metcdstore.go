package metcdstore

import (
	"errors"
	"log"

	"github.com/weaveworks/mesh/metcd"
	"golang.org/x/net/context"

	"github.com/weaveworks/flux/common/data"
	"github.com/weaveworks/flux/common/store"
)

func New(ctx context.Context, minPeerCount int, logger *log.Logger) store.ServiceDefiner {
	return &metcdStore{
		ctx:    ctx,
		server: metcd.NewDefaultServer(minPeerCount, logger),
	}
}

type metcdStore struct {
	ctx    context.Context
	server metcd.Server
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
