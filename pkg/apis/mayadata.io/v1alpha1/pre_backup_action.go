/*
Copyright 2020 The MayaData Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    https://www.apache.org/licenses/LICENSE-2.0
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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PreBackupAction represents the pre-backup action for DMaaS
type PreBackupAction struct {
	metav1.TypeMeta `json:"inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata.omitempty"`

	// +optional
	Spec PreBackupActionSpec `json:"spec"`

	// +optional
	Status PreBackupActionStatus `json:"status"`
}

// PreBackupActionSpec represents the list of action to be executed prior to back up
type PreBackupActionSpec struct {
	// SetLabel defines the list of labels to be applied to resources
	// +optional
	SetLabel PreBackupActionSetLabel `json:"setLabel,omitempty"`

	// SetAnnotation defines the list of annotation to be applied on resources
	// +optional
	SetAnnotation PreBackupActionSetAnnotation `json:"setAnnotation,omitempty"`

	// OneTimeAction specify if this PrePreBackupAction needs to be executed periodically or not
	// - If true, action will be taken until it becomes success, marks Phase as Completed and stops further action.
	// - If false, action will be taken in regular intervals, leaving Phase as InProgress.
	// Default is false
	// +optional
	OneTimeAction bool `json:"oneTimeAction,omitempty"`
}

// PreBackupActionSetLabel represents the labels and, list of resource to be labeled
type PreBackupActionSetLabel struct {
	// Labels contains list of label to be set on specified resources
	Labels []string `json:"labels"`

	// IncludeResourceList is list of resource, need to be labeled
	IncludeResourceList []TargetResource `json:"includeResourceList"`

	// ExcludeResourceList is list of resources, need not to be labeled
	// +optional
	// +nullable
	ExcludeResourceList []TargetResource `json:"excludeResourceList,omitempty"`
}

// PreBackupActionSetAnnotation represents list of resource to be annotated
// with volume details for restic base backup
type PreBackupActionSetAnnotation struct {
	// IncludeResourceList is list of resources to be annotated
	IncludeResourceList []TargetResource `json:"includeResourceList"`

	// IncludeResourceList is list of resources not to be annotated
	// +optional
	// +nullable
	ExcludeResourceList []TargetResource `json:"excludeResourceList,omitempty"`
}

// TargetResource represents the k8s resource information used for pre-backup action
type TargetResource struct {
	// APIVersion of the resource
	// +optional
	APIVersion string `json:"apiversion,omitempty"`

	// Kind of the resource
	Kind string `json:"kind"`

	// Name of the resource
	// +optional
	Name []string `json:"name,omitempty"`

	// Namespace of the resource
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// PreBackupActionPhase reprensets the phase of prePreBackupAction
type PreBackupActionPhase string

const (
	// PreBackupActionPhaseInProgress reprensets the in progress phase of PreBackupAction
	PreBackupActionPhaseInProgress PreBackupActionPhase = "InProgress"

	// PreBackupActionPhaseCompleted reprensets the in completed phase of PreBackupAction
	PreBackupActionPhaseCompleted PreBackupActionPhase = "Completed"
)

// PreBackupActionStatus defines the status of PreBackupAction resource
type PreBackupActionStatus struct {
	// Phase defines the PreBackupAction stage
	Phase PreBackupActionPhase `json:"phase"`

	// LastSuccessfulTime represents the time when PreBackupAction executed successfully
	// +nullable
	LastSuccessfulTimestamp *metav1.Time `json:"lastSuccessfulTimestamp,omitempty"`

	// LastFailureTimeStamp represents the time when PreBackupAction failed
	// +nullable
	LastFailureTimestamp *metav1.Time `json:"lastFailureTimestamp,omitempty"`

	// UpdatedSelectedList represents the list of resources labeled successfully
	// +nullable
	UpdatedSelectedList []TargetResource `json:"updatedSelectedList,omitempty"`

	// UpdatedAnnotatedList represents the list of resources annotated successfully
	// +nullable
	UpdatedAnnotatedList []TargetResource `json:"updatedAnnotatedList,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PreBackupActionList represents the list of PreBackupAction resource
type PreBackupActionList struct {
	metav1.TypeMeta `json:"inline"`

	// +optional
	metav1.ListMeta `json:"metadata.omitempty"`

	Items []PreBackupAction `json:"items"`
}
