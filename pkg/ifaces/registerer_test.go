package ifaces

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
)

func TestRegisterer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := NewWatcher(10)
	registry := NewRegisterer(watcher, 10)
	// mock net.Interfaces and linkSubscriber to control which interfaces are discovered
	watcher.interfaces = func() ([]Interface, error) {
		return []Interface{{"foo", 1}, {"bar", 2}, {"baz", 3}}, nil
	}
	inputLinks := make(chan netlink.LinkUpdate, 10)
	watcher.linkSubscriber = func(ch chan<- netlink.LinkUpdate, done <-chan struct{}) error {
		go func() {
			for link := range inputLinks {
				ch <- link
			}
		}()
		return nil
	}

	outputEvents, err := registry.Subscribe(ctx)
	require.NoError(t, err)

	// initial set of fetched elements
	for i := 0; i < 3; i++ {
		getEvent(t, outputEvents, timeout)
	}
	assert.Equal(t, "foo", registry.ifaces[1])
	assert.Equal(t, "bar", registry.ifaces[2])
	assert.Equal(t, "baz", registry.ifaces[3])

	// updates
	inputLinks <- upAndRunning("bae", 4)
	inputLinks <- down("bar", 2)
	for i := 0; i < 2; i++ {
		getEvent(t, outputEvents, timeout)
	}

	assert.Equal(t, "foo", registry.ifaces[1])
	assert.NotContains(t, registry.ifaces, 2)
	assert.Equal(t, "baz", registry.ifaces[3])
	assert.Equal(t, "bae", registry.ifaces[4])

	// repeated updates that do not involve a change in the current track of interfaces
	// will be ignored
	inputLinks <- upAndRunning("fiu", 1)
	inputLinks <- down("foo", 1)
	for i := 0; i < 2; i++ {
		getEvent(t, outputEvents, timeout)
	}

	assert.Equal(t, "fiu", registry.ifaces[1])
	assert.Equal(t, "baz", registry.ifaces[3])
	assert.Equal(t, "bae", registry.ifaces[4])
}
