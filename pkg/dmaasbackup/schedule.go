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
	"github.com/pkg/errors"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	velerobackup "github.com/vmware-tanzu/velero/pkg/backup"
	velerobuilder "github.com/vmware-tanzu/velero/pkg/builder"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
)

func (d *dmaasBackup) createSchedule(dbkp *v1alpha1.DMaaSBackup) (*velerov1api.Schedule, error) {
	name := d.generateScheduleName(*dbkp)
	scheduleObj := velerobuilder.ForSchedule(d.veleroNs, name).
		Template(dbkp.Spec.VeleroScheduleSpec.Template).
		CronSchedule(dbkp.Spec.VeleroScheduleSpec.Schedule).
		ObjectMeta(
			velerobuilder.WithLabels(
				// add label using key, value
				dmaasBackupLabel, dbkp.Name,
			),
		).
		Result()

	return d.scheduleClient.Create(scheduleObj)
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

// updateScheduleInfo checks for relevant velero schedule/backup and
// update the dbkp with schedule/backup details
func (d *dmaasBackup) updateScheduleInfo(dbkp *v1alpha1.DMaaSBackup) error {
	var (
		schedule *velerov1api.Schedule
		bkp      *velerov1api.Backup
		err      error
	)

	// check if dbkp is having config for non-scheduled backup
	if yes := isSchedule(*dbkp); !yes {
		bkp, err = d.backupClient.Get(dbkp.Name, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to update scheduleInfo")
		}
		if bkp.Name == "" {
			// Backup could have deleted due to expiration of TTL
			// TODO: Should we clear backupname from dmaasbackup? and/or recreate?
			return nil
		}
		if dbkp.Status.VeleroBackupName == nil {
			backupName := bkp.Name
			dbkp.Status.VeleroBackupName = &backupName
		}
		return nil
	}

	// check for any queued dummy entry in veleroschedule
	dummySchedule := getDummyVeleroSchedule(dbkp)

	if dummySchedule == nil {
		// we haven't planned any schedule creation
		goto lastschedule_cleanup
	}

	// dummy entry exists in veleroschedule
	// check for any stale schedules

	// for scheduled backup
	schedule, err = d.scheduleClient.Get(dummySchedule.ScheduleName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// no schedule exists with dummy entry's name
			return nil
		}
		return errors.Wrapf(err, "failed to update scheduleInfo")
	}

	// we found stale schedule
	// update dummy entry with schedule details
	updateDummyVeleroSchedule(dbkp, dummySchedule, schedule)

lastschedule_cleanup:
	// check if we have any last schedule
	lastSchedule := getLastVeleroSchedule(dbkp)
	if lastSchedule == nil {
		// there is not last created schedule
		return nil
	}

	if lastSchedule.Status != v1alpha1.Active {
		// last schedule is not active
		return nil
	}

	err = d.scheduleClient.Delete(
		lastSchedule.ScheduleName,
		&metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		d.logger.WithError(err).
			Errorf("failed to delete stale schedule=%s", lastSchedule.ScheduleName)
		return errors.Wrapf(err,
			"failed to delete stale schedule=%s", lastSchedule.ScheduleName)
	}

	lastSchedule.Status = v1alpha1.Deleted
	d.logger.Infof("Schedule=%s deleted", lastSchedule.ScheduleName)
	return nil
}

// cleanupOldSchedule remove the old schedules if fullbackup retention mentioned
// TODO : need to fix for retention count for restic
func (d *dmaasBackup) cleanupOldSchedule(dbkp *v1alpha1.DMaaSBackup) error {
	if dbkp.Spec.PeriodicFullBackupCfg.CronTime == "" {
		return nil
	}

	d.logger.Debug("Processing cleanup for schedule")

	if len(dbkp.Status.VeleroSchedules) == 0 {
		d.logger.Debug("No schedule present in status")
		return nil
	}

	if dbkp.Spec.PeriodicFullBackupCfg.FullBackupRetentionThreshold == 0 {
		d.logger.Debug("Skipping cleanup since FullBackupRetentionThreshold is 0")
		return nil
	}

	// get latest velero schedule
	latestSchedule := getLatestVeleroSchedule(dbkp)
	if latestSchedule.Status != v1alpha1.Active {
		return errors.Errorf("schedule=%s is having state=%s, expected state=%s",
			latestSchedule.ScheduleName,
			latestSchedule.Status,
			v1alpha1.Active,
		)
	}

	// since first schedule is active one, we will skip it
	for index, schedule := range dbkp.Status.VeleroSchedules[1:] {
		if schedule.Status != v1alpha1.Active {
			continue
		}

		// this is stale schedule, need to delete it
		err := d.scheduleClient.Delete(schedule.ScheduleName, &metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			d.logger.Warningf("failed to delete schedule=%s err=%s, will retry in next sync",
				schedule.ScheduleName,
				err,
			)
			continue
		}

		// schedule is deleted, update it with deleted state
		dbkp.Status.VeleroSchedules[index+1].Status = v1alpha1.Deleted
	}

	// delete backups for schedule as per fullBackupRetentionThreshold
	// we need to retain the backups created by current active schedule and
	// last 'FullBackupRetentionThreshold' number of schedules.
	requiredSchedule := dbkp.Spec.PeriodicFullBackupCfg.FullBackupRetentionThreshold + 1

	for index, schedule := range dbkp.Status.VeleroSchedules[requiredSchedule:] {
		if schedule.Status == v1alpha1.Deleted {
			continue
		}

		backupList, err := d.backupLister.List(
			labels.SelectorFromSet(map[string]string{
				dmaasBackupLabel:              dbkp.Name,
				velerov1api.ScheduleNameLabel: schedule.ScheduleName,
			}),
		)
		if err != nil {
			d.logger.Warningf("failed to list backup for schedule=%s err=%s, will retry in next sync",
				schedule.ScheduleName,
				err,
			)
		}
		if len(backupList) == 0 {
			// no backup exists for schedule
			dbkp.Status.VeleroSchedules[requiredSchedule+index].Status = v1alpha1.Erased
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
		// setting status will be handle by next sync when all the backups for this schedules
		// are deleted
	}
	return nil
}

func isSchedule(dbkp v1alpha1.DMaaSBackup) bool {
	return dbkp.Spec.VeleroScheduleSpec.Schedule != ""
}
