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
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	velerov1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	velerov1informer "github.com/vmware-tanzu/velero/pkg/generated/informers/externalversions/velero/v1"
	velerov1lister "github.com/vmware-tanzu/velero/pkg/generated/listers/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
	clientset "github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned"

	apimachineryclock "k8s.io/apimachinery/pkg/util/clock"
)

// DMaaSBackupper execute operations on dmaasbackup resource
type DMaaSBackupper interface {
	// Execute creates the velero backup for given dmaasbackup
	Execute(obj *v1alpha1.DMaaSBackup, logger logrus.FieldLogger) (bool, error)

	// Delete perform the cleanup required as part of dmaasbackup deletion
	Delete(obj *v1alpha1.DMaaSBackup, logger logrus.FieldLogger) error
}

type dmaasBackup struct {
	veleroNs           string
	dmaasClient        clientset.Interface
	backupLister       velerov1lister.BackupNamespaceLister
	pvbLister          velerov1lister.PodVolumeBackupLister
	backupClient       velerov1.BackupInterface
	scheduleClient     velerov1.ScheduleInterface
	deleteBackupClient velerov1.DeleteBackupRequestInterface
	logger             logrus.FieldLogger
	clock              apimachineryclock.Clock

	// shouldRequeue should be set if immediate reconciliation is needed for object
	shouldRequeue bool
}

// NewDMaaSBackupper returns the interface to execute operation on dmaasbackup resource
func NewDMaaSBackupper(
	veleroNs string,
	dmaasClient clientset.Interface,
	veleroClient velerov1.VeleroV1Interface,
	veleroInformer velerov1informer.Interface,
	clock apimachineryclock.Clock,
) DMaaSBackupper {
	return &dmaasBackup{
		veleroNs:           veleroNs,
		dmaasClient:        dmaasClient,
		backupClient:       veleroClient.Backups(veleroNs),
		scheduleClient:     veleroClient.Schedules(veleroNs),
		deleteBackupClient: veleroClient.DeleteBackupRequests(veleroNs),
		clock:              clock,
		backupLister:       veleroInformer.Backups().Lister().Backups(veleroNs),
		pvbLister:          veleroInformer.PodVolumeBackups().Lister(),
		shouldRequeue:      false,
	}
}

func (d *dmaasBackup) Execute(obj *v1alpha1.DMaaSBackup, logger logrus.FieldLogger) (bool, error) {
	var err error

	d.logger = logger

	d.logger.Debug("Executing dmaasbackup")
	defer d.logger.Debug("Execution of dmaasbackup completed")

	// reset shouldRequeue
	d.shouldRequeue = false

	// check for stale velero schedule created by dmaas-operator
	// This is necessary because there are chances where operator has created
	// required schedule/backup but missed to update the dmaasbackup object, due to error or restart
	err = d.updateScheduleInfo(obj)
	if err != nil {
		return d.shouldRequeue, errors.Wrapf(err, "failed to check schedule information")
	}

	if obj.Spec.PeriodicFullBackupCfg.CronTime != "" {
		err = d.processPeriodicConfigSchedule(obj)
	} else {
		err = d.processNonperiodicConfigSchedule(obj)
	}

	return d.shouldRequeue, err
}

func (d *dmaasBackup) Delete(obj *v1alpha1.DMaaSBackup, logger logrus.FieldLogger) error {
	d.logger = logger

	d.logger.Debug("deleting schedule information")

	// since dmaasbackup object is being deleted, we will skip updating
	// stale schedules in status.veleroschedule
	// We will fetch the active velero schedule from etcd and delete it

	scheduleList, err := d.scheduleClient.List(
		metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v1alpha1.DMaaSBackupLabelKey, obj.Name)},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to get velero schedules created by dmaas operator")
	}

	var deleteErr error

	// if any error happened during deletion we will log it and
	// we will return only last error, if any, occurred during schedule deletion
	for _, schedule := range scheduleList.Items {
		err := d.scheduleClient.Delete(schedule.Name, &metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			d.logger.Warningf("failed to delete schedule=%s err=%s", schedule.Name, err)
			deleteErr = err
			continue
		}
		d.logger.Infof("schedule=%s deleted", schedule.Name)
	}

	return deleteErr
}
