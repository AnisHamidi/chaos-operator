/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"strconv"
	"strings"

	toxiproxy "github.com/Shopify/toxiproxy/client"
	"github.com/pingcap/errors"
	chaosv1alpha1 "github.com/snapp-incubator/toxiproxy-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NetworkChaosReconciler reconciles a NetworkChaos object
type NetworkChaosReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Constants for Toxiproxy configurations
const (
	toxiproxyImage  = "ghcr.io/shopify/toxiproxy" // The Docker image for Toxiproxy
	toxiproxyPort   = 8474                        // Default port for Toxiproxy
	portFormatIndex = 5                           // Index for extracting port in format "[::]:port"
)

//+kubebuilder:rbac:groups=chaos.snappcloud.io,resources=networkchaos,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=chaos.snappcloud.io,resources=networkchaos/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=chaos.snappcloud.io,resources=networkchaos/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NetworkChaos object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *NetworkChaosReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch NetworkChaos object
	networkChaos := &chaosv1alpha1.NetworkChaos{}
	err := r.Client.Get(ctx, req.NamespacedName, networkChaos)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "NetworkChaos resource not found", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get NetworkChaos", req.NamespacedName)
		return ctrl.Result{}, err
	}

	// Ensure toxiproxy Deployment is created
	if err := r.ensureToxiproxyDeployment(ctx, req, networkChaos); err != nil {
		return ctrl.Result{}, err
	}

	// Ensure toxiproxy Service is created
	if err := r.ensureToxiproxyService(ctx, req, networkChaos); err != nil {
		return ctrl.Result{}, err
	}

	// Manage Toxiproxy Proxies and Toxics
	if err := r.manageToxiproxyProxies(ctx, req, networkChaos); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkChaosReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1alpha1.NetworkChaos{}).
		Complete(r)
}

func (r *NetworkChaosReconciler) ensureToxiproxyDeployment(ctx context.Context, req ctrl.Request, networkChaos *chaosv1alpha1.NetworkChaos) error {
	log := log.FromContext(ctx)

	deployment := &appsv1.Deployment{}
	chaosName := networkChaos.GetName()

	// Try to get the Deployment if it exists
	err := r.Client.Get(ctx, types.NamespacedName{Name: "toxiproxy-" + chaosName, Namespace: req.Namespace}, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			dep := r.createToxiproxyDeployment(req.Namespace, chaosName)
			err = r.Client.Create(ctx, dep)
			if err != nil {
				log.Error(err, "Failed to create toxiproxy Deployment")
				return err
			}
			log.Info("Toxiproxy Deployment created successfully")
		} else {
			log.Error(err, "Failed to get toxiproxy Deployment")
			return err
		}
	}
	return nil
}
func (r *NetworkChaosReconciler) ensureToxiproxyService(ctx context.Context, req ctrl.Request, networkChaos *chaosv1alpha1.NetworkChaos) error {
	log := log.FromContext(ctx)

	svc := &corev1.Service{}
	chaosName := networkChaos.GetName()

	// Try to get the Service if it exists
	err := r.Client.Get(ctx, types.NamespacedName{Name: "toxiproxy-" + chaosName, Namespace: req.Namespace}, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			ser := r.createToxiproxyService(req.Namespace, "toxiproxy-"+chaosName, "toxiproxy-"+chaosName, toxiproxyPort, toxiproxyPort)
			err = r.Client.Create(ctx, ser)
			if err != nil {
				log.Error(err, "Failed to create toxiproxy Service")
				return err
			}
			log.Info("toxiproxy Service created successfully")
		} else {
			log.Error(err, "Failed to get toxiproxy Service")
			return err
		}
	}
	return nil
}
func (r *NetworkChaosReconciler) createToxiproxyDeployment(ns string, name string) *appsv1.Deployment {
	// Define labels
	labels := map[string]string{"app": "toxiproxy-" + name}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "toxiproxy-" + name,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "proxy",
						Image: toxiproxyImage,
						// Define other container attributes (ports, env vars, etc.)
					}},
				},
			},
		},
	}

	return dep
}
func (r *NetworkChaosReconciler) createToxiproxyService(ns string, name string, selector string, port int, targetPort int) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			//	Name:      "toxiproxy-" + ns,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": selector, // This should match labels of the Pods in the Deployment
			},
			Ports: []corev1.ServicePort{{
				Port:       int32(port),                // The port that the service should serve on
				TargetPort: intstr.FromInt(targetPort), // The target port on the pod to forward to
			}},
		},
	}

	return svc
}
func (r *NetworkChaosReconciler) manageToxiproxyProxies(ctx context.Context, req ctrl.Request, networkChaos *chaosv1alpha1.NetworkChaos) error {

	// Create a new Toxiproxy client
	// TODO
	// it should be change to name of toxiporxy service -> "toxiproxy-"+chaosName:toxiproxyPort
	//toxiproxyClient := toxiproxy.NewClient("localhost:8474")
	toxiproxyClient := toxiproxy.NewClient("toxiproxy-" + networkChaos.GetName() + req.Namespace + "-svc.cluster.local:8474")

	// Attempt to retrieve an existing proxy
	proxy, err := r.getOrCreateProxy(ctx, req, toxiproxyClient, networkChaos)
	if err != nil {
		return err
	}

	if err := r.ensureToxiproxyServiceForProxy(ctx, req, proxy, networkChaos); err != nil {
		return err
	}

	if err := r.manageToxics(ctx, req, proxy, networkChaos); err != nil {
		return err
	}

	return nil
}

