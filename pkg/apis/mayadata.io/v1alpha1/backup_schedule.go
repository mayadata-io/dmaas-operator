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

// Package v1alpha1 provides v1alpha1 version API spec for dmaas-operator
package v1alpha1

import (
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupSchedule represents the BackBackupSchedule resource
type BackupSchedule struct {
	metav1.TypeMeta `json:"inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata.omitempty"`

	// +optional
	Spec BackupScheduleSpec `json:"spec"`

	// +optional
	Status BackupScheduleStatus `json:"status"`
}

// BackupScheduleSpec defines the spec for BacBackupSchedule resource
type BackupScheduleSpec struct {
	// State defines if given BacBackupSchedule is active or not
	// Default value is Active
	// +optional
	// +nullable
	State *string `json:"state,omitempty"`

	// PeriodicFullBackupCfg defines the config for periodic full backup
	// if PeriodicFullBackupCfg is provided then VeleroScheduleSpec should not be empty
	// +optional
	PeriodicFullBackupCfg PeriodicFullBackupConfig `json:"periodicFullBackup,omitempty"`

	// VeleroScheduleSpec defines the spec for backup schedule
	// In case of non-scheduled backup, VeleroScheduleSpec will be empty
	// +optional
	VeleroScheduleSpec velerov1.ScheduleSpec `json:"veleroScheduleSpec,omitempty"`
}

// PeriodicFullBackupConfig defines the configuration for periodic full backup
type PeriodicFullBackupConfig struct {
	// CronTime is cron expression defining when to run full backup
	CronTime string `json:"cronTime,omitempty"`

	// RetryThresholdOnFailure defines number of retry should be
	// attempted on backup failure
	// Default value is 0
	RetryThresholdOnFailure int `json:"retryThresholdOnFailure,omitempty"`

	// FullBackupRetentionThreshold represents the number of full backup needs to be retained
	FullBackupRetentionThreshold int `json:"fullBackupRetentionThreshold"`
}

// BackupSchedulePhase represents the phase of BackupSchedule
type BackupSchedulePhase string

const (
	// BackupSchedulePhaseInProgress represents the in progress phase of BackupSchedule
	BackupSchedulePhaseInProgress BackupSchedulePhase = "InProgress"

	// BackupSchedulePhasePaused represents the pause phase of BackupSchedule
	BackupSchedulePhasePaused BackupSchedulePhase = "Paused"

	// BackupSchedulePhaseActive represents the active phase of BackupSchedule
	BackupSchedulePhaseActive BackupSchedulePhase = "Active"
)

// BackupScheduleStatus represents the status of BackupSchedule resource
type BackupScheduleStatus struct {
	// Phase represents the current phase of BackupSchedule
	Phase BackupSchedulePhase `json:"phase"`

	// Reason represents the cause of failure in BackupSchedule
	Reason string `json:"reason,omitempty"`

	// Message represents the cause/action/outcome of BackupSchedule
	Message string `json:"message,omitempty"`

	// VeleroSchedules represents the list of Velero Schedule created by BackupSchedule
	// +nullable
	VeleroSchedules []VeleroScheduleDetails `json:"veleroSchedules,omitempty"`

	// VeleroBackupName represents the name of Velero Backup, created by BackupSchedule,
	// if VeleroScheduleSpec is having empty schedule
	VeleroBackupName string `json:"veleroBackupName,omitempty"`

	// LatestBackupStatus represents the status of latest backup created by BackupSchedule
	LatestBackupStatus LatestBackupStatusDetails `json:"latestBackukpStatus,omitempty"`
}

// VeleroScheduleStatus represents the status of VeleroSchedule
type VeleroScheduleStatus string

const (
	// Active represents the active state of VeleroSchedule
	Active VeleroScheduleStatus = "Active"

	// Deleted represents the deleted VeleroSchedule
	Deleted VeleroScheduleStatus = "Deleted"

	// Erased represents the VeleroSchedule,for which data at remote storage is deleted
	Erased VeleroScheduleStatus = "Erased"
)

// VeleroScheduleDetails represents the information about schedule
type VeleroScheduleDetails struct {
	// ScheduleName represents the velero schedule resource name
	ScheduleName string `json:"scheduleName"`

	// CreationTimestamp defines the time-stamp of velero schedule CR
	// +nullable
	CreationTimestamp *metav1.Time `json:"creationTimestamp,omitempty"`

	// Status represents the velero schedule status
	Status VeleroScheduleStatus `json:"status"`
}

// LatestBackupStatusDetails represents the status of Latest backup
type LatestBackupStatusDetails struct {
	// BackupName represents the name of the backup, which is in-progress or completed recently
	BackupName string `json:"backupName"`

	// Phase represents the given Backup Phase
	Phase string `json:"phase"`

	// StartTimestamp represents the Backup start time
	// +nullable
	StartTimestamp *metav1.Time `json:"startTimestamp,omitempty"`

	// CompletionTimestamp represents the completion time of the given Backup
	// +nullable
	CompletionTimestamp *metav1.Time `json:"completionTimestamp,omitempty"`

	// SnapshotStatus represents the list of snapshotStatus
	// +nullable
	SnapshotStatus []SnapshotStatusDetails `json:"snapshotStatus,omitempty"`
}

// SnapshotType defines the type of snapshot
type SnapshotType string

const (
	// SnapshotTypeRestic represents the restic base snapshot
	SnapshotTypeRestic SnapshotType = "Restic"

	// SnapshotTypeCStor represents the cstor base snapshot
	SnapshotTypeCStor SnapshotType = "cStor"
)

// SnapshotStatusDetails represents the snapshot information and it's status
type SnapshotStatusDetails struct {
	// SnapshotID represents the ID of the snapshot
	SnapshotID string `json:"snapshotID"`

	// Type represents the type of snapshot
	Type SnapshotType `json:"snapshotType"`

	// Phase represents the phase of PodVolumeBackup or CStorBackup resource
	Phase string `json:"phase"`

	// StartTimestamp represents the start time of snapshot
	// +nullable
	StartTimestamp *metav1.Time `json:"startTimestamp,omitempty"`

	// CompletionTimestamp represents the completion time of snapshot
	// +nullable
	CompletionTimestamp *metav1.Time `json:"completionTimestamp,omitempty"`

	// Progress represents the progress of the snapshot
	Progress Progress `json:"progress"`
}

// Progress represents the progress of Snapshot
type Progress struct {
	// TotalBytes represents the total amount of data for the snapshot
	// This value is updated from PodVolumeBackup or CStorBackup resource
	TotalBytes int `json:"totalSize"`

	// BytesDone represents the amount of data snapshotted
	// This value is updated from PodVolumeBackup or CStorBackup resource
	BytesDone int `json:"bytesDone"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupScheduleList represents the list of BackBackupSchedule resource
type BackupScheduleList struct {
	metav1.TypeMeta `json:"inline"`

	// +optional
	metav1.ListMeta `json:"metadata.omitempty"`

	Items []BackupSchedule `json:"items"`
}
