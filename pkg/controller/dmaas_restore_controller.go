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

	"github.com/sirupsen/logrus"

	clientset "github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned"
	informers "github.com/mayadata-io/dmaas-operator/pkg/generated/informers/externalversions/mayadata.io/v1alpha1"

	velero "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	apimachineryclock "k8s.io/apimachinery/pkg/util/clock"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	restoreSyncPeriod = 30 * time.Second
)

type dmaasRestoreController struct {
	*controller
	namespace        string
	openebsNamespace string
	veleroNamespace  string
	kubeClient       kubernetes.Interface
	dmaasClient      clientset.Interface
	veleroClient     velero.Interface
	clock            apimachineryclock.Clock
}

// NewDMaaSRestoreController returns controller for dmaasrestore resource
func NewDMaaSRestoreController(
	namespace, openebsNamespace, veleroNamespace string,
	kubeClient kubernetes.Interface,
	dmaasClient clientset.Interface,
	dmaasRestoreInformer informers.DMaaSRestoreInformer,
	veleroClient velero.Interface,
	logger logrus.FieldLogger,
	clock apimachineryclock.Clock,
	numWorker int,
) Controller {
	c := &dmaasRestoreController{
		controller:       newController("dmaasRestore", logger, numWorker),
		namespace:        namespace,
		openebsNamespace: openebsNamespace,
		veleroNamespace:  veleroNamespace,
		kubeClient:       kubeClient,
		dmaasClient:      dmaasClient,
		veleroClient:     veleroClient,
		clock:            clock,
	}

	c.reconcile = c.processRestore
	c.syncPeriod = restoreSyncPeriod

	dmaasRestoreInformer.Informer().AddEventHandler(
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

func (d *dmaasRestoreController) processRestore(key string) (bool, error) {
	log := d.logger.WithField("key", key)

	log.Infof("processing restore")

	return false, nil
}
