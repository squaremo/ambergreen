package metcdstore

import (
	"io/ioutil"
	"log"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/weaveworks/flux/common/store"
	"github.com/weaveworks/flux/common/store/test"

	"github.com/weaveworks/flux/common/netutil"
	"golang.org/x/net/context"
)

func TestMetcdStore(t *testing.T) {
	// This is a stopgap solution until metcdstore implements the complete Store
	// interface and we can use the nice test suite.

	// Start the metcd store and wait for it to be ready.
	logger := log.New(ioutil.Discard, "", log.LstdFlags)
	s := New(context.Background(), 1, logger)
	time.Sleep(3 * time.Second)

	// Check for a nonexistent service.
	if err := s.CheckRegisteredService("s1"); err == nil {
		t.Fatal("CheckRegisteredService: want error, have none")
	}

	// Remove a nonexistent service.
	if err := s.RemoveService("s1"); err == nil {
		t.Fatal("RemoveService: want error, have none")
	}

	// Add a service and check that it's there.
	a1, _ := netutil.ParseIPPort("1.2.3.4:56789", "tcp", false)
	service := store.Service{
		Address:      &a1,
		InstancePort: 1234,
		Protocol:     "tcp",
	}
	if err := s.AddService("s1", service); err != nil {
		t.Fatalf("AddService: %v", err)
	}
	if err := s.CheckRegisteredService("s1"); err != nil {
		t.Fatalf("CheckRegisteredService: %v", err)
	}

	// Remove a nonexistent instance.
	if err := s.RemoveInstance("s1", "i1"); err == nil {
		t.Fatal("RemoveInstance: want error, have none")
	}

	// Add an instance to the service.
	a2, _ := netutil.ParseIPPort("4.3.2.1:1000", "tcp", false)
	instance := store.Instance{
		Host:          store.Host{IP: net.ParseIP("10.11.12.13")},
		ContainerRule: "container-rule",
		Address:       &a2,
		Labels:        map[string]string{"label-key": "label-value"},
	}
	if err := s.AddInstance("s1", "i1", instance); err != nil {
		t.Fatalf("AddInstance: %v", err)
	}

	// Set a container rule on the service.
	containerRule := store.ContainerRule{
		Selector: store.Selector{
			"selector-key": "selector-value",
		},
	}
	if err := s.SetContainerRule("s1", "r1", containerRule); err != nil {
		t.Fatalf("SetContainerRule: %v", err)
	}

	// Get the service and make sure it has the instance and the container rule.
	serviceInfo, err := s.GetService("s1", store.QueryServiceOptions{
		WithInstances:      true,
		WithContainerRules: true,
	})
	if err != nil {
		t.Fatalf("GetService: %v", err)
	}
	if _, ok := serviceInfo.Instances["i1"]; !ok {
		t.Fatal("Service s1 missing instance i1")
	}
	if want, have := instance, serviceInfo.Instances["i1"]; !reflect.DeepEqual(want, have) {
		t.Fatalf("want %#+v, have %#+v", want, have)
	}
	if _, ok := serviceInfo.ContainerRules["r1"]; !ok {
		t.Fatal("Service s1 missing container rule r1")
	}
	if want, have := containerRule, serviceInfo.ContainerRules["r1"]; !reflect.DeepEqual(want, have) {
		t.Fatalf("want %#+v, have %#+v", want, have)
	}

	// Add a new service with two instances.
	if err := s.AddService("s2", service); err != nil {
		t.Fatalf("AddService: s2: %v", err)
	}
	if err := s.AddInstance("s2", "i2a", instance); err != nil {
		t.Fatalf("AddInstance: s2: i2a: %v", err)
	}
	if err := s.AddInstance("s2", "i2b", instance); err != nil {
		t.Fatalf("AddInstance: s2: i2b: %v", err)
	}

	// GetAllServices and make sure it returns both services.
	serviceInfos, err := s.GetAllServices(store.QueryServiceOptions{
		WithInstances:      true,
		WithContainerRules: true,
	})
	if err != nil {
		t.Fatalf("GetAllServices: %v", err)
	}
	if want, have := 2, len(serviceInfos); want != have {
		t.Fatalf("GetAllServices: want %d, have %d", want, have)
	}
	if want, have := 2, len(serviceInfos["s2"].Instances); want != have {
		t.Fatalf("GetAllServices: want %d, have %d", want, have)
	}
	if _, ok := serviceInfos["s2"].Instances["i2a"]; !ok {
		t.Fatal("GetAllServices: s2: i2a: not found")
	}
	if _, ok := serviceInfos["s2"].Instances["i2b"]; !ok {
		t.Fatal("GetAllServices: s2: i2b: not found")
	}
	if want, have := 0, len(serviceInfos["s2"].ContainerRules); want != have {
		t.Fatalf("GetAllServices: want %d, have %d", want, have)
	}

	// Remove instance i1 from service s1.
	if err := s.RemoveInstance("s1", "i1"); err != nil {
		t.Fatalf("RemoveInstance: %v", err)
	}

	// Remove service s1.
	if err := s.RemoveService("s1"); err != nil {
		t.Fatalf("RemoveService: %v", err)
	}

	// Remove all services and make sure no services are returned.
	if err := s.RemoveAllServices(); err != nil {
		t.Fatalf("RemoveAllServices: %v", err)
	}
	serviceInfos, err = s.GetAllServices(store.QueryServiceOptions{
		WithInstances:      true,
		WithContainerRules: true,
	})
	if err != nil {
		t.Fatalf("GetAllServices: %v", err)
	}
	if want, have := 0, len(serviceInfos); want != have {
		t.Fatalf("GetAllServices: want %d, have %d", want, have)
	}
}

func DontTestMetcdStoreSuite(t *testing.T) {
	logger := log.New(ioutil.Discard, "", log.LstdFlags)
	s := New(context.Background(), 1, logger)
	time.Sleep(3 * time.Second)
	test.RunStoreTestSuite(testableStore{s}, t)
}

type testableStore struct{ store.Store }

func (ts testableStore) Reset(t *testing.T) {
	if err := ts.Store.RemoveAllServices(); err != nil {
		t.Fatalf("during Reset: %v", err)
	}
}
