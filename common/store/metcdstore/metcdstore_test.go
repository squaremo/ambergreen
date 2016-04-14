package metcdstore

import (
	"io/ioutil"
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/weaveworks/flux/common/store"

	"golang.org/x/net/context"
)

func TestServiceKeyRegexp(t *testing.T) {
	for key, want := range map[string]struct {
		serviceName string
		ok          bool
	}{
		"": {"", false},
		"/weave-flux/service/foo":           {"", false},
		"/weave-flux/service/foo/":          {"", false},
		"/weave-flux/service/foo/details":   {"foo", true},
		"/weave-flux/service/foo/instance":  {"", false},
		"/weave-flux/service/foo/groupspec": {"", false},
	} {
		serviceName, ok := parseServiceKey([]byte(key))
		if serviceName != want.serviceName || ok != want.ok {
			t.Errorf("%q: want (%q %v), have (%q %v)", key, want.serviceName, want.ok, serviceName, ok)
		}
	}
}

func TestInstanceKeyRegexp(t *testing.T) {
	for key, want := range map[string]struct {
		serviceName  string
		instanceName string
		ok           bool
	}{
		"": {"", "", false},
		"/weave-flux/service/foo":               {"", "", false},
		"/weave-flux/service/foo/instance/":     {"", "", false},
		"/weave-flux/service/foo/instance/bar":  {"foo", "bar", true},
		"/weave-flux/service/foo/instance/bar/": {"", "", false},
		"/weave-flux/service/foo/groupspec/bar": {"", "", false},
	} {
		serviceName, instanceName, ok := parseInstanceKey([]byte(key))
		if serviceName != want.serviceName || instanceName != want.instanceName || ok != want.ok {
			t.Errorf("%q: want (%q %q %v), have (%q %q %v)", key, want.serviceName, want.instanceName, want.ok, serviceName, instanceName, ok)
		}
	}
}

func TestContainerRuleKeyRegexp(t *testing.T) {
	for key, want := range map[string]struct {
		serviceName       string
		containerRuleName string
		ok                bool
	}{
		"": {"", "", false},
		"/weave-flux/service/foo":                {"", "", false},
		"/weave-flux/service/foo/groupspec/":     {"", "", false},
		"/weave-flux/service/foo/instance/bar":   {"", "", false},
		"/weave-flux/service/foo/groupspec/bar":  {"foo", "bar", true},
		"/weave-flux/service/foo/groupspec/bar/": {"", "", false},
	} {
		serviceName, containerRuleName, ok := parseContainerRuleKey([]byte(key))
		if serviceName != want.serviceName || containerRuleName != want.containerRuleName || ok != want.ok {
			t.Errorf("%q: want (%q %q %v), have (%q %q %v)", key, want.serviceName, want.containerRuleName, want.ok, serviceName, containerRuleName, ok)
		}
	}
}

func TestMetcdStore(t *testing.T) {
	// This is a stopgap solution until metcdstore implements the complete Store
	// interface and we can use the nice test suite.

	logger := log.New(ioutil.Discard, "", log.LstdFlags)
	s := New(context.Background(), 1, logger)
	time.Sleep(3 * time.Second)

	if err := s.CheckRegisteredService("s1"); err == nil {
		t.Fatal("CheckRegisteredService: want error, have none")
	}

	if err := s.RemoveService("s1"); err == nil {
		t.Fatal("RemoveService: want error, have none")
	}

	service := store.Service{
		Address:  "1.2.3.4",
		Port:     1234,
		Protocol: "tcp",
	}

	if err := s.AddService("s1", service); err != nil {
		t.Fatalf("AddService: %v", err)
	}

	if err := s.CheckRegisteredService("s1"); err != nil {
		t.Fatalf("CheckRegisteredService: %v", err)
	}

	if err := s.RemoveInstance("s1", "i1"); err == nil {
		t.Fatal("RemoveInstance: want error, have none")
	}

	instance := store.Instance{
		State:         store.LIVE,
		Host:          store.Host{IPAddress: "10.11.12.13"},
		ContainerRule: "container-rule",
		Address:       "address",
		Port:          123,
		Labels:        map[string]string{"label-key": "label-value"},
	}

	if err := s.AddInstance("s1", "i1", instance); err != nil {
		t.Fatalf("AddInstance: %v", err)
	}

	containerRule := store.ContainerRule{
		Selector: store.Selector{
			"selector-key": "selector-value",
		},
	}

	if err := s.SetContainerRule("s1", "r1", containerRule); err != nil {
		t.Fatalf("SetContainerRule: %v", err)
	}

	serviceInfo, err := s.GetService("s1", store.QueryServiceOptions{
		WithInstances:      true,
		WithContainerRules: true,
	})
	if err != nil {
		t.Fatalf("GetService: %v", err)
	}
	if want, have := "s1", serviceInfo.Name; want != have {
		t.Fatalf("want %q, have %q", want, have)
	}
	if want, have := service, serviceInfo.Service; !reflect.DeepEqual(want, have) {
		t.Fatalf("want %#+v, have %#+v", want, have)
	}
	if want, have := 1, len(serviceInfo.Instances); want != have {
		t.Fatalf("want %d, have %d", want, have)
	}
	if want, have := instance, serviceInfo.Instances[0].Instance; !reflect.DeepEqual(want, have) {
		t.Fatalf("want %#+v, have %#+v", want, have)
	}
	if want, have := 1, len(serviceInfo.ContainerRules); want != have {
		t.Fatalf("want %d, have %d", want, have)
	}
	if want, have := containerRule, serviceInfo.ContainerRules[0].ContainerRule; !reflect.DeepEqual(want, have) {
		t.Fatalf("want %#+v, have %#+v", want, have)
	}

	/*
		s.AddService("s2", service)
		s.AddInstance("s2", "i2", instance)
		s.AddInstance("s2", "i3", instance)

		serviceInfos, err := s.GetAllServices(store.QueryServiceOptions{
			WithInstances:      true,
			WithContainerRules: true,
		})
		if err != nil {
			t.Fatalf("GetAllServices: %v", err)
		}
		t.Logf("GetAllServices: %#+v", serviceInfos)
	*/

	if err := s.RemoveInstance("s1", "i1"); err != nil {
		t.Fatalf("RemoveInstance: %v", err)
	}

	if err := s.RemoveService("s1"); err != nil {
		t.Fatalf("RemoveService: %v", err)
	}
}
