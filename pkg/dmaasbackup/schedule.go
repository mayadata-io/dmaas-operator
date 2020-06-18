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
	velerobuilder "github.com/vmware-tanzu/velero/pkg/builder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
)

func (d *dmaasBackup) createSchedule(dbkp *v1alpha1.DMaaSBackup) (*velerov1api.Schedule, error) {
	name := d.generateScheduleName(*dbkp)
	return d.createScheduleUsingName(dbkp, name)
}

func (d *dmaasBackup) createScheduleUsingName(dbkp *v1alpha1.DMaaSBackup, name string) (*velerov1api.Schedule, error) {
	scheduleObj := velerobuilder.ForSchedule(d.veleroNs, name).
		Template(dbkp.Spec.VeleroScheduleSpec.Template).
		CronSchedule(dbkp.Spec.VeleroScheduleSpec.Schedule).
		ObjectMeta(
			velerobuilder.WithLabels(
				// add label using key, value
				v1alpha1.DMaaSBackupLabelKey, dbkp.Name,
			),
		).
		Result()

	return d.scheduleClient.Create(scheduleObj)
}

func (d *dmaasBackup) createBackup(dbkp *v1alpha1.DMaaSBackup) (*velerov1api.Backup, error) {
	name := d.generateBackupName(*dbkp)
	backupObj := velerobuilder.ForBackup(d.veleroNs, name).
		FromSchedule(&velerov1api.Schedule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: d.veleroNs,
				Name:      name,
			},
			Spec: velerov1api.ScheduleSpec{
				Template: dbkp.Spec.VeleroScheduleSpec.Template,
			},
		}).
		ObjectMeta(
			velerobuilder.WithLabels(
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
