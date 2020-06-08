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
	"time"

	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (d *dmaasBackup) processFullBackupSchedule(obj *v1alpha1.DMaaSBackup) error {
	d.logger.Debug("Processing fullbackup")

	cr, err := cron.ParseStandard(obj.Spec.PeriodicFullBackupCfg.CronTime)
	if err != nil {
		return errors.Wrapf(err, "failed to parse cronTime")
	}

	activeSchedule := getActiveVeleroSchedule(obj)

	isDue, nextDue := getNextDue(cr, activeSchedule, d.clock.Now())
	if !isDue {
		d.logger.Debugf("Full Backup is not due, next due after %s", nextDue.String())
		return nil
	}

	d.logger.Debug("Creating full backup")

	// before creating a new schedule, we will delete the old schedule
	// upon successful deletion we will create a new schedule
	if activeSchedule != nil {
		err = d.scheduleClient.Delete(
			activeSchedule.ScheduleName,
			&metav1.DeleteOptions{},
		)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				d.logger.WithError(err).
					Errorf("failed to delete schedule=%s", activeSchedule.ScheduleName)
				return errors.Wrapf(err,
					"failed to delete schedule=%s", activeSchedule.ScheduleName)
			}
		}
		activeSchedule.Status = v1alpha1.Deleted
		d.logger.Infof("Schedule=%s deleted", activeSchedule.ScheduleName)
	}

	// create a new schedule
	newSchedule, err := d.createSchedule(obj)
	if err != nil {
		d.logger.WithError(err).Errorf("failed to create new schedule")
		return err
	}
	appendVeleroSchedule(obj, newSchedule)
	d.logger.Infof("Schedule=%s created", newSchedule.Name)
	return nil
}

func getNextDue(cr cron.Schedule, schedule *v1alpha1.VeleroScheduleDetails, now time.Time) (bool, time.Duration) {
	var lastSync time.Time
	if schedule != nil && schedule.CreationTimestamp != nil {
		lastSync = schedule.CreationTimestamp.Time
	}

	nextSync := cr.Next(lastSync)
	return now.After(nextSync), nextSync.Sub(lastSync)
}
