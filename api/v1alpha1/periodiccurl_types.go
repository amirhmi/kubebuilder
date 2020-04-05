/*


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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//Define custom types
type URL string

// PeriodicCurlSpec defines the desired state of PeriodicCurl
type PeriodicCurlSpec struct {
	Url string `json:"url,omitempty"`

	Coefficient int32 `json:"coefficient,omitempty"`
}

// PeriodicCurlStatus defines the observed state of PeriodicCurl
type PeriodicCurlStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	NextCurlTime string `json:"nextCurlTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PeriodicCurl is the Schema for the periodiccurls API
type PeriodicCurl struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PeriodicCurlSpec   `json:"spec,omitempty"`
	Status PeriodicCurlStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PeriodicCurlList contains a list of PeriodicCurl
type PeriodicCurlList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PeriodicCurl `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PeriodicCurl{}, &PeriodicCurlList{})
}
