package agent

import (
	"context"
	"fmt"
	"github.com/lx1036/gateway/pkg/agent/envoy"
	mesh "istio.io/api/mesh/v1alpha1"
	"sync"
	"time"
)

const (
	localHostIPv4 = "127.0.0.1"
)

type AgentConfig struct {
	MinimumDrainDuration        time.Duration
	EnvoyStatusPort             int
	EnvoyPrometheusPort         int
	ExitOnZeroActiveConnections bool
}

type Agent struct {
	proxyConfig *mesh.ProxyConfig
	agentConfig *AgentConfig

	envoy       envoy.Agent
	envoyConfig envoy.ProxyConfig

	// Signals true completion (e.g. with delayed graceful termination of Envoy)
	wg sync.WaitGroup
}

func NewAgent(proxyConfig *mesh.ProxyConfig, agentOpts *AgentConfig, eopts envoy.ProxyConfig) *Agent {

	return &Agent{
		proxyConfig: proxyConfig,
		agentConfig: agentOpts,
		//secOpts:     sopts,
		envoyConfig: eopts,
		//fileWatcher: filewatcher.NewWatcher(),
	}
}

func (agent *Agent) Run(ctx context.Context) (func(), error) {

	// 1. start Sds server

	// 2. start envoy

	err := agent.initEnvoy()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize envoy agent: %v", err)
	}

	agent.wg.Add(1)
	go func() {
		defer agent.wg.Done()
		// This is a blocking call for graceful termination.
		agent.envoy.Run(ctx)
	}()

	return agent.wg.Wait, nil
}

func (agent *Agent) initEnvoy() error {

	// Back-fill envoy options from proxy config options
	agent.envoyConfig.BinaryPath = agent.proxyConfig.BinaryPath
	agent.envoyConfig.AdminPort = agent.proxyConfig.ProxyAdminPort
	agent.envoyConfig.DrainDuration = agent.proxyConfig.DrainDuration
	agent.envoyConfig.Concurrency = agent.proxyConfig.Concurrency.GetValue()
	//agent.envoyConfig.SkipDeprecatedLogs = agent.agentConfig.EnvoySkipDeprecatedLogs
	envoyProxy := envoy.NewEnvoy(agent.envoyConfig)
	drainDuration := agent.proxyConfig.TerminationDrainDuration.AsDuration()
	envoy.NewAgent(envoyProxy, drainDuration, agent.agentConfig.MinimumDrainDuration, localHostIPv4,
		int(agent.proxyConfig.ProxyAdminPort), agent.agentConfig.EnvoyStatusPort, agent.agentConfig.EnvoyPrometheusPort,
		agent.agentConfig.ExitOnZeroActiveConnections)

	return nil
}
