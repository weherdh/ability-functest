/*
Copyright 2022.

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

package v1

import (
	appv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LiteTestSpec defines the desired state of LiteTest
type LiteTestSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of LiteTest. Edit liteTest_types.go to remove/update
	LiteEnv        []corev1.EnvVar         `json:"liteEnv,omitempty"`
	ReportImage    string                  `json:"reportImage,omitempty"`
	JobTemplate    batchv1.JobTemplateSpec `json:"jobTemplate"`
	DeployTemplate appv1.DeploymentSpec    `json:"deployTemplate"`
}

// LiteTestStatus defines the observed state of LiteTest
type LiteTestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	//+optional
	TestStatus string
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LiteTest is the Schema for the litetests API
type LiteTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LiteTestSpec   `json:"spec,omitempty"`
	Status LiteTestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LiteTestList contains a list of LiteTest
type LiteTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LiteTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LiteTest{}, &LiteTestList{})
}
