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
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ArgoRolloutConfigKeeperClusterScopeSpec defines the desired state of ArgoRolloutConfigKeeperClusterScope
type ArgoRolloutConfigKeeperClusterScopeSpec struct {
	FinalizerName       string            `json:"finalizerName"`
	AppLabel            string            `json:"appLabel,omitempty"`
	AppVersionLabel     string            `json:"appVersionLabel,omitempty"`
	ConfigLabelSelector map[string]string `json:"configLabelSelector,omitempty"`
	IgnoredNamespaces   []string          `json:"ignoredNamespaces,omitempty"`
}

func (in *ArgoRolloutConfigKeeperClusterScopeSpec) UnmarshalJSON(b []byte) error {
	type alias ArgoRolloutConfigKeeperClusterScopeSpec
	tmp := struct {
		*alias
	}{
		alias: (*alias)(in),
	}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	if tmp.alias.AppVersionLabel == "" {
		tmp.alias.AppVersionLabel = "app.kubernetes.io/version"
	}
	if tmp.alias.AppLabel == "" {
		tmp.alias.AppLabel = "app.kubernetes.io/name"
	}
	return nil
}

// ArgoRolloutConfigKeeperClusterScopeStatus defines the observed state of ArgoRolloutConfigKeeperClusterScope
type ArgoRolloutConfigKeeperClusterScopeStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="FinalizerName",type="string",JSONPath=".spec.finalizerName",description="The name of managed Finalizer"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath="..metadata.creationTimestamp",description="Aqua Database Age"

// ArgoRolloutConfigKeeperClusterScope is the Schema for the argorolloutconfigkeeperclusterscopes API
type ArgoRolloutConfigKeeperClusterScope struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArgoRolloutConfigKeeperClusterScopeSpec   `json:"spec,omitempty"`
	Status ArgoRolloutConfigKeeperClusterScopeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ArgoRolloutConfigKeeperClusterScopeList contains a list of ArgoRolloutConfigKeeperClusterScope
type ArgoRolloutConfigKeeperClusterScopeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArgoRolloutConfigKeeperClusterScope `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArgoRolloutConfigKeeperClusterScope{}, &ArgoRolloutConfigKeeperClusterScopeList{})
}
