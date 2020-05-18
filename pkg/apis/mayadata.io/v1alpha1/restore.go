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
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DMaasRestore represents the restore resource
type DMaasRestore struct {
	metav1.TypeMeta `json:"inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata.omitempty"`

	// +optional
	Spec RestoreSpec `json:"spec"`

	// +optional
	Status RestoreStatus `json:"status"`
}

// RestoreSpec defines the spec for restore
type RestoreSpec struct {
	// BackupName defines the backup from/till which restore should happen
	// +optional
	BackupName string `json:"backupName,omitempty"`

	// BackupScheduleName defines the schedule name from which restore should happen
	// +optional
	BackupScheduleName string `json:"backupScheduleName,omitempty"`

	// OpenEBSNamespace represents the nsamespace in which openebs is installed
	OpenEBSNamespace string `json:"openebsNamespace"`

	// RestoreSpec defines the spec for velero restore resource
	RestoreSpec velerov1.RestoreSpec `json:"restoreSpec"`
}

// RestoreStatusPhase represents the phase of restore
type RestoreStatusPhase string

const (
	// RestoreStatusPhaseInProgress represents in-progress phase of restore
	RestoreStatusPhaseInProgress RestoreStatusPhase = "InProgress"

	// RestoreStatusPhaseActive represents active phase of restore
	RestoreStatusPhaseActive RestoreStatusPhase = "Active"

	// RestoreStatusPhaseCompleted represents the completed restore
	RestoreStatusPhaseCompleted RestoreStatusPhase = "Completed"

	// RestoreStatusPhaseEmpty represents empty phase of restore
	RestoreStatusPhaseEmpty RestoreStatusPhase = ""
)

// RestoreStatus represents the status of restore
type RestoreStatus struct {
	// Phase represents the restore phase
	Phase RestoreStatusPhase `json:"phase"`

	// Reason represents the cause of failure in Restore
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message represents the cause/action/outcome of Restore
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions represents the pre-flight action, to be executed before restore, and it's status
	// +nullable
	Conditions []RestoreCondition `json:"conditions,omitempty"`

	// TotalBackupCount represents the count of backup, to be restored
	TotalBackupCount int `json:"totalBackupCount"`

	// CompletedBackupCount represents the restored count of backup
	CompletedBackupCount int `json:"completedBackupCount"`

	// Restores represents the list of RestoreDetails
	// +nullable
	Restores []RestoreDetails `json:"restoreList,omitempty"`
}

// RestoreConditionType represents the type of RestoreCondition
type RestoreConditionType string

const (
	// RestoreConditionTypeSynedBackupData represents the sync of Backup in k8s cluster
	RestoreConditionTypeSynedBackupData RestoreConditionType = "BackupSync"

	// RestoreConditionTypeValidatedBackupData represents the validation of Backup Data
	RestoreConditionTypeValidatedBackupData RestoreConditionType = "ValidateBackupData"

	// RestoreConditionTypeCheckedResources represents the validation of PVC, StorageClass, ServiceAccount
	RestoreConditionTypeCheckedResources RestoreConditionType = "ResourceValidate"

	// RestoreConditionTypeIPSet represents the setting of IP address
	RestoreConditionTypeIPSet RestoreConditionType = "Setting IP Successful"
)

// RestoreCondition represents the restore pre-flight check
type RestoreCondition struct {
	/* Predefined types of Restore */
	// Type defines the RestoreCondition type
	Type RestoreConditionType `json:"type"`

	// Status defines the status of Condition, True or False
	Status corev1.ConditionStatus `json:"status"`

	// LastUpdatedTime represents the last updation time of condition
	// +nullable
	LastUpdatedTime *metav1.Time `json:"lastUpdatedTime,omitempty"`

	// LastTransitionTime represents the transition time of condition
	// +nullable
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	/* Reason, Message when error occurs */
	// Reason represents the cause of failure in condition
	Reason string `json:"reason,omitempty"`

	// Message represents the cause/warning/message from condition
	Message string `json:"message,omitempty"`
}

// RestoreDetails represents the restore information
type RestoreDetails struct {
	// RestoreName is velero restore resource name
	RestoreName string `json:"name"`

	// BackupName is name of backup from which restore is created
	BackupName string `json:"backupName"`

	// StartedAt is time stamp at which restore started
	// +nullable
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// CompletedAt is time stamp at which restore completed
	// +nullable
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// State represents the restore state
	State string `json:"state"`

	// PVRestoreStatus represents the list of PVRestoreStatusDetails
	// +nullable
	PVRestoreStatus []PVRestoreStatusDetails `json:"snapshotStatus,omitempty"`
}

// PVRestoreStatusDetails represents the status of PV restore
type PVRestoreStatusDetails struct {
	// SnapshotID is snapshotID from which restore started
	SnapshotID string `json:"snapshotID"`

	// Type defines the snapshot type
	Type SnapshotType `json:"snapshotType"`

	// Phase represents the restore phase of snapshot
	Phase string `json:"phase"`

	// StartTimestamp represents the timestamp at which restore of the snapshot started
	// +nullable
	StartTimestamp *metav1.Time `json:"startTimestamp,omitempty"`

	// CompletionTimestamp represents the timestamp at which restore of the snapshot completed
	// +nullable
	CompletionTimestamp *metav1.Time `json:"completionTimestamp,omitempty"`

	// Conditions represents the list of condition for the PV restore
	// +nullable
	Conditions []RestoreCondition `json:"conditions,omitempty"`

	// Progress represents the progress of snapshot restore
	Progress Progress `json:"progress"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DMaasRestoreList represents the list of DMaasRestore resource
type DMaasRestoreList struct {
	metav1.TypeMeta `json:"inline"`

	// +optional
	metav1.ListMeta `json:"metadata.omitempty"`

	Items []DMaasRestore `json:"items"`
}
