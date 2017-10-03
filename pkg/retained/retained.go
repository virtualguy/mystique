// Copyright © 2017 The Things Industries, distributed under the MIT license (see LICENSE file)

// Package retained implements a store for retained messages.
package retained

import (
	"context"
	"sync"

	"github.com/TheThingsIndustries/mystique/pkg/packet"
	"github.com/TheThingsIndustries/mystique/pkg/topic"
)

// Store for retained messages
type Store interface {
	// Retain the message if the RETAIN bit is set
	Retain(*packet.PublishPacket)

	// Get all currently retained messages that match the filters
	Get(filter ...string) []*packet.PublishPacket
}

// SimpleStore returns a simple store for retained messages
func SimpleStore(ctx context.Context) Store {
	return &retainedMessages{
		ctx:      ctx,
		messages: make(map[string]*packet.PublishPacket),
	}
}

type retainedMessages struct {
	mu       sync.RWMutex
	ctx      context.Context
	messages map[string]*packet.PublishPacket
}

func (r *retainedMessages) Retain(pkt *packet.PublishPacket) {
	if !pkt.Retain {
		return
	}
	pkt.Retain = false // Unset retain flag on original message
	r.mu.Lock()
	if len(pkt.Message) > 0 {
		retained := *pkt
		retained.Retain = true // Set retain flag on message copy
		r.messages[pkt.TopicName] = &retained
		retainedGauge.Inc()
	} else if _, ok := r.messages[pkt.TopicName]; ok {
		delete(r.messages, pkt.TopicName)
		retainedGauge.Dec()
	}
	r.mu.Unlock()
}

func (r *retainedMessages) Get(filter ...string) (packets []*packet.PublishPacket) {
	r.mu.RLock()
	defer r.mu.RUnlock()
nextMessage:
	for t, pkt := range r.messages {
		for _, filter := range filter {
			if topic.Match(t, filter) {
				packets = append(packets, pkt)
				continue nextMessage
			}
		}
	}
	return
}