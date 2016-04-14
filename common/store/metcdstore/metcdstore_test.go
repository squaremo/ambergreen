package metcdstore

import (
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/weaveworks/flux/common/store"

	"golang.org/x/net/context"
)

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
		State:         store.NOADDR,
		Host:          store.Host{IPAddress: "10.32.1.2"},
		ContainerRule: "i1-container-rule",
		Address:       "i1-address",
		Port:          32001,
		Labels:        map[string]string{"label-key": "label-value"},
	}

	if err := s.AddInstance("s1", "i1", instance); err != nil {
		t.Fatalf("AddInstance: %v", err)
	}

	// TODO(pb): check

	if err := s.RemoveInstance("s1", "i1"); err != nil {
		t.Fatalf("RemoveInstance: %v", err)
	}

	if err := s.RemoveService("s1"); err != nil {
		t.Fatalf("RemoveService: %v", err)
	}
}
