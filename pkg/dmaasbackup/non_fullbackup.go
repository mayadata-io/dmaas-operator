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
)

func (d *dmaasBackup) processNonFullBackupSchedule(dbkp *v1alpha1.DMaaSBackup) error {
	d.logger.Debug("Processing non fullbackup")

	// non full backup schedule can be of two types, schedule or normal backup
	// check for scheduled backup
	if dbkp.Spec.VeleroScheduleSpec.Schedule != "" {
		// dmaasbackup has schedule spec
		// create schedule if not created
		if len(dbkp.Status.VeleroSchedules) == 0 {
			schedule, err := d.createSchedule(dbkp)
			if err != nil {
				d.logger.WithError(err).
					Errorf("failed to create schedule")
				return errors.Wrapf(err, "failed to create schedule")
			}
			// add schedule information to dmaasbackup
			appendVeleroSchedule(dbkp, schedule)
			d.logger.Infof("Schedule=%s created", schedule.Name)
			return nil
		}
		// schedule is already created
		return nil
	}

	// dmaasbackup have information for normal backup
	if dbkp.Status.VeleroBackupName == nil {
		// need to create velero backup
		bkp, err := d.createBackup(dbkp)
		if err != nil {
			d.logger.WithError(err).
				Errorf("failed to create backup")
			return errors.Wrapf(err, "failed to create backup")
		}
		bkpName := bkp.Name
		dbkp.Status.VeleroBackupName = &bkpName
		d.logger.Infof("Backup=%s created", bkpName)
	}

	return nil
}
