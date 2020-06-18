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

package dmaasbackup

import (
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sort"
	"time"

	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
)

// ScheduleByCreationTimestamp sorts a list of veleroschedule by creation timestamp, using their names as a tie breaker.
type ScheduleByCreationTimestamp []v1alpha1.VeleroScheduleDetails

func (o ScheduleByCreationTimestamp) Len() int      { return len(o) }
func (o ScheduleByCreationTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o ScheduleByCreationTimestamp) Less(i, j int) bool {
	if (*o[i].CreationTimestamp).Equal(o[j].CreationTimestamp) {
		return o[i].ScheduleName < o[j].ScheduleName
	}
	return (*o[i].CreationTimestamp).Before(o[j].CreationTimestamp)
}

// getLatestVeleroSchedule return the latest schedule from status.veleroschedule
func getLatestVeleroSchedule(dbkp *v1alpha1.DMaaSBackup) *v1alpha1.VeleroScheduleDetails {
	if len(dbkp.Status.VeleroSchedules) == 0 {
		return nil
	}

	return &dbkp.Status.VeleroSchedules[0]
}

// getPreviousVeleroSchedule return the last created schedule from status.veleroschedule
func getPreviousVeleroSchedule(dbkp *v1alpha1.DMaaSBackup) *v1alpha1.VeleroScheduleDetails {
	if len(dbkp.Status.VeleroSchedules) < 2 {
		return nil
	}

	return &dbkp.Status.VeleroSchedules[1]
}

// getEmptyQueuedVeleroSchedule return the empty-schedule queued for full backup
func getEmptyQueuedVeleroSchedule(dbkp *v1alpha1.DMaaSBackup) *v1alpha1.VeleroScheduleDetails {
	scheduleLen := len(dbkp.Status.VeleroSchedules)
	if scheduleLen == 0 {
		return nil
	}

	lastEntry := &dbkp.Status.VeleroSchedules[scheduleLen-1]

	if lastEntry.CreationTimestamp.IsZero() && lastEntry.Status == "" {
		return lastEntry
	}

	return nil
}

// addEmptyVeleroSchedule add the velero schedule, with empty timestamp and status, for given schedule name
// and sort dbkp.Status.VeleroSchedules using schedule creationTimestamp in reverse order
func addEmptyVeleroSchedule(dbkp *v1alpha1.DMaaSBackup, scheduleName string) {
	if scheduleName == "" {
		return
	}

	scheduleDetails := v1alpha1.VeleroScheduleDetails{
		ScheduleName:      scheduleName,
		CreationTimestamp: &metav1.Time{Time: time.Time{}},
	}

	dbkp.Status.VeleroSchedules = append(dbkp.Status.VeleroSchedules, scheduleDetails)

	// sort veleroschedule with descending creationtimestamp
	sort.Sort(sort.Reverse(ScheduleByCreationTimestamp(dbkp.Status.VeleroSchedules)))
}

// updateEmptyQueuedVeleroSchedule updates the given empty-queued Entry with schedule details
// and sort dbkp.Status.VeleroSchedules using schedule creationTimestamp in reverse order
func updateEmptyQueuedVeleroSchedule(dbkp *v1alpha1.DMaaSBackup, emptyEntry *v1alpha1.VeleroScheduleDetails, schedule *velerov1api.Schedule) {
	creationTime := schedule.CreationTimestamp
	emptyEntry.CreationTimestamp = &creationTime

	emptyEntry.Status = v1alpha1.Active

	// sort veleroschedule with descending creationtimestamp
	sort.Sort(sort.Reverse(ScheduleByCreationTimestamp(dbkp.Status.VeleroSchedules)))
}
