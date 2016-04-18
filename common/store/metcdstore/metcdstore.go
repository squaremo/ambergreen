package metcdstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
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
	key := []byte(serviceRootKey(serviceName))
	resp, err := s.server.Range(s.ctx, &etcdserverpb.RangeRequest{
		Key:      key,
		RangeEnd: metcd.PrefixRangeEnd(key),
	})
	if err != nil {
		return nil, err
	}
	return serviceInfo(resp, opts)
}

func (s *metcdStore) GetAllServices(opts store.QueryServiceOptions) ([]*store.ServiceInfo, error) {
	key := []byte(serviceRoot)
	resp, err := s.server.Range(s.ctx, &etcdserverpb.RangeRequest{
		Key:      key,
		RangeEnd: metcd.PrefixRangeEnd(key),
	})
	if err != nil {
		return nil, err
	}
	return serviceInfos(resp, opts)
}

func serviceInfo(resp *etcdserverpb.RangeResponse, opts store.QueryServiceOptions) (*store.ServiceInfo, error) {
	p := newServiceParser(resp, opts)
	if !p.next() {
		return nil, fmt.Errorf("failed to parse a service: %v", p.err())
	}
	service := p.service()
	return &service, nil
}

func serviceInfos(resp *etcdserverpb.RangeResponse, opts store.QueryServiceOptions) ([]*store.ServiceInfo, error) {
	var services []*store.ServiceInfo
	p := newServiceParser(resp, opts)
	for p.next() {
		service := p.service()
		services = append(services, &service)
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return services, nil
}

// In the style of bufio.Scanner.
type serviceParser struct {
	resp  *etcdserverpb.RangeResponse
	opts  store.QueryServiceOptions
	index int
	name  string
	curr  store.ServiceInfo
	er    error
}

func newServiceParser(resp *etcdserverpb.RangeResponse, opts store.QueryServiceOptions) *serviceParser {
	return &serviceParser{
		resp: resp,
		opts: opts,
	}
}

func (p *serviceParser) next() bool {
	// We enter this function with p.index pointing at a service key. We leave
	// this function when p.index points at a different service key (or EOF).
	// Assume that KV order is service key, then other related keys.
	p.curr = store.ServiceInfo{}
	for ; p.index < len(p.resp.Kvs); p.index++ {
		// Assumes resp.Kvs order is service key, then other keys.
		if serviceName, ok := parseServiceKey(p.resp.Kvs[p.index].Key); ok {
			if p.curr.Name != "" { // we already parsed a service key, so this is a new one
				return true // yield the ServiceInfo to the caller
			}
			p.curr.Name = serviceName
			if err := json.Unmarshal(p.resp.Kvs[p.index].Value, &p.curr.Service); err != nil {
				p.er = err
				return false
			}
		} else if serviceName, instanceName, ok := parseInstanceKey(p.resp.Kvs[p.index].Key); ok {
			if p.curr.Name != serviceName {
				p.er = fmt.Errorf("inconsistent service names: %q, %q", p.curr.Name, serviceName)
				return false
			}
			if p.opts.WithInstances {
				var instance store.Instance
				if err := json.Unmarshal(p.resp.Kvs[p.index].Value, &instance); err != nil {
					p.er = err
					return false
				}
				p.curr.Instances = append(p.curr.Instances, store.InstanceInfo{
					Name:     instanceName,
					Instance: instance,
				})
			}
		} else if serviceName, containerRuleName, ok := parseContainerRuleKey(p.resp.Kvs[p.index].Key); ok {
			if p.curr.Name != serviceName {
				p.er = fmt.Errorf("inconsistent service names: %q, %q", p.curr.Name, serviceName)
				return false
			}
			if p.opts.WithContainerRules {
				var containerRule store.ContainerRule
				if err := json.Unmarshal(p.resp.Kvs[p.index].Value, &containerRule); err != nil {
					p.er = err
					return false
				}
				p.curr.ContainerRules = append(p.curr.ContainerRules, store.ContainerRuleInfo{
					Name:          containerRuleName,
					ContainerRule: containerRule,
				})
			}
		} else {
			p.er = fmt.Errorf("unknown key %q", p.resp.Kvs[p.index].Key)
			return false
		}
	}
	// Special case: we just had one service
	if p.curr.Name != "" {
		return true
	}
	// Regular case: no more to parse
	return false
}

func (p *serviceParser) service() store.ServiceInfo {
	return p.curr
}

func (p *serviceParser) err() error {
	return p.er
}

const (
	serviceKeyPrefixStr = `^` + serviceRoot + `(?P<serviceName>[^\/]+)/`
	serviceKeyStr       = serviceKeyPrefixStr + `details$`
	instanceKeyStr      = serviceKeyPrefixStr + `instance/(?P<instanceName>[^\/]+)$`
	containerRuleKeyStr = serviceKeyPrefixStr + `groupspec/(?P<containerRuleName>[^\/]+)$`
)

var (
	serviceKeyRegexp       = regexp.MustCompile(serviceKeyStr)
	instanceKeyRegexp      = regexp.MustCompile(instanceKeyStr)
	containerRuleKeyRegexp = regexp.MustCompile(containerRuleKeyStr)
)

func parseServiceKey(key []byte) (serviceName string, ok bool) {
	m := serviceKeyRegexp.FindSubmatch(key)
	if len(m) < 2 {
		return "", false
	}
	return string(m[1]), true
}

func parseInstanceKey(key []byte) (serviceName, instanceName string, ok bool) {
	m := instanceKeyRegexp.FindSubmatch(key)
	if len(m) < 3 {
		return "", "", false
	}
	return string(m[1]), string(m[2]), true
}

func parseContainerRuleKey(key []byte) (serviceName, containerRuleName string, ok bool) {
	m := containerRuleKeyRegexp.FindSubmatch(key)
	if len(m) < 3 {
		return "", "", false
	}
	return string(m[1]), string(m[2]), true
}

func (s *metcdStore) WatchServices(ctx context.Context, resCh chan<- store.ServiceChange, errorSink daemon.ErrorSink, opts store.QueryServiceOptions) {
	return // TODO(pb)
}
