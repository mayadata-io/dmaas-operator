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
	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (d *dmaasBackup) pauseSchedule(obj *v1alpha1.DMaaSBackup) error {
	d.logger.Debug("Pausing dmaasbackup")

	for _, schedule := range obj.Status.VeleroSchedules {
		d.logger.Infof("Deleting schedule=%s", schedule.ScheduleName)

		err := d.scheduleClient.Delete(schedule.ScheduleName, &metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			d.logger.Errorf("error deleting schedule=%s", schedule.ScheduleName)
			return errors.Wrapf(err, "failed to delete schedule %s", schedule.ScheduleName)
		}
	}

	// update status to paused
	obj.Status.Phase = v1alpha1.DMaaSBackupPhasePaused
	d.logger.Infof("successfully paused")
	return nil
}
