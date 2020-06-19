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
	"sort"

	"github.com/pkg/errors"
	velerobackup "github.com/vmware-tanzu/velero/pkg/backup"

	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
)

func (d *dmaasBackup) processNonperiodicConfigSchedule(dbkp *v1alpha1.DMaaSBackup) error {
	d.logger.Debug("Processing non fullbackup")
	defer d.logger.Debug("Processing non-fullbackup completed")

	// non full backup schedule can be of two types, schedule or normal backup
	// check for non-scheduled backup
	// dmaasbackup have information for normal backup
	if dbkp.Status.VeleroBackupName == nil &&
		dbkp.Spec.VeleroScheduleSpec.Schedule == "" {
		// need to create velero backup
		bkp, err := d.createBackup(dbkp)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			d.logger.WithError(err).
				Errorf("failed to create backup")
			return errors.Wrapf(err, "failed to create backup")
		}

		bkpName := bkp.Name
		dbkp.Status.VeleroBackupName = &bkpName
		dbkp.Status.Phase = v1alpha1.DMaaSBackupPhaseCompleted
		d.logger.Infof("Backup=%s created", bkpName)
		return nil
	}

	// check for scheduled backup
	if dbkp.Spec.VeleroScheduleSpec.Schedule == "" {
		// not velero schedule spec
		return nil
	}

	// We may have queued empty velero schedule entry with name for schedule creation
	// check if such entry exist
	emptySchedule := getEmptyQueuedVeleroSchedule(dbkp)

	if emptySchedule != nil {
		// we have queued empty schedule for schedule creation in last reconciliation
		// let's create new schedule using name from empty schedule
		// and update it
		newSchedule, err := d.createScheduleUsingName(dbkp, emptySchedule.ScheduleName)
		if err != nil {
			if !apierrors.IsAlreadyExists(err) {
				d.logger.WithError(err).Errorf("failed to create new schedule")
				return err
			}
		}
		updateEmptyQueuedVeleroSchedule(dbkp, emptySchedule, newSchedule)
		return nil
	}

	// no empty schedule entry exists in veleroschedule
	// check if veleroschedule is empty or not
	if len(dbkp.Status.VeleroSchedules) != 0 {
		d.logger.Debug("Velero schedule already created")
		return nil
	}

	// we haven't added empty schedule for schedule creation
	// let's add empty schedule in veleroschedule to reserve schedule name
	newScheduleName := d.generateScheduleName(*dbkp)

	addEmptyVeleroSchedule(dbkp, newScheduleName)

	// set shouldRequeue true so that we can reconcile this object immediately
	d.shouldRequeue = true

	d.logger.Infof("Schedule=%s is queued for creation", newScheduleName)
	return nil
}

type byBackupCreationTimeStamp []*velerov1api.Backup

func (b byBackupCreationTimeStamp) Len() int      { return len(b) }
func (b byBackupCreationTimeStamp) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byBackupCreationTimeStamp) Less(i, j int) bool {
	if (b[i].CreationTimestamp).Equal(&b[j].CreationTimestamp) {
		return b[i].Name < b[j].Name
	}
	return (b[i].CreationTimestamp).Before(&b[j].CreationTimestamp)
}

func (d *dmaasBackup) cleanupNonPeriodicSchedule(dbkp *v1alpha1.DMaaSBackup) error {
	d.logger.Debug("Processing cleanup for non-periodic schedule")

	// get schedule info
	schedule := getLatestVeleroSchedule(dbkp)
	if schedule == nil {
		d.logger.Debug("Schedule is not created")
		return nil
	}

	// check if schedule is active or not
	if schedule.Status != v1alpha1.Active {
		d.logger.Debug("Schedule is not active")
		return nil
	}

	backupList, err := d.backupLister.List(
		labels.SelectorFromSet(map[string]string{
			v1alpha1.DMaaSBackupLabelKey:  dbkp.Name,
			velerov1api.ScheduleNameLabel: schedule.ScheduleName,
		}),
	)

	if err != nil {
		return errors.Wrapf(err, "failed to get list of backup for schedule=%s", schedule.ScheduleName)
	}

	defer d.logger.Debug("Completed cleanup for non-periodic schedule")

	sort.Sort(sort.Reverse(byBackupCreationTimeStamp(backupList)))

	// We need to retain `fullBackupRetentionThreshold` number of backups
	// check for completed backup count
	var completedBackup int

	for _, backup := range backupList {
		switch backup.Status.Phase {
		case "", velerov1api.BackupPhaseNew:
		default:
			completedBackup++
		}
		if completedBackup > dbkp.Spec.PeriodicFullBackupCfg.FullBackupRetentionThreshold {
			deleteRequest := velerobackup.NewDeleteBackupRequest(backup.Name, string(backup.UID))
			if _, err := d.deleteBackupClient.Create(deleteRequest); err != nil {
				d.logger.Warningf("Failed to create deleteRequest for backup=%s err=%s",
					backup.Name,
					err,
				)
			}
			d.logger.Infof("Delete request created for backup=%s", backup.Name)
		}
	}

	return nil
}
