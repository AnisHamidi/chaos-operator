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

package v1alpha1

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var networkchaoslog = logf.Log.WithName("networkchaos-resource")

func (r *NetworkChaos) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-chaos-snappcloud-io-v1alpha1-networkchaos,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.snappcloud.io,resources=networkchaos,verbs=create;update,versions=v1alpha1,name=mnetworkchaos.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &NetworkChaos{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *NetworkChaos) Default() {
	networkchaoslog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-chaos-snappcloud-io-v1alpha1-networkchaos,mutating=false,failurePolicy=fail,sideEffects=None,groups=chaos.snappcloud.io,resources=networkchaos,verbs=create;update,versions=v1alpha1,name=vnetworkchaos.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NetworkChaos{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NetworkChaos) ValidateCreate() (admission.Warnings, error) {
	networkchaoslog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NetworkChaos) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	networkchaoslog.Info("validate update", "name", r.Name)
	// fmt.Printf("Type of old object: %T\n", old)
	// oldNetworkChaosSpec, ok := old.(*NetworkChaosSpec)
	// fmt.Printf("Type of oldNetworkChaosSpec object: %T\n", oldNetworkChaosSpec)
	oldNetworkChaos, _ := old.(*NetworkChaos)
	fmt.Printf("Type of oldNetworkChaos object: %T\n", oldNetworkChaos)

	// if !ok {
	// 	return nil, errors.New("invalid object type")
	// }
	if r.Spec.Stream != oldNetworkChaos.Spec.Stream {
		return nil, errors.New("modification of Stream field is not allowed")
	}
	if r.Spec.Upstream != oldNetworkChaos.Spec.Upstream {
		return nil, errors.New("modification of Upstream field is not allowed")
	}

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NetworkChaos) ValidateDelete() (admission.Warnings, error) {
	networkchaoslog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
