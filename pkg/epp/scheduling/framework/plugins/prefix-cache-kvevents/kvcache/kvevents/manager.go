package kvevents

import (
	"k8s.io/klog/v2"
	"sync"
	"context"
)

// SubscriberManager manages multiple ZMQ subscribers, one per LLM engine.
type SubscriberManager struct {
	mu          sync.RWMutex

	pool        *Pool
	subscribers map[string]*subscriberEntry
}

// subscriberEntry represents a single subscriber and its cancellation.
type subscriberEntry struct {
	subscriber *zmqSubscriber
	cancel     context.CancelFunc
	endpoint   string
}

func NewSubscriberManager(pool *Pool) *SubscriberManager {
	return &SubscriberManager{
		pool:        pool,
		subscribers: make(map[string]*subscriberEntry),
	}
}

func (manager *SubscriberManager)CreateOrUpdateSubscriber(ctx context.Context, podIdentifier, endpoint, topicFilter string, remoteSocket bool) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()


	// Check if subscriber already exists
	if entry, exists := manager.subscribers[podIdentifier]; exists {
		if entry.endpoint == endpoint {
			return nil
		}

		// Endpoint changed, remove old subscriber
		entry.cancel()
		delete(manager.subscribers, podIdentifier)
	}

	// Create new subscriber
	klog.Infof("Create new subscriber for podIdentifier:%s, endpoint:%s", podIdentifier, endpoint)
	subscriber := newZMQSubscriber(manager.pool, endpoint, topicFilter, remoteSocket)
	// Create a context and start subscriber
	subCtx, cancel := context.WithCancel(ctx)
	go subscriber.Start(subCtx)
	// Update subscribers
	manager.subscribers[podIdentifier] = &subscriberEntry{
		subscriber: subscriber,
		cancel:     cancel,
		endpoint:   endpoint,
	}

	return nil
}
