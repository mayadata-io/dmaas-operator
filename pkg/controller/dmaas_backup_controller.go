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
	"time"

	clientset "github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned"
	informers "github.com/mayadata-io/dmaas-operator/pkg/generated/informers/externalversions/mayadata.io/v1alpha1"

	"github.com/sirupsen/logrus"
	velero "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	apimachineryclock "k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	backupSyncPeriod = 30 * time.Second
)

type dmaasBackupController struct {
	*controller
	namespace        string
	openebsNamespace string
	veleroNamespace  string
	kubeClient       kubernetes.Interface
	dmaasClient      clientset.Interface
	veleroClient     velero.Interface
	clock            apimachineryclock.Clock
}

// NewDMaaSBackupController returns controller for dmaasBackup resource
func NewDMaaSBackupController(
	namespace, openebsNamespace, veleroNamespace string,
	kubeClient kubernetes.Interface,
	dmaasClient clientset.Interface,
	dmaasBackupInformer informers.DMaaSBackupInformer,
	veleroClient velero.Interface,
	logger logrus.FieldLogger,
	clock apimachineryclock.Clock,
	numWorker int,
) Controller {
	c := &dmaasBackupController{
		controller:       newController("dmaasBackup", logger, numWorker),
		namespace:        namespace,
		openebsNamespace: openebsNamespace,
		veleroNamespace:  veleroNamespace,
		kubeClient:       kubeClient,
		dmaasClient:      dmaasClient,
		veleroClient:     veleroClient,
		clock:            clock,
	}

	c.reconcile = c.processBackup
	c.syncPeriod = backupSyncPeriod
	c.destroy = c.destroyBackup

	dmaasBackupInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.enqueue(obj, qOpSync)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				_ = oldObj
				c.enqueue(newObj, qOpSync)
			},
			DeleteFunc: func(obj interface{}) {
				c.enqueue(obj, qOpDestroy)
			},
		},
	)
	return c
}

func (d *dmaasBackupController) processBackup(key string) error {
	log := d.logger.WithField("key", key).WithField("operation", "sync")

	log.Infof("processing backup")

	return nil
}

func (d *dmaasBackupController) destroyBackup(key string) error {
	log := d.logger.WithField("key", key).WithField("operation", "delete")

	log.Infof("deleting backup")

	return nil
}
