package kvevents

import (
	"context"
	"encoding/binary"
	"time"

	"k8s.io/klog/v2"
	"github.com/go-zeromq/zmq4"
)

// RawMessage holds the raw transport-level data from a received pub/sub message.
type RawMessage struct {
	// Topic is the original transport topic string.
	Topic string
	// Sequence is the message sequence number from the transport.
	Sequence uint64
	// Payload is the raw encoded event batch bytes, not yet decoded.
	Payload []byte
}

// zmqSubscriber connects to a ZMQ publisher and forwards messages to a pool.
type zmqSubscriber struct {
	pool        *Pool
	endpoint    string
	remote      bool
	topicFilter string
}

// newZMQSubscriber creates a new ZMQ subscriber.
func newZMQSubscriber(pool *Pool, endpoint, topicFilter string, remote bool) *zmqSubscriber {
	return &zmqSubscriber{
		pool:        pool,
		endpoint:    endpoint,
		remote:      remote,
		topicFilter: topicFilter,
	}
}

func (zmq *zmqSubscriber) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			klog.Infof("shutting down zmq subscriber for endpoint %s", zmq.endpoint)
			return

		default:
			zmq.runSubscriber(ctx)
			select {
			case <-time.After(5 * time.Second):
				klog.Infof("waiting for seconds for retrying...")
			case <-ctx.Done():
				klog.Infof("shutting down zmq subscriber for endpoint %s", zmq.endpoint)
				return
			}
		}
	}
}

func (zmq *zmqSubscriber) runSubscriber(ctx context.Context) {
	subSocket := zmq4.NewSub(ctx,
		zmq4.WithAutomaticReconnect(true),
		zmq4.WithDialerMaxRetries(-1),
	)
	defer subSocket.Close()

	// server 端
	if !zmq.remote {
		if err := subSocket.Listen(zmq.endpoint); err != nil {
			klog.Errorf("failed to linsten on endpoint %s: %v", zmq.endpoint, err)
			return
		}

		klog.Infof("listening on endpoint %s successfully", zmq.endpoint)
	} else {
		if err := subSocket.Dial(zmq.endpoint); err != nil {
			klog.Errorf("failed to dial endpoint %s: %v", zmq.endpoint, err)
			return
		}

		klog.Infof("dialing endpoint %s successfully", zmq.endpoint)
	}

	if err := subSocket.SetOption(zmq4.OptionSubscribe, zmq.topicFilter); err != nil {
		klog.Errorf("failed to set subscribe option: %v", err)
		return
	}

	for {
		msg, err := subSocket.Recv()
		if err != nil {
			if ctx.Err() != nil {
				return // context cancel, shutdown, no error msg
			}

			klog.Errorf("failed to receive message: %v", err)
			return
		}

		parts := msg.Frames
		if len(parts) != 3 {
			klog.Errorf("invalid message format: %v", parts)
			continue
		}

		/**
		/Users/lx1036/Code/k8s/vllm/vllm/distributed/kv_events.py
		payload = self._pack.encode(event)
        seq_bytes = seq.to_bytes(8, "big")
        self._pub.send_multipart((self._topic_bytes, seq_bytes, payload))
		 */
		topic := string(parts[0])
		seqBytes := parts[1]
		payload := parts[2]
		seq := binary.BigEndian.Uint64(seqBytes)

		// 发送到 pool
		zmq.pool.AddTask(&RawMessage{
			Topic: topic,
			Sequence:   seq,
			Payload:  payload,
		})
	}
}
