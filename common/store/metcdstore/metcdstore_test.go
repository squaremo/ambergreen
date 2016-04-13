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

	logger.Printf("CheckRegisteredService")
	if err := s.CheckRegisteredService("s1"); err == nil {
		t.Fatal("CheckRegisteredService: want error, have none")
	}

	logger.Printf("AddService")
	if err := s.AddService("s1", store.Service{
		Address:  "1.2.3.4",
		Port:     1234,
		Protocol: "tcp",
	}); err != nil {
		t.Fatalf("AddService: %v", err)
	}

	logger.Printf("CheckRegisteredService")
	if err := s.CheckRegisteredService("s1"); err != nil {
		t.Fatalf("CheckRegisteredService: %v", err)
	}
}