func (r *NetworkChaosReconciler) getOrCreateProxy(ctx context.Context, req ctrl.Request, toxiproxyClient *toxiproxy.Client, networkChaos *chaosv1alpha1.NetworkChaos) (*toxiproxy.Proxy, error) {
	log := log.FromContext(ctx)

	proxy, err := toxiproxyClient.Proxy(networkChaos.GetName())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			// Proxy does not exist, create a new one
			// TODO
			// a service validation should be done on upstream name ******
			proxy, err = toxiproxyClient.CreateProxy(networkChaos.GetName(), "", networkChaos.Spec.Upstream.Name+":"+networkChaos.Spec.Upstream.Port)
			if err != nil {
				log.Error(err, "Failed to create proxy")
				return proxy, err
			}
			log.Info("proxy for service " + networkChaos.Spec.Upstream.Name + "created successfully in namespace " + req.Namespace)
		} else {
			log.Error(err, "Failed to get proxy")
			return proxy, err
		}
	}
	return proxy, nil
}
func (r *NetworkChaosReconciler) ensureToxiproxyServiceForProxy(ctx context.Context, req ctrl.Request, proxy *toxiproxy.Proxy, networkChaos *chaosv1alpha1.NetworkChaos) error {
	log := log.FromContext(ctx)

	proxyPort := proxy.Listen[portFormatIndex:] // the format is " [::]:port"
	port, err := strconv.Atoi(proxyPort)
	if err != nil {
		log.Error(err, "its empty")
	}
	svc := &corev1.Service{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: "toxiproxy-" + networkChaos.GetName() + networkChaos.Spec.Upstream.Name, Namespace: req.Namespace}, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			ser := r.createToxiproxyService(req.Namespace, "toxiproxy-"+networkChaos.GetName()+"-"+networkChaos.Spec.Upstream.Name, "toxiproxy-"+networkChaos.GetName(), port, port)
			err = r.Client.Create(ctx, ser)
			if err != nil {
				log.Error(err, "Failed to create Service")
				return err
			}

			log.Info("Service created successfully")
		} else {
			// Error other than NotFound
			log.Error(err, "Failed to get Service")
			return err
		}
	}
	return nil
}

func (r *NetworkChaosReconciler) manageToxics(ctx context.Context, req ctrl.Request, proxy *toxiproxy.Proxy, networkChaos *chaosv1alpha1.NetworkChaos) error {
	log := log.FromContext(ctx)

	// Check if the toxic already exists
	exists := false
	toxics, err := proxy.Toxics()
	if err != nil {
		log.Error(err, "Failed to get toxics")
		return err
	}

	for _, toxic := range toxics {
		if toxic.Name == networkChaos.GetName() {
			exists = true
			break
		}
	}

	// Add a toxic if it doesn't exist
	if !exists {
		_, err = proxy.AddToxic(networkChaos.GetName(), "latency", networkChaos.Spec.Stream, networkChaos.Spec.LatencyToxic.Probability, toxiproxy.Attributes{
			"latency": networkChaos.Spec.LatencyToxic.Latency,
			"jitter":  networkChaos.Spec.LatencyToxic.Jitter,
		})
		if err != nil {
			log.Error(err, "Failed to create toxic")
			return err
		}
		log.Info("toxic(name) added on port .... with this upstream")
	} else {
		log.Info("toxic(name) already exists")
	}
	return nil
}
