package etcdstore

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	etcd_errors "github.com/coreos/etcd/error"
	"github.com/coreos/go-etcd/etcd"

	"github.com/squaremo/flux/common/daemon"
	"github.com/squaremo/flux/common/data"
	"github.com/squaremo/flux/common/store"
)

type etcdStore struct {
	client *etcd.Client
}

func NewFromEnv() store.Store {
	etcd_address := os.Getenv("ETCD_PORT")
	if etcd_address == "" {
		etcd_address = os.Getenv("ETCD_ADDRESS")
	}
	if strings.HasPrefix(etcd_address, "tcp:") {
		etcd_address = "http:" + etcd_address[4:]
	}
	if etcd_address == "" {
		etcd_address = "http://127.0.0.1:4001"
	}

	return New(etcd_address)
}

func New(addr string) store.Store {
	return &etcdStore{client: etcd.NewClient([]string{addr})}
}

// Check if we can talk to etcd
func (es *etcdStore) Ping() error {
	rr := etcd.NewRawRequest("GET", "version", nil, nil)
	_, err := es.client.SendRequest(rr)
	return err
}

const ROOT = "/weave/service/"

func serviceRootKey(serviceName string) string {
	return ROOT + serviceName
}

func serviceKey(serviceName string) string {
	return fmt.Sprintf("%s%s/details", ROOT, serviceName)
}

func groupSpecKey(serviceName, groupName string) string {
	return fmt.Sprintf("%s%s/groupspec/%s", ROOT, serviceName, groupName)
}

func instanceKey(serviceName, instanceName string) string {
	return fmt.Sprintf("%s%s/instance/%s", ROOT, serviceName, instanceName)
}

type parsedRootKey struct {
}

type parsedServiceRootKey struct {
	serviceName string
}

type parsedServiceKey struct {
	serviceName string
}

type parsedGroupSpecKey struct {
	serviceName string
	groupName   string
}

func (k parsedGroupSpecKey) relevantTo(opts store.WatchServicesOptions) (bool, string) {
	return opts.WithGroupSpecChanges, k.serviceName
}

type parsedInstanceKey struct {
	serviceName  string
	instanceName string
}

func (k parsedInstanceKey) relevantTo(opts store.WatchServicesOptions) (bool, string) {
	return opts.WithInstanceChanges, k.serviceName
}

// Parse a path to find its type

func parseKey(key string) interface{} {
	if len(key) <= len(ROOT) {
		return parsedRootKey{}
	}

	p := strings.Split(key[len(ROOT):], "/")
	if len(p) == 1 {
		return parsedServiceRootKey{p[0]}
	}

	switch p[1] {
	case "details":
		return parsedServiceKey{p[0]}

	case "groupspec":
		if len(p) == 3 {
			return parsedGroupSpecKey{p[0], p[2]}
		}

	case "instance":
		if len(p) == 3 {
			return parsedInstanceKey{p[0], p[2]}
		}
	}

	return nil
}

func (es *etcdStore) CheckRegisteredService(serviceName string) error {
	_, err := es.client.Get(serviceRootKey(serviceName), false, false)
	return err
}

func (es *etcdStore) AddService(name string, details data.Service) error {
	json, err := json.Marshal(&details)
	if err != nil {
		return fmt.Errorf("Failed to encode: %s", err)
	}
	_, err = es.client.Set(serviceKey(name), string(json), 0)
	return err
}

func (es *etcdStore) RemoveService(serviceName string) error {
	_, err := es.client.Delete(serviceRootKey(serviceName), true)
	return err
}

func (es *etcdStore) RemoveAllServices() error {
	_, err := es.client.Delete(ROOT, true)
	return err
}

func (es *etcdStore) GetService(serviceName string, opts store.QueryServiceOptions) (store.ServiceInfo, error) {
	var svc store.ServiceInfo
	if opts.WithInstances {
		svc.Instances = make([]store.InstanceInfo, 0)
	}
	if opts.WithGroupSpecs {
		svc.ContainerGroupSpecs = make([]store.ContainerGroupSpecInfo, 0)
	}
	return svc, es.traverse(serviceRootKey(serviceName), visitService(&svc, opts))
}

func (es *etcdStore) GetAllServices(opts store.QueryServiceOptions) ([]store.ServiceInfo, error) {
	svcs := make([]store.ServiceInfo, 0)
	err := es.traverse(ROOT, func(node *etcd.Node) error {
		if isServiceRoot(node) {
			var svc store.ServiceInfo
			err := traverse(node, visitService(&svc, opts))
			if err != nil {
				return err
			}
			svcs = append(svcs, svc)
			return nil
		}
		return nil
	})
	return svcs, err
}

// ===== helpers

func isServiceRoot(node *etcd.Node) bool {
	if !node.Dir {
		return false
	}
	switch parseKey(node.Key).(type) {
	case parsedServiceRootKey:
		return true
	}
	return false
}

type visitor func(*etcd.Node) error

