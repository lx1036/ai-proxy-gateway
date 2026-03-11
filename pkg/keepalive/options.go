package keepalive

import (
	"github.com/lx1036/gateway/pkg/util"
	"math"
	"time"
)

const (
	// Infinity is the maximum possible duration for keepalive values
	Infinity = time.Duration(math.MaxInt64)
)

var (
	grpcKeepaliveInterval = util.LookupEnv("GRPC_KEEPALIVE_INTERVAL", "30") // time.Second
)

type Options struct {
	// After a duration of this time if the server/client doesn't see any activity it pings the peer to see if the transport is still alive.
	Time time.Duration
	// After having pinged for keepalive check, the server waits for a duration of Timeout and if no activity is seen even after that
	// the connection is closed.
	Timeout time.Duration
	// MaxServerConnectionAge is a duration for the maximum amount of time a
	// connection may exist before it will be closed by the server sending a GoAway.
	// A random jitter is added to spread out connection storms.
	// See https://github.com/grpc/grpc-go/blob/bd0b3b2aa2a9c87b323ee812359b0e9cda680dad/keepalive/keepalive.go#L49
	MaxServerConnectionAge time.Duration // default value is infinity
	// MaxServerConnectionAgeGrace is an additive period after MaxServerConnectionAge
	// after which the connection will be forcibly closed by the server.
	MaxServerConnectionAgeGrace time.Duration // default value 10s
}

func DefaultOption() *Options {
	return &Options{
		Time:                        time.Second * 30,
		Timeout:                     time.Second * 10,
		MaxServerConnectionAge:      Infinity,
		MaxServerConnectionAgeGrace: 10 * time.Second,
	}
}
