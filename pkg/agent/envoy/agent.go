package envoy

import (
	"context"
	"k8s.io/klog/v2"
	"time"
)

const errOutOfMemory = "signal: killed"

type Agent struct {
	envoy *Envoy

	// channel for Envoy exit notifications
	statusCh chan error
	abortCh  chan error
}

func NewAgent(envoy *Envoy, terminationDrainDuration, minDrainDuration time.Duration, localhost string,
	adminPort, statusPort, prometheusPort int, exitOnZeroActiveConnections bool,
) *Agent {

	return &Agent{
		envoy:    envoy,
		statusCh: make(chan error, 1),
		abortCh:  make(chan error, 1),
	}

}

func (a *Agent) Run(ctx context.Context) {
	klog.Info("Starting proxy agent")

	go a.runWait(a.abortCh)

	select {
	case err := <-a.statusCh:
		if err != nil {
			if err.Error() == errOutOfMemory {
				klog.Warningf("Envoy may have been out of memory killed. Check memory usage and limits.")
			}
			klog.Errorf("Envoy exited with error: %v", err)
		} else {
			klog.Infof("Envoy exited normally")
		}

	case <-ctx.Done():
		//a.terminate()
		klog.Info("Agent has successfully terminated")
	}
}

func (a *Agent) runWait(abortCh <-chan error) {
	err := a.envoy.Run(abortCh)
	//a.envoy.Cleanup()
	a.statusCh <- err
}
