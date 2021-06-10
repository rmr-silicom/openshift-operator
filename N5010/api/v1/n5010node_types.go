// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2021 Intel Corporation

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// N5010NodeSpec defines the desired state of N5010Node
type N5010NodeSpec struct {
	// FPGA devices to be updated
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	FPGA []N5010Fpga `json:"fpga,omitempty"`
	// Fortville devices to be updated
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	DryRun bool `json:"dryRun,omitempty"`
	// Allows for updating devices without draining the node
	DrainSkip bool `json:"drainSkip,omitempty"`
}

// N5010NodeStatus defines the observed state of N5010Node
type N5010NodeStatus struct {
	// Provides information about device update status
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Provides information about FPGA inventory on the node
	// +operator-sdk:csv:customresourcedefinitions:type=status
	FPGA []N5010FpgaStatus `json:"fpga,omitempty"`
}

type N5010FpgaStatus struct {
	PciAddr          string `json:"PCIAddr,omitempty"`
	DeviceID         string `json:"deviceId,omitempty"`
	BitstreamID      string `json:"bitstreamId,omitempty"`
	BitstreamVersion string `json:"bitstreamVersion,omitempty"`
	BootPage         string `json:"bootPage,omitempty"`
	NumaNode         int    `json:"numaNode,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Flash",type=string,JSONPath=`.status.conditions[?(@.type=="Flashed")].reason`

// N5010Node is the Schema for the n3000nodes API
// +operator-sdk:csv:customresourcedefinitions:displayName="N5010Node",resources={{N5010Node,v1,node}}
type N5010Node struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   N5010NodeSpec   `json:"spec,omitempty"`
	Status N5010NodeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// N5010NodeList contains a list of N5010Node
type N5010NodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []N5010Node `json:"items"`
}

func init() {
	SchemeBuilder.Register(&N5010Node{}, &N5010NodeList{})
}
