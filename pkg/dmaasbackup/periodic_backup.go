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

	"github.com/pkg/errors"
	"github.com/robfig/cron"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
)

func (d *dmaasBackup) processPeriodicConfigSchedule(obj *v1alpha1.DMaaSBackup) error {
	d.logger.Debug("Processing fullbackup")
	defer d.logger.Debug("Processing fullbackup completed")

	// We may have queued empty velero schedule entry with name for schedule creation
	// check if such entry exist
	emptySchedule := getEmptyQueuedVeleroSchedule(obj)
	if emptySchedule != nil {
		// we have queued empty schedule for full backup
		// let's create new schedule using name from empty schedule
		// and update it
		newSchedule, err := d.createScheduleUsingName(obj, emptySchedule.ScheduleName)
		if err != nil {
			if !apierrors.IsAlreadyExists(err) {
				d.logger.WithError(err).Errorf("failed to create new schedule")
				return err
			}
		}
		updateEmptyQueuedVeleroSchedule(obj, emptySchedule, newSchedule)

		lastSchedule := getPreviousVeleroSchedule(obj)
		if lastSchedule == nil {
			// there is no last created schedule
			return nil
		}

		// Delete the old active schedule
		// Note: if any backups pending for this schedule exists then
		// it won't be impacted by this operator and will be executed in queue
		err = d.scheduleClient.Delete(
			lastSchedule.ScheduleName,
			&metav1.DeleteOptions{},
		)
		if err != nil && !apierrors.IsNotFound(err) {
			d.logger.WithError(err).
				Errorf("failed to delete schedule=%s", lastSchedule.ScheduleName)

			return errors.Wrapf(err,
				"failed to delete schedule=%s", lastSchedule.ScheduleName)
		}
		lastSchedule.Status = v1alpha1.Deleted
		d.logger.Infof("Schedule=%s deleted", lastSchedule.ScheduleName)
		return nil
	}

	cr, err := cron.ParseStandard(obj.Spec.PeriodicFullBackupCfg.CronTime)
	if err != nil {
		return errors.Wrapf(err, "failed to parse cronTime")
	}

	// we will use the latest schedule from status.veleroschedule
	// as an active schedule to calculate the due for a full backup
	activeSchedule := getLatestVeleroSchedule(obj)
	isDue, nextDue := getNextDue(cr, activeSchedule, d.clock.Now())
	if !isDue {
		d.logger.Debugf("Full Backup is not due, next due after %s", nextDue.String())
		return nil
	}

	d.logger.Debug("Updating next schedule name in veleroschedules for full backup")

	// we will add new entry in status.VeleroSchedules with empty status and creationtimestamp
	// so that in next reconciliation we can create schedule using that entry, and delete the
	// last active schedule
	newScheduleName := d.generateScheduleName(*obj)

	addEmptyVeleroSchedule(obj, newScheduleName)

	// set shouldRequeue true so that we can reconcile this object immediately
	d.shouldRequeue = true

	d.logger.Infof("Schedule=%s is queued for creation", newScheduleName)
	return nil
}

func getNextDue(cr cron.Schedule, schedule *v1alpha1.VeleroScheduleDetails, now time.Time) (bool, time.Duration) {
	var lastSync time.Time
	if schedule != nil && schedule.CreationTimestamp != nil {
		lastSync = schedule.CreationTimestamp.Time
	}

	nextSync := cr.Next(lastSync)
	return now.After(nextSync), nextSync.Sub(now)
}
