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

package controller

import (
	"encoding/json"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
	clientset "github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned"
	informers "github.com/mayadata-io/dmaas-operator/pkg/generated/informers/externalversions/mayadata.io/v1alpha1"
	"github.com/pkg/errors"

	"github.com/mayadata-io/dmaas-operator/pkg/dmaasbackup"
	dmaaslister "github.com/mayadata-io/dmaas-operator/pkg/generated/listers/mayadata.io/v1alpha1"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	apimachineryclock "k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/cache"
)

var (
	backupSyncPeriod = 30 * time.Second
)

type dmaasBackupController struct {
	*controller
	namespace   string
	dmaasClient clientset.Interface
	lister      dmaaslister.DMaaSBackupLister
	backupper   dmaasbackup.DMaaSBackupper
	clock       apimachineryclock.Clock
}

// NewDMaaSBackupController returns controller for dmaasBackup resource
func NewDMaaSBackupController(
	namespace string,
	dmaasClient clientset.Interface,
	dmaasBackupInformer informers.DMaaSBackupInformer,
	backupper dmaasbackup.DMaaSBackupper,
	logger logrus.FieldLogger,
	clock apimachineryclock.Clock,
	numWorker int,
) Controller {
	c := &dmaasBackupController{
		controller:  newController("dmaasBackup", logger, numWorker),
		namespace:   namespace,
		dmaasClient: dmaasClient,
		backupper:   backupper,
		lister:      dmaasBackupInformer.Lister(),
		clock:       clock,
	}

	c.reconcile = c.processBackup
	c.syncPeriod = backupSyncPeriod
	c.sync = c.syncAll

	dmaasBackupInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.enqueue(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				_ = oldObj
				c.enqueue(newObj)
			},
			DeleteFunc: func(obj interface{}) {
				c.enqueue(obj)
			},
		},
	)
	return c
}

func (d *dmaasBackupController) processBackup(key string) error {
	log := d.logger.WithField("key", key)

	log.Debug("Processing dmaasbackup")

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.WithError(err).Errorf("failed to split key")
		return nil
	}

	original, err := d.lister.DMaaSBackups(ns).Get(name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "failed to get dmaasbackup")
	}

	switch original.Spec.State {
	case v1alpha1.DMaaSBackupStateEmpty, v1alpha1.DMaaSBackupStateActive:
		// process only active/new dmaasbackup
		if original.Status.Phase == v1alpha1.DMaaSBackupPhaseCompleted {
			log.Debug("DMaaSBackup completed, skipping")
			return nil
		}
	case v1alpha1.DMaaSBackupStatePaused:
		if original.Status.Phase == v1alpha1.DMaaSBackupPhasePaused {
			log.Debug("DMaaSBackup paused, skipping")
			return nil
		}
		// dmaasbackup is paused but it is not processed by operator
	}

	dbkp := original.DeepCopy()
	dbkp.Status.Reason = ""

	logger := logrus.New()
	logger.Out, logger.Level = log.Logger.Out, log.Logger.Level
	logger.SetFormatter(new(logrus.JSONFormatter))

	// set dmaasbackup phase InProgress
	dbkp.Status.Phase = v1alpha1.DMaaSBackupPhaseInProgress

	dbkplogger := logger.WithField("dmaasbackup", key)
	err = d.backupper.Execute(dbkp, dbkplogger)
	if err != nil {
		log.WithError(err).Errorf("failed to execute backup")
		dbkp.Status.Reason = err.Error()
	}

	log.Debug("Updating dmaasbackup")

	_, err = patchBackup(original, dbkp, d.dmaasClient)
	if err != nil {
		log.WithError(err).Error("failed to update dmaasbackup")
	}
	return nil
}

func patchBackup(original, updated *v1alpha1.DMaaSBackup, client clientset.Interface) (*v1alpha1.DMaaSBackup, error) {
	origBytes, err := json.Marshal(original)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal original dmaasbackup")
	}

	updatedBytes, err := json.Marshal(updated)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal updated dmaasbackup")
	}

	patchBytes, err := jsonpatch.CreateMergePatch(origBytes, updatedBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create json merge patch for dmaasbackup")
	}

	return client.MayadataV1alpha1().
		DMaaSBackups(original.Namespace).
		Patch(original.Name, types.MergePatchType, patchBytes)
}

// syncAll fetched all the dmaasbackup and process it
func (d *dmaasBackupController) syncAll() {
	d.logger.Debug("syncing all dmaasbackup resources")

	dbkps, err := d.lister.DMaaSBackups(d.namespace).List(labels.Everything())
	if err != nil {
		d.logger.WithError(err).Error("failed to list all the resources")
		return
	}

	for _, dbkp := range dbkps {
		d.enqueue(dbkp)
	}
}
