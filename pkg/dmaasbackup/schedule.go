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
	"github.com/vmware-tanzu/velero/pkg/builder"
	velerobuilder "github.com/vmware-tanzu/velero/pkg/builder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
)

func (d *dmaasBackup) createSchedule(dbkp *v1alpha1.DMaaSBackup) (*velerov1api.Schedule, error) {
	name := d.generateScheduleName(*dbkp)
	scheduleObj := velerobuilder.ForSchedule(d.velerons, name).
		Template(dbkp.Spec.VeleroScheduleSpec.Template).
		CronSchedule(dbkp.Spec.VeleroScheduleSpec.Schedule).
		ObjectMeta(
			builder.WithLabels(
				// add label using key, value
				v1alpha1.DMaaSBackupLabelKey, dbkp.Name,
			),
		).
		Result()

	return d.scheduleClient.Create(scheduleObj)
}

func (d *dmaasBackup) createBackup(dbkp *v1alpha1.DMaaSBackup) (*velerov1api.Backup, error) {
	name := d.generateBackupName(*dbkp)
	backupObj := velerobuilder.ForBackup(d.velerons, name).
		FromSchedule(&velerov1api.Schedule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: d.velerons,
				Name:      name,
			},
			Spec: velerov1api.ScheduleSpec{
				Template: dbkp.Spec.VeleroScheduleSpec.Template,
			},
		}).
		ObjectMeta(
			builder.WithLabels(
				// add label using key, value
				v1alpha1.DMaaSBackupLabelKey, dbkp.Name,
			),
		).
		Result()

	return d.backupClient.Create(backupObj)
}

func (d *dmaasBackup) generateScheduleName(dbkp v1alpha1.DMaaSBackup) string {
	return dbkp.Name + "-" + d.clock.Now().Format("20060102150405")
}

func (d *dmaasBackup) generateBackupName(dbkp v1alpha1.DMaaSBackup) string {
	return dbkp.Name
}

// updateScheduleInfo checks for relevant velero schedule/backup and
// update the dbkp with schedule/backup details
func (d *dmaasBackup) updateScheduleInfo(dbkp *v1alpha1.DMaaSBackup) error {
	// TODO
	return nil
}

// cleanupOldSchedule remove the old schedules if fullbackup retention mentioned
func (d *dmaasBackup) cleanupOldSchedule(dbkp *v1alpha1.DMaaSBackup) error {
	// TODO
	return nil
}
