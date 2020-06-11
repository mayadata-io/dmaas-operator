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
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
	clientset "github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned"
	informers "github.com/mayadata-io/dmaas-operator/pkg/generated/informers/externalversions/mayadata.io/v1alpha1"

	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	apimachineryclock "k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/cache"

	"github.com/mayadata-io/dmaas-operator/pkg/dmaasbackup"
	dmaaslister "github.com/mayadata-io/dmaas-operator/pkg/generated/listers/mayadata.io/v1alpha1"
)

var (
	// backupReconcilePeriod defines the interval at which updated dmaasbackup will be reconciled
	backupReconcilePeriod = 10 * time.Second

	// backupSyncPeriod defines the interval at which all the dmaasbackups will be reconciled
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
	c.reconcilePeriod = backupReconcilePeriod
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
	log := d.logger.WithField("dmaasbackup", key)

	log.Debug("Processing dmaasbackup")

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.WithError(err).Errorf("failed to split key")
		return nil
	}

	original, err := d.lister.DMaaSBackups(ns).Get(name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("dmaasbackup not found")
			return nil
		}
		return errors.Wrapf(err, "failed to get dmaasbackup")
	}

	if yes, msg := shouldProcessDMaaSBackup(*original); !yes {
		log.Debug(msg)
		return nil
	}

	dbkp := original.DeepCopy()

	// initialize dmaas backup's meta data and status
	initializeDMaaSBackupMetaAndStatus(dbkp)

	switch dbkp.DeletionTimestamp {
	case nil:
		err = d.backupper.Execute(dbkp, log)
		if err != nil {
			log.WithError(err).Errorf("failed to execute backup")
		}
	default:
		err = d.backupper.Delete(dbkp, log)
		if err == nil {
			log.Infof("All resources for dmaasbackup deleted")

			// remove finalizer
			dbkp.ObjectMeta = removeDMaaSFinalizer(dbkp.ObjectMeta)
		} else {
			log.WithError(err).Errorf("failed to delete backup resources")
		}
	}

	if err != nil {
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
		if yes, _ := shouldProcessDMaaSBackup(*dbkp); !yes {
			continue
		}
		d.enqueue(dbkp)
	}
}

// shouldProcessDMaaSBackup return true if dbkp is active or needs reconciliation
func shouldProcessDMaaSBackup(dbkp v1alpha1.DMaaSBackup) (shouldProcess bool, msg string) {
	switch dbkp.Spec.State {
	case v1alpha1.DMaaSBackupStateEmpty, v1alpha1.DMaaSBackupStateActive:
		// process only active/new dmaasbackup
		if dbkp.Status.Phase == v1alpha1.DMaaSBackupPhaseCompleted {
			msg = "DMaaSBackup completed, skipping"
			shouldProcess = false
			return
		}
	case v1alpha1.DMaaSBackupStatePaused:
		if dbkp.Status.Phase == v1alpha1.DMaaSBackupPhasePaused {
			msg = "DMaaSBackup paused, skipping"
			shouldProcess = false
			return
		}
		// dmaasbackup is paused but it is not processed by operator
	}
	shouldProcess = true
	return
}

// initializeDMaaSBackupMetaAndStatus update the dmaasbackup metadata and status
// with required value
func initializeDMaaSBackupMetaAndStatus(dbkp *v1alpha1.DMaaSBackup) {
	dbkp.Status.Reason = ""

	if dbkp.Status.Phase == v1alpha1.DMaaSBackupPhaseInProgress {
		// We already initialized this dmaasbackup
		return
	}

	// add dmaas operator finalizer
	dbkp.ObjectMeta = addDMaaSFinalizer(dbkp.ObjectMeta)

	// set dmaasbackup phase InProgress
	dbkp.Status.Phase = v1alpha1.DMaaSBackupPhaseInProgress
}

// addDMaaSFinalizer add dmaas operator related finalizer to given object
func addDMaaSFinalizer(obj metav1.ObjectMeta) metav1.ObjectMeta {
	finalizers := obj.GetFinalizers()
	for _, finalizer := range finalizers {
		if finalizer == v1alpha1.DMaaSFinalizer {
			return obj
		}
	}

	obj.SetFinalizers(append(finalizers, v1alpha1.DMaaSFinalizer))
	return obj
}

// removeDMaaSFinalizer remove dmaas operator related finalizer to given object
func removeDMaaSFinalizer(obj metav1.ObjectMeta) metav1.ObjectMeta {
	finalizers := obj.GetFinalizers()[:0]
	for _, finalizer := range obj.GetFinalizers() {
		if finalizer != v1alpha1.DMaaSFinalizer {
			finalizers = append(finalizers, finalizer)
		}
	}

	obj.SetFinalizers(finalizers)
	return obj
}