func visitService(svc *store.ServiceInfo, opts store.QueryServiceOptions) visitor {
	return func(node *etcd.Node) error {
		if node.Dir {
			return traverse(node, visitService(svc, opts))
		}

		switch key := parseKey(node.Key).(type) {
		case parsedServiceKey:
			svc.Name = key.serviceName
			err := unmarshalIntoService(&svc.Service, node)
			if err != nil {
				return err
			}
		case parsedInstanceKey:
			if opts.WithInstances {
				inst, err := unmarshalInstance(key.instanceName, node)
				if err != nil {
					return err
				}
				svc.Instances = append(svc.Instances, store.InstanceInfo{
					Name:     key.instanceName,
					Instance: inst,
				})
			}
		case parsedGroupSpecKey:
			if opts.WithGroupSpecs {
				spec, err := unmarshalGroupSpec(key.groupName, node)
				if err != nil {
					return err
				}
				svc.ContainerGroupSpecs = append(svc.ContainerGroupSpecs, store.ContainerGroupSpecInfo{
					Name:               key.groupName,
					ContainerGroupSpec: spec,
				})
			}
		}
		return nil
	}
}

func unmarshalIntoService(svc *data.Service, node *etcd.Node) error {
	return json.Unmarshal([]byte(node.Value), &svc)
}

func unmarshalGroupSpec(name string, node *etcd.Node) (data.ContainerGroupSpec, error) {
	var gs data.ContainerGroupSpec
	return gs, json.Unmarshal([]byte(node.Value), &gs)
}

func unmarshalInstance(name string, node *etcd.Node) (data.Instance, error) {
	var instance data.Instance
	return instance, json.Unmarshal([]byte(node.Value), &instance)
}

func traverse(node *etcd.Node, visit visitor) error {
	for _, child := range node.Nodes {
		if err := visit(child); err != nil {
			return err
		}
	}
	return nil
}

func (es *etcdStore) traverse(key string, visit visitor) error {
	r, err := es.client.Get(key, false, true)
	if err != nil {
		if etcderr, ok := err.(*etcd.EtcdError); ok && etcderr.ErrorCode == etcd_errors.EcodeKeyNotFound {
			return nil
		}
		return err
	}

	return traverse(r.Node, visit)
}

func (es *etcdStore) SetContainerGroupSpec(serviceName string, groupName string, spec data.ContainerGroupSpec) error {
	json, err := json.Marshal(spec)
	if err != nil {
		return err
	}

	if _, err := es.client.Set(groupSpecKey(serviceName, groupName), string(json), 0); err != nil {
		return err
	}

	return err
}

func (es *etcdStore) RemoveContainerGroupSpec(serviceName string, groupName string) error {
	_, err := es.client.Delete(groupSpecKey(serviceName, groupName), true)
	return err
}

func (es *etcdStore) AddInstance(serviceName string, instanceName string, details data.Instance) error {
	json, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("Failed to encode: %s", err)
	}
	if _, err := es.client.Set(instanceKey(serviceName, instanceName), string(json), 0); err != nil {
		return fmt.Errorf("Unable to write: %s", err)
	}
	return nil
}

func (es *etcdStore) RemoveInstance(serviceName, instanceName string) error {
	_, err := es.client.Delete(instanceKey(serviceName, instanceName), true)
	return err
}

func (es *etcdStore) WatchServices(resCh chan<- data.ServiceChange, stopCh <-chan struct{}, errorSink daemon.ErrorSink, opts store.WatchServicesOptions) {
	etcdCh := make(chan *etcd.Response, 1)
	watchStopCh := make(chan bool, 1)
	go func() {
		_, err := es.client.Watch(ROOT, 0, true, etcdCh, nil)
		if err != nil {
			errorSink.Post(err)
		}
	}()

	svcs := make(map[string]struct{})
	store.ForeachServiceInstance(es, func(name string, svc data.Service) error {
		svcs[name] = struct{}{}
		return nil
	}, nil)

	handleResponse := func(r *etcd.Response) {
		// r is nil on error
		if r == nil {
			return
		}

		switch r.Action {
		case "delete":
			switch key := parseKey(r.Node.Key).(type) {
			case parsedRootKey:
				for name := range svcs {
					resCh <- data.ServiceChange{name, true}
				}
				svcs = make(map[string]struct{})

			case parsedServiceRootKey:
				delete(svcs, key.serviceName)
				resCh <- data.ServiceChange{key.serviceName, true}

			case interface {
				relevantTo(opts store.WatchServicesOptions) (bool, string)
			}:
				if relevant, service := key.relevantTo(opts); relevant {
					resCh <- data.ServiceChange{service, false}
				}
			}

		case "set":
			switch key := parseKey(r.Node.Key).(type) {
			case parsedServiceKey:
				svcs[key.serviceName] = struct{}{}
				resCh <- data.ServiceChange{key.serviceName, false}

			case interface {
				relevantTo(opts store.WatchServicesOptions) (bool, string)
			}:
				if relevant, service := key.relevantTo(opts); relevant {
					resCh <- data.ServiceChange{service, false}
				}
			}
		}
	}

	go func() {
		for {
			select {
			case <-stopCh:
				watchStopCh <- true
				return

			case r := <-etcdCh:
				handleResponse(r)
			}
		}
	}()
}
