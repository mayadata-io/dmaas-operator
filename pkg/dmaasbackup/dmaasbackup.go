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
	clientset "github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	velerov1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	velerov1informer "github.com/vmware-tanzu/velero/pkg/generated/informers/externalversions/velero/v1"
	velerov1lister "github.com/vmware-tanzu/velero/pkg/generated/listers/velero/v1"

	apimachineryclock "k8s.io/apimachinery/pkg/util/clock"
)

// DMaaSBackupper execute operations on dmaasbackup resource
type DMaaSBackupper interface {
	// Execute creates the velero backup for given dmaasbackup
	Execute(obj *v1alpha1.DMaaSBackup, logger logrus.FieldLogger) error
}

type dmaasBackup struct {
	velerons       string
	dmaasClient    clientset.Interface
	backupLister   velerov1lister.BackupNamespaceLister
	pvbLister      velerov1lister.PodVolumeBackupLister
	backupClient   velerov1.BackupInterface
	scheduleClient velerov1.ScheduleInterface
	logger         logrus.FieldLogger
	clock          apimachineryclock.Clock
}

// NewDMaaSBackupper returns the interface to execute operation on dmaasbackup resource
func NewDMaaSBackupper(
	velerons string,
	dmaasClient clientset.Interface,
	veleroClient velerov1.VeleroV1Interface,
	veleroInformer velerov1informer.Interface,
	clock apimachineryclock.Clock,
) DMaaSBackupper {
	return &dmaasBackup{
		velerons:       velerons,
		dmaasClient:    dmaasClient,
		backupClient:   veleroClient.Backups(velerons),
		scheduleClient: veleroClient.Schedules(velerons),
		clock:          clock,
		backupLister:   veleroInformer.Backups().Lister().Backups(velerons),
		pvbLister:      veleroInformer.PodVolumeBackups().Lister(),
	}
}

func (d *dmaasBackup) Execute(obj *v1alpha1.DMaaSBackup, logger logrus.FieldLogger) error {
	var err error

	d.logger = logger

	d.logger.Debug("updating schedule information")

	// check for stale velero schedule created by dmaas-operator
	// This is necessary because there are chances where operator has created
	// required schedule/backup but missed to update the dmaasbackup object, due to error or restart
	err = d.updateScheduleInfo(obj)
	if err != nil {
		return errors.Wrapf(err, "failed to check schedule information")
	}

	switch obj.Spec.State {
	case v1alpha1.DMaaSBackupStatePaused:
		err = d.pauseSchedule(obj)
	default:
		if obj.Spec.PeriodicFullBackupCfg.CronTime != "" {
			err = d.processFullBackupSchedule(obj)
		} else {
			err = d.processNonFullBackupSchedule(obj)
		}
	}

	if err != nil {
		return errors.Wrapf(err, "failed to process dmaasbackup")
	}

	err = d.cleanupOldSchedule(obj)
	if err != nil {
		return errors.Wrapf(err, "failed to perform cleanup for old schedule")
	}

	d.logger.Debug("updating backup information")
	// update latest backup information for dmaasbackup
	err = d.updateBackupInfo(obj)

	return err
}
