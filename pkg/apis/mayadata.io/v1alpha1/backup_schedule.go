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

// DMaaSBackup represents the backup/schedule resource
type DMaaSBackup struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec DMaaSBackupSpec `json:"spec"`

	// +optional
	Status DMaaSBackupStatus `json:"status"`
}

// DMaasBackupState represents the state of DMaasBackup
type DMaasBackupState string

const (
	// DMaaSBackupStateEmpty means dmaasbackup is active
	DMaaSBackupStateEmpty DMaasBackupState = ""

	// DMaaSBackupStateActive means dmaasbackup is active
	DMaaSBackupStateActive DMaasBackupState = "Active"

	// DMaaSBackupStatePaused means dmaasbackup is paused
	DMaaSBackupStatePaused DMaasBackupState = "Paused"
)

// DMaaSBackupSpec defines the spec for DMaaSBackup resource
type DMaaSBackupSpec struct {
	// State defines if given DMaaSBackup is active or not
	// Default value is Active
	// +optional
	State DMaasBackupState `json:"state,omitempty"`

	// PreBackupActionName name of relevant prebackupaction
	// +optional
	// +nullable
	PreBackupActionName *string `json:"preBackupActionName"`

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

	// FullBackupRetentionThreshold represents the number of full backup needs to be retained
	FullBackupRetentionThreshold int `json:"fullBackupRetentionThreshold"`

	// DisableSuccessfulBackupCheckForRetention disable the checks to retain successful backup
	// from the schedules.
	DisableSuccessfulBackupCheckForRetention bool `json:"disableSuccessfulBackupCheckForRetention,omitempty"`
}

// DMaaSBackupPhase represents the phase of DMaaSBackup
type DMaaSBackupPhase string

const (
	// DMaaSBackupPhaseActive represents the active phase of DMaaSBackup
	DMaaSBackupPhaseActive DMaaSBackupPhase = "Active"

	// DMaaSBackupPhaseInProgress represents the in progress phase of DMaaSBackup
	DMaaSBackupPhaseInProgress DMaaSBackupPhase = "InProgress"

	// DMaaSBackupPhasePaused represents the pause phase of DMaaSBackup
	DMaaSBackupPhasePaused DMaaSBackupPhase = "Paused"

	// DMaaSBackupPhaseCompleted represents the completed phase of DMaaSBackup
	// This is applicable if dmaasbackup is not a scheduled backup
	DMaaSBackupPhaseCompleted DMaaSBackupPhase = "Completed"
)

// DMaaSBackupStatus represents the status of DMaaSBackup resource
type DMaaSBackupStatus struct {
	// Phase represents the current phase of DMaaSBackup
	Phase DMaaSBackupPhase `json:"phase"`

	// Reason represents the cause of failure in DMaaSBackup
	Reason string `json:"reason,omitempty"`

	// Message represents the cause/action/outcome of DMaaSBackup
	Message string `json:"message,omitempty"`

	// VeleroSchedulesUpdatedTimestamp represents the last time veleroschedules got updated
	// +nullable
	VeleroSchedulesUpdatedTimestamp metav1.Time `json:"veleroSchedulesUpdatedTimestamp,omitempty"`

	// VeleroSchedules represents the list of Velero Schedule created by DMaaSBackup
	// +nullable
	VeleroSchedules []VeleroScheduleDetails `json:"veleroSchedules,omitempty"`

	// VeleroBackupName represents the name of Velero Backup, created by DMaaSBackup,
	// if VeleroScheduleSpec is having empty schedule
	VeleroBackupName *string `json:"veleroBackupName,omitempty"`
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

// SnapshotStatusDetails represents the snapshot information and it's status
type SnapshotStatusDetails struct {
	// SnapshotID represents the ID of the snapshot
	SnapshotID string `json:"snapshotID"`

	// PVName represents the PV name on which snapshot is created
	PVName string `json:"pvName"`

	// Type represents the type of snapshot
	Type string `json:"snapshotType"`

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

// DMaaSBackupList represents the list of DMaaSBackup resource
type DMaaSBackupList struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DMaaSBackup `json:"items"`
}
