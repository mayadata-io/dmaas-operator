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

	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
	"sort"
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

func getActiveVeleroSchedule(dbkp *v1alpha1.DMaaSBackup) *v1alpha1.VeleroScheduleDetails {
	if len(dbkp.Status.VeleroSchedules) == 0 {
		return nil
	}

	// sort veleroschedule with descending creationtimestamp
	// veleroschedule have only one active schedule
	sort.Sort(sort.Reverse(ScheduleByCreationTimestamp(dbkp.Status.VeleroSchedules)))

	return &dbkp.Status.VeleroSchedules[0]
}

// appendVeleroSchedule add given schedule to scheduledetails with Active status
func appendVeleroSchedule(dbkp *v1alpha1.DMaaSBackup, schedule *velerov1api.Schedule) {
	scheduleDetails := v1alpha1.VeleroScheduleDetails{
		ScheduleName:      schedule.Name,
		CreationTimestamp: &schedule.CreationTimestamp,
		Status:            v1alpha1.Active,
	}
	dbkp.Status.VeleroSchedules = append(dbkp.Status.VeleroSchedules, scheduleDetails)
}
