package pool

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/weaveworks/flux/balancer/model"
	"github.com/weaveworks/flux/common/netutil"
)

func TestPoolOfOne(t *testing.T) {
	// Empty pool
	pool := NewInstancePool()
	require.Nil(t, pool.PickInstance())

	pool.UpdateInstances([]model.Instance{})
	require.Nil(t, pool.PickInstance())

	// Add an instance
	instance := model.Instance{
		Name:    "foo instance",
		Address: netutil.IPPort{net.IP{192, 168, 3, 135}, 32768},
	}

	pool.UpdateInstances([]model.Instance{instance})
	picked := pool.PickInstance()
	require.Equal(t, instance, picked.Instance)

	pool.Succeeded(picked)
	picked = pool.PickInstance()
	require.Equal(t, instance, picked.Instance)

	// Even if the instance is failed, it's the only one in the
	// pool, so it should still get picked
	pool.Failed(picked)
	require.Empty(t, pool.ready)
	require.Equal(t, instance, pool.PickInstance().Instance)

	// Remove instance
	pool.UpdateInstances([]model.Instance{})
	require.Nil(t, pool.PickInstance())
}

func TestFailAndRetryInstance(t *testing.T) {
	// if you fail an instance, it won't be considered ready until
	// it is retired.
	pool := NewInstancePool()
	now := time.Now()
	pool.now = func() time.Time { return now }

	inst1 := model.Instance{
		Name:    "instance one",
		Address: netutil.IPPort{net.IP{192, 168, 3, 101}, 1001},
	}
	inst2 := model.Instance{
		Name:    "instance two",
		Address: netutil.IPPort{net.IP{192, 168, 3, 102}, 1002},
	}
	inst3 := model.Instance{
		Name:    "instance three",
		Address: netutil.IPPort{net.IP{192, 168, 3, 103}, 1003},
	}

	pool.UpdateInstances([]model.Instance{inst1})
	picked1 := pool.PickInstance()
	require.Equal(t, inst1, picked1.Instance)
	pool.Failed(picked1)

	// incidentally test that failed instances remain failed, when
	// included in an update
	pool.UpdateInstances([]model.Instance{inst1, inst2})

	// check that inst2 (ready) is preferred to inst1 (failed)
	for i := 0; i < 20; i++ {
		picked2 := pool.PickInstance()
		require.Equal(t, inst2, picked2.Instance)
		pool.Succeeded(picked2)
	}

	// Fail inst2 and retry inst1
	now = now.Add(retry_interval_base)
	pool.Failed(pool.PickInstance())
	pool.ProcessRetries()

	// Now inst1 should get picked
	picked1 = pool.PickInstance()
	require.Equal(t, inst1, picked1.Instance)

	// Add a ready inst3
	pool.UpdateInstances([]model.Instance{inst1, inst2, inst3})

	// check that inst3 (ready) is preferred to inst1 (retrying)
	// and inst2 (failed)
	for i := 0; i < 20; i++ {
		picked3 := pool.PickInstance()
		require.Equal(t, inst3, picked3.Instance)
		pool.Succeeded(picked3)
	}

	pool.Succeeded(picked1)
	pool.UpdateInstances([]model.Instance{inst1, inst2})

	// inst3 has gone, inst2 is failed, so inst1 is preferred
	for i := 0; i < 20; i++ {
		picked1 = pool.PickInstance()
		require.Equal(t, inst1, picked1.Instance)
		pool.Succeeded(picked1)
	}
}

func TestRetryBackoff(t *testing.T) {
	pool := NewInstancePool()
	now := time.Now()
	pool.now = func() time.Time { return now }

	instance := model.Instance{
		Name:    "instance one",
		Address: netutil.IPPort{net.IP{192, 168, 3, 101}, 32768},
	}

	pool.UpdateInstances([]model.Instance{instance})

	for i := uint(0); i < 5; i++ {
		// invariant: the instance is ready here
		pool.Failed(pool.PickInstance())
		require.Empty(t, pool.ready)
		now = now.Add((1 << i) * retry_interval_base)
		pool.ProcessRetries()
		require.NotEmpty(t, pool.ready)
	}
}
