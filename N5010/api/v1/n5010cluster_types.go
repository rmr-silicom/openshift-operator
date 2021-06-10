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

type SyncStatus string

var (
	// InProgressSync indicates that the synchronization of the CR is in progress
	InProgressSync SyncStatus = "InProgress"
	// SucceededSync indicates that the synchronization of the CR succeeded
	SucceededSync SyncStatus = "Succeeded"
	// FailedSync indicates that the synchronization of the CR failed
	FailedSync SyncStatus = "Failed"
	// IgnoredSync indicates that the CR is ignored
	IgnoredSync SyncStatus = "Ignored"
)

type N5010Fpga struct {
	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\.\-\/]+
	UserImageURL string `json:"userImageURL"`
	// +kubebuilder:validation:Pattern=`^[a-fA-F0-9]{4}:[a-fA-F0-9]{2}:[01][a-fA-F0-9]\.[0-7]$`
	PCIAddr string `json:"PCIAddr"`
	// MD5 checksum verified against calculated one from downloaded user image. Optional.
	// +kubebuilder:validation:Pattern=`^[a-fA-F0-9]{32}$`
	CheckSum string `json:"checksum,omitempty"`
}

type N5010ClusterNode struct {
	// +kubebuilder:validation:Pattern=[a-z0-9\.\-]+
	NodeName string      `json:"nodeName"`
	FPGA     []N5010Fpga `json:"fpga,omitempty"`
}

// N5010ClusterSpec defines the desired state of N5010Cluster
type N5010ClusterSpec struct {
	// List of the nodes with their devices to be updated
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Nodes     []N5010ClusterNode `json:"nodes"`
	DryRun    bool               `json:"dryrun,omitempty"`
	DrainSkip bool               `json:"drainSkip,omitempty"`
}

// N5010ClusterStatus defines the observed state of N5010Cluster
type N5010ClusterStatus struct {
	// Indicates the synchronization status of the CR
	// +operator-sdk:csv:customresourcedefinitions:type=status
	SyncStatus    SyncStatus `json:"syncStatus,omitempty"`
	LastSyncError string     `json:"lastSyncError,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// N5010Cluster is the Schema for the n3000clusters API
// +operator-sdk:csv:customresourcedefinitions:displayName="N5010Cluster",resources={{N5010Node,v1,node}}
type N5010Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   N5010ClusterSpec   `json:"spec,omitempty"`
	Status N5010ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// N5010ClusterList contains a list of N5010Cluster
type N5010ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []N5010Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&N5010Cluster{}, &N5010ClusterList{})
}
