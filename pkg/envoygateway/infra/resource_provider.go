package infra

import (
	"context"
	"fmt"
	"github.com/lx1036/gateway/pkg/envoygateway/xds/bootstrap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/lx1036/gateway/pkg/envoygateway/ir"

	envoygatewayv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"

)

const (
	EnvoyReadinessPort = 19003

	ShutdownManagerReadyPath = "/shutdown/ready"
	ShutdownManagerPort = 19002

	ShutdownManagerHealthCheckPath = "/healthz"


)

type ResourceRender struct {
	infra *ir.ProxyInfra

}


func NewResourceRender(ctx context.Context, infra *ir.Infra) (*ResourceRender, error) {



	return &ResourceRender{

	}, nil
}


func (r *ResourceRender) Deployment() (*appsv1.Deployment, error) {

	envoyProxy := r.infra.GetEnvoyProxy()
	provider := envoyProxy.GetEnvoyProxyProvider()
	if provider.Type != envoygatewayv1alpha1.EnvoyProxyProviderTypeKubernetes {
		return nil, fmt.Errorf("invalid provider type %v for Kubernetes infra manager", provider.Type)
	}

	deploymentConfig := provider.Kubernetes.EnvoyDeployment
	if deploymentConfig == nil {
		return nil, nil
	}

	containers, err := r.getEnvoyProxyContainers(deploymentConfig.Pod)


	deployment := &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			//Name:      "envoy-gateway-proxy",
			Namespace: r.Namespace(),
			Labels: ,
			Annotations: ,
			OwnerReferences: r.ownerReferences(),
		},
		Spec:       appsv1.DeploymentSpec{
			Replicas:                deploymentConfig.Replicas,
			Selector:                nil,
			Template:                corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ,
					Annotations: ,
				},
				Spec:       corev1.PodSpec{
					AutomountServiceAccountToken:  ptr.To[bool](false),
					Containers:                    containers,
					InitContainers:                deploymentConfig.InitContainers,
					ServiceAccountName:            r.serviceAccountName(),
					//TerminationGracePeriodSeconds: expectedTerminationGracePeriodSeconds(proxyConfig.Spec.Shutdown),
					DNSPolicy:                     corev1.DNSClusterFirst,
					RestartPolicy:                 corev1.RestartPolicyAlways,
					SchedulerName:                 "default-scheduler", // INFO: 使用 default-scheduler
					SecurityContext:               deploymentConfig.Pod.SecurityContext,
					Affinity:                      deploymentConfig.Pod.Affinity,
					Tolerations:                   deploymentConfig.Pod.Tolerations,
					//Volumes:                       r.expectedVolumes(deploymentConfig.Pod),
					ImagePullSecrets:              deploymentConfig.Pod.ImagePullSecrets,
					NodeSelector:                  deploymentConfig.Pod.NodeSelector,
					TopologySpreadConstraints:     deploymentConfig.Pod.TopologySpreadConstraints,
					PriorityClassName:             ptr.Deref(deploymentConfig.Pod.PriorityClassName, ""),
				},
			},
			Strategy:                *deploymentConfig.Strategy,
			//MinReadySeconds:         0,
			RevisionHistoryLimit:    ptr.To[int32](10),
			//Paused:                  false,
			ProgressDeadlineSeconds: ptr.To[int32](600),
		},
	}


}

func (r *ResourceRender) getEnvoyProxyContainers() ([]corev1.Container, error){


	containers := []corev1.Container{
		{
			Name: "envoy",
			Image:                    proxyImage,
			ImagePullPolicy:          corev1.PullIfNotPresent,
			Command:                  []string{"envoy"},
			Args:                     args,
			Env:                      expectedContainerEnv(containerSpec),
			Resources:                *containerSpec.Resources,
			Ports:                    ports,
			//VolumeMounts:             expectedContainerVolumeMounts(containerSpec, gatewayNamespaceMode),
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			TerminationMessagePath:   "/dev/termination-log",
			//SecurityContext:          expectedEnvoySecurityContext(containerSpec),


			StartupProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   "/ready",
						Port:   intstr.IntOrString{Type: intstr.Int, IntVal: EnvoyReadinessPort},
						Scheme: corev1.URISchemeHTTP,
					},
				},
				TimeoutSeconds:   1,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				FailureThreshold: 30,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   "/ready",
						Port:   intstr.IntOrString{Type: intstr.Int, IntVal: EnvoyReadinessPort},
						Scheme: corev1.URISchemeHTTP,
					},
				},
				TimeoutSeconds:   1,
				PeriodSeconds:    5,
				SuccessThreshold: 1,
				FailureThreshold: 1,
			},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   "/ready",
						Port:   intstr.IntOrString{Type: intstr.Int, IntVal: EnvoyReadinessPort},
						Scheme: corev1.URISchemeHTTP,
					},
				},
				TimeoutSeconds:   1,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},


			Lifecycle: &corev1.Lifecycle{
				PreStop: &corev1.LifecycleHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   ShutdownManagerReadyPath,
						Port:   intstr.FromInt32(ShutdownManagerPort),
						Scheme: corev1.URISchemeHTTP,
					},
				},
			},

		},
		{
			Name: "shutdown-manager",
			Image:                    expectedShutdownManagerImage(shutdownManager),
			ImagePullPolicy:          corev1.PullIfNotPresent,
			Command:                  []string{"envoy-gateway"},
			Args:                     expectedShutdownManagerArgs(shutdownConfig),
			Env:                      expectedContainerEnv(nil),
			Resources:                *envoygatewayv1alpha1.DefaultShutdownManagerContainerResourceRequirements(),
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			TerminationMessagePath:   "/dev/termination-log",
			SecurityContext: expectedShutdownManagerSecurityContext(containerSpec),

			StartupProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   ShutdownManagerHealthCheckPath,
						Port:   intstr.IntOrString{Type: intstr.Int, IntVal: ShutdownManagerPort},
						Scheme: corev1.URISchemeHTTP,
					},
				},
				TimeoutSeconds:   1,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				FailureThreshold: 30,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   ShutdownManagerHealthCheckPath,
						Port:   intstr.IntOrString{Type: intstr.Int, IntVal: ShutdownManagerPort},
						Scheme: corev1.URISchemeHTTP,
					},
				},
				TimeoutSeconds:   1,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   ShutdownManagerHealthCheckPath,
						Port:   intstr.IntOrString{Type: intstr.Int, IntVal: ShutdownManagerPort},
						Scheme: corev1.URISchemeHTTP,
					},
				},
				TimeoutSeconds:   1,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
			Lifecycle: &corev1.Lifecycle{
				PreStop: &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{
						Command: expectedShutdownPreStopCommand(shutdownConfig),
					},
				},
			},
		},
	}

	return containers, nil
}



func BuildProxyArgs(infra *ir.ProxyInfra,) ([]string, error){

	bootstrapConfigurations, err := bootstrap.GetRenderBootstrapConfig(bootstrapConfigOptions)


	args := []string{
		"--service-cluster", serviceCluster,
		"--service-node", serviceNode,
		"--config-yaml", bootstrapConfigurations,
		"--log-level", string(infra.EnvoyProxy.Spec.Logging.DefaultEnvoyProxyLoggingLevel()),
		"--cpuset-threads",
		"--drain-strategy", "immediate",
	}


	if infra.EnvoyProxy != nil {
		args = append(args, infra.EnvoyProxy.Spec.ExtraArgs...)
	}

	return args, nil
}
