package envoy

import (
	"google.golang.org/protobuf/types/known/durationpb"
	"os"
	"os/exec"

	"k8s.io/klog/v2"
)

var (
	enableEnvoyCoreDump = os.Getenv("ENABLE_ENVOY_CORE_DUMP") == "true"
)

type ProxyConfig struct {
	BinaryPath string

	AdminPort     int32
	DrainDuration *durationpb.Duration
	Concurrency   int32

	ComponentLogLevel  string
	SkipDeprecatedLogs bool
}

type Envoy struct {
	ProxyConfig
	extraArgs []string
}

func NewEnvoy(cfg ProxyConfig) *Envoy {
	var args []string

	if cfg.ComponentLogLevel != "" {
		// Use the old setting if we don't set any component log levels in LogLevel
		args = append(args, "--component-log-level", cfg.ComponentLogLevel)
	}

	if cfg.SkipDeprecatedLogs {
		args = append(args, "--skip-deprecated-logs")
	}

	// Explicitly enable core dumps. This may be desirable more often (by default), but for now we only set it in VM tests.
	if enableEnvoyCoreDump {
		args = append(args, "--enable-core-dump")
	}

	return &Envoy{
		ProxyConfig: cfg,
		extraArgs:   args,
	}
}

func (envoy *Envoy) Run(abort <-chan error) error {

	// /usr/local/bin/envoy -c /etc/istio/proxy/envoy-rev.json --drain-time-s 45 --drain-strategy immediate --local-address-ip-version v4 --file-flush-interval-msec 1000
	// --disable-hot-restart --allow-unknown-static-fields -l warning --component-log-level misc:error --skip-deprecated-logs --concurrency 2
	cmd := exec.Command(envoy.BinaryPath) //, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-abort:
		klog.Warningf("Aborting proxy")
		if errKill := cmd.Process.Kill(); errKill != nil {
			klog.Warningf("killing proxy caused an error %v", errKill)
		}
		return err
	case err := <-done:
		return err
	}
}
