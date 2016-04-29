package metcdstore

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"

	"github.com/weaveworks/flux/common/store"
)

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
	p.name = ""
	p.curr = store.ServiceInfo{
		Instances:      map[string]store.Instance{},
		ContainerRules: map[string]store.ContainerRule{},
	}
	for ; p.index < len(p.resp.Kvs); p.index++ {
		// Assumes resp.Kvs order is service key, then other keys.
		if serviceName, ok := parseServiceKey(p.resp.Kvs[p.index].Key); ok {
			if p.name != "" { // we already parsed a service key, so this is a new one
				return true // yield the ServiceInfo to the caller
			}
			p.name = serviceName
			if err := json.Unmarshal(p.resp.Kvs[p.index].Value, &p.curr.Service); err != nil {
				p.er = err
				return false
			}
		} else if serviceName, instanceName, ok := parseInstanceKey(p.resp.Kvs[p.index].Key); ok {
			if p.name != serviceName {
				p.er = fmt.Errorf("inconsistent service names: %q, %q", p.name, serviceName)
				return false
			}
			if p.opts.WithInstances {
				var instance store.Instance
				if err := json.Unmarshal(p.resp.Kvs[p.index].Value, &instance); err != nil {
					p.er = err
					return false
				}
				p.curr.Instances[instanceName] = instance
			}
		} else if serviceName, containerRuleName, ok := parseContainerRuleKey(p.resp.Kvs[p.index].Key); ok {
			if p.name != serviceName {
				p.er = fmt.Errorf("inconsistent service names: %q, %q", p.name, serviceName)
				return false
			}
			if p.opts.WithContainerRules {
				var containerRule store.ContainerRule
				if err := json.Unmarshal(p.resp.Kvs[p.index].Value, &containerRule); err != nil {
					p.er = err
					return false
				}
				p.curr.ContainerRules[containerRuleName] = containerRule
			}
		} else {
			p.er = fmt.Errorf("unknown key %q", p.resp.Kvs[p.index].Key)
			return false
		}
	}
	// Special case: we just had one service
	if p.name != "" {
		return true
	}
	// Regular case: no more to parse
	return false
}

func (p *serviceParser) service() (string, store.ServiceInfo) {
	return p.name, p.curr
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
