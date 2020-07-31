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
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	velerobackup "github.com/vmware-tanzu/velero/pkg/backup"
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
		d.logger.Infof("Schedule=%s created", emptySchedule.ScheduleName)
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

func (d *dmaasBackup) cleanupPeriodicSchedule(dbkp *v1alpha1.DMaaSBackup) error {
	d.logger.Debug("Processing cleanup for periodic schedule")

	// delete backups for schedule as per fullBackupRetentionThreshold.
	// we need to retain the backups created by current active schedule and
	// last 'FullBackupRetentionThreshold' number of schedules.
	requiredSchedule := dbkp.Spec.PeriodicFullBackupCfg.FullBackupRetentionThreshold

	deletedScheduleCount := getDeletedScheduleCount(dbkp)

	if deletedScheduleCount < requiredSchedule {
		d.logger.Debugf("Number of deleted schedules are %v, required %v schedules to trigger cleanup",
			deletedScheduleCount,
			requiredSchedule)
		return nil
	}

	if !dbkp.Spec.PeriodicFullBackupCfg.DisableSuccessfulBackupCheckForRetention {
		// dbkp have number of veleroschedules created according to periodicFullBackup config
		// get veleroschedule index, from dbkp.VeleroSchedules, having successful backup
		successfulScheduleIdx, err := d.getSuccessfulScheduleIdx(dbkp)
		if err != nil {
			d.logger.Debugf("Failed to get successful backup index err=%v",
				err)
			return nil
		}

		if requiredSchedule < successfulScheduleIdx {
			d.logger.Infof("Updating requiredSchedule to %v, to retain successful backup",
				successfulScheduleIdx)

			// we don't have any successful backup in requiredSchedule schedules.
			// To retain schedule having successful backup, set requiredSchedule
			// to successfulScheduleIdx.
			requiredSchedule = successfulScheduleIdx
		}
	}

	defer d.logger.Debug("Cleanup completed for periodic schedule")

	// since we need to retain active schedule, We will perform cleanup from (requiredSchedule+1) schedule
	for index, schedule := range dbkp.Status.VeleroSchedules[requiredSchedule+1:] {
		if schedule.Status != v1alpha1.Deleted {
			// schedule is not deleted, skip it
			continue
		}

		backupList, err := d.listScheduledBackups(dbkp, schedule.ScheduleName)
		if err != nil {
			d.logger.Warningf("failed to list backup for schedule=%s err=%s, will retry in next sync",
				schedule.ScheduleName,
				err,
			)
			continue
		}

		if len(backupList) == 0 {
			// no backup exists for schedule
			dbkp.Status.VeleroSchedules[requiredSchedule+index+1].Status = v1alpha1.Erased
			d.logger.Infof("Status of VeleroSchedule=%s updated to '%v'",
				schedule.ScheduleName,
				v1alpha1.Erased)
			continue
		}

		// create delete request for all backup of the schedule
		for _, bkp := range backupList {
			deleteRequest := velerobackup.NewDeleteBackupRequest(bkp.Name, string(bkp.UID))
			if _, err := d.deleteBackupClient.Create(deleteRequest); err != nil {
				d.logger.Warningf("failed to create deleteRequest for backup=%s err=%s",
					bkp.Name,
					err,
				)
			}
		}

		d.logger.Infof("DeleteBackupRequests created for %s's backups", schedule.ScheduleName)

		// setting status will be handle by next sync when all the backups for this schedules
		// are deleted
	}
	return nil
}

// getSuccessfulScheduleIdx return index for the veleroschedules having successful backup for the given dmaasbackup
// if no successful backup exists then it returns error
func (d *dmaasBackup) getSuccessfulScheduleIdx(dbkp *v1alpha1.DMaaSBackup) (int, error) {
	for idx, schedule := range dbkp.Status.VeleroSchedules {
		if schedule.Status == v1alpha1.Erased {
			continue
		}

		backupList, err := d.listScheduledBackups(dbkp, schedule.ScheduleName)
		if err != nil {
			d.logger.Warningf("failed to list backup for schedule=%s err=%s, will retry in next sync",
				schedule.ScheduleName,
				err,
			)
			continue
		}

		for _, bkp := range backupList {
			if bkp.Status.Phase == velerov1api.BackupPhaseCompleted {
				return idx, nil
			}
		}
	}

	return 0, errors.New("no successful backup present")
}
