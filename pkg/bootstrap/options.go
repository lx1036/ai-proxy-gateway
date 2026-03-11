package bootstrap

import (
	"github.com/lx1036/gateway/pkg/keepalive"
	"github.com/lx1036/gateway/pkg/util"
	"time"
)

type PilotArgs struct {
	//ServerOptions      DiscoveryServerOptions
	//InjectionOptions   InjectionOptions
	PodName            string
	Namespace          string
	CniNamespace       string
	Revision           string
	MeshConfigFile     string
	NetworksConfigFile string
	//RegistryOptions    RegistryOptions
	//CtrlZOptions       *ctrlz.Options
	//KrtDebugger        *krt.DebugHandler `json:"-"`
	//KeepaliveOptions   *keepalive.Options
	ShutdownDuration time.Duration
	JwtRule          string

	ServerOptions DiscoveryServerOptions

	KeepaliveOptions *keepalive.Options

	RegistryOptions RegistryOptions
}

type DiscoveryServerOptions struct {
	// The listening address for HTTP (debug). If the port in the address is empty or "0" (as in "127.0.0.1:" or "[::1]:0")
	// a port number is automatically chosen.
	HTTPAddr string

	// The listening address for HTTPS (webhooks). If the port in the address is empty or "0" (as in "127.0.0.1:" or "[::1]:0")
	// a port number is automatically chosen.
	// If the address is empty, the secure port is disabled, and the
	// webhooks are registered on the HTTP port - a gateway in front will
	// terminate TLS instead.
	HTTPSAddr string

	// The listening address for gRPC. If the port in the address is empty or "0" (as in "127.0.0.1:" or "[::1]:0")
	// a port number is automatically chosen.
	GRPCAddr string

	// The listening address for the monitoring port. If the port in the address is empty or "0" (as in "127.0.0.1:" or "[::1]:0")
	// a port number is automatically chosen.
	MonitoringAddr string

	EnableProfiling bool

	// Optional TLS configuration
	//TLSOptions TLSOptions

	// The listening address for secured gRPC. If the port in the address is empty or "0" (as in "127.0.0.1:" or "[::1]:0")
	// a port number is automatically chosen.
	SecureGRPCAddr string
}

type RegistryOptions struct {
	// If FileDir is set, the below kubernetes options are ignored
	FileDir string

	Registries []string

	// Kubernetes controller options
	//KubeOptions kubecontroller.Options
	// ClusterRegistriesNamespace specifies where the multi-cluster secret resides
	ClusterRegistriesNamespace string
	KubeConfig                 string
}

func NewPilotArgs(initFuncs ...func(*PilotArgs)) *PilotArgs {
	p := &PilotArgs{}

	// Apply Default Values.
	p.applyDefaults()

	// Apply custom initialization functions.
	for _, fn := range initFuncs {
		fn(p)
	}

	return p
}

var (
	PodNamespace = util.LookupEnv("PodNamespace", "ai-proxy-gateway-system")
	PodName      = util.LookupEnv("PodName", "")
	Revision     = util.LookupEnv("Revision", "")
)

func (p *PilotArgs) applyDefaults() {
	p.Namespace = PodNamespace
	p.CniNamespace = PodNamespace
	p.PodName = PodName
	p.Revision = Revision
	//p.RegistryOptions.KubeOptions.Revision = Revision
	//p.JwtRule = JwtRule
	p.KeepaliveOptions = keepalive.DefaultOption()
	p.RegistryOptions.ClusterRegistriesNamespace = p.Namespace
	//p.KrtDebugger = new(krt.DebugHandler)
}
