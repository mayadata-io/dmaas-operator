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

// getLastVeleroSchedule return the last created schedule from status.veleroschedule
func getLastVeleroSchedule(dbkp *v1alpha1.DMaaSBackup) *v1alpha1.VeleroScheduleDetails {
	if len(dbkp.Status.VeleroSchedules) < 2 {
		return nil
	}

	return &dbkp.Status.VeleroSchedules[1]
}

// appendVeleroSchedule add given schedule to scheduledetails with Active status
// and sort dbkp.Status.VeleroSchedules using schedule creationTimestamp in reverse order
func appendVeleroSchedule(dbkp *v1alpha1.DMaaSBackup, schedule *velerov1api.Schedule) {
	scheduleDetails := v1alpha1.VeleroScheduleDetails{
		ScheduleName:      schedule.Name,
		CreationTimestamp: &schedule.CreationTimestamp,
		Status:            v1alpha1.Active,
	}
	dbkp.Status.VeleroSchedules = append(dbkp.Status.VeleroSchedules, scheduleDetails)

	// sort veleroschedule with descending creationtimestamp
	sort.Sort(sort.Reverse(ScheduleByCreationTimestamp(dbkp.Status.VeleroSchedules)))
}

// getVeleroScheduleDetails return schedule details from dmaasbackup status for the given schedule
func getVeleroScheduleDetails(scheduleName string, dbkp *v1alpha1.DMaaSBackup) *v1alpha1.VeleroScheduleDetails {
	for index, scheduleDetail := range dbkp.Status.VeleroSchedules {
		if scheduleDetail.ScheduleName == scheduleName {
			return &dbkp.Status.VeleroSchedules[index]
		}
	}
	return nil
}

// getDummyVeleroSchedule return the dummy schedule queued for full backup
func getDummyVeleroSchedule(dbkp *v1alpha1.DMaaSBackup) *v1alpha1.VeleroScheduleDetails {
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

// addDummyVeleroSchedule add the velero schedule with empty timestamp and status
// and sort dbkp.Status.VeleroSchedules using schedule creationTimestamp in reverse order
func addDummyVeleroSchedule(dbkp *v1alpha1.DMaaSBackup, name string) {
	if name == "" {
		return
	}

	scheduleDetails := v1alpha1.VeleroScheduleDetails{
		ScheduleName:      name,
		CreationTimestamp: &metav1.Time{Time: time.Time{}},
	}

	dbkp.Status.VeleroSchedules = append(dbkp.Status.VeleroSchedules, scheduleDetails)

	// sort veleroschedule with descending creationtimestamp
	sort.Sort(sort.Reverse(ScheduleByCreationTimestamp(dbkp.Status.VeleroSchedules)))
}

// updateDummyVeleroSchedule updates the given dummyEntry with schedule details
// and sort dbkp.Status.VeleroSchedules using schedule creationTimestamp in reverse order
func updateDummyVeleroSchedule(dbkp *v1alpha1.DMaaSBackup, dummyEntry *v1alpha1.VeleroScheduleDetails, schedule *velerov1api.Schedule) {
	creationTime := schedule.CreationTimestamp
	dummyEntry.CreationTimestamp = &creationTime

	dummyEntry.Status = v1alpha1.Active

	// sort veleroschedule with descending creationtimestamp
	sort.Sort(sort.Reverse(ScheduleByCreationTimestamp(dbkp.Status.VeleroSchedules)))
}
