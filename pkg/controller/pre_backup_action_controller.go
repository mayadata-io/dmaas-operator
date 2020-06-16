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

	apimachineryclock "k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	preBackupActionSyncPeriod = 30 * time.Second
)

type preBackupActionController struct {
	*controller
	namespace   string
	kubeClient  kubernetes.Interface
	dmaasClient clientset.Interface
	clock       apimachineryclock.Clock
}

// NewPreBackupActionController returns controller for prebackupaction resource
func NewPreBackupActionController(
	namespace string,
	kubeClient kubernetes.Interface,
	dmaasClient clientset.Interface,
	preBackupActionInformer informers.PreBackupActionInformer,
	logger logrus.FieldLogger,
	clock apimachineryclock.Clock,
	numWorker int,
) Controller {
	c := &preBackupActionController{
		controller:  newController("prebackupaction", logger, numWorker),
		namespace:   namespace,
		kubeClient:  kubeClient,
		dmaasClient: dmaasClient,
		clock:       clock,
	}

	c.reconcile = c.processPreBackupAction
	c.syncPeriod = preBackupActionSyncPeriod

	preBackupActionInformer.Informer().AddEventHandler(
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

func (p *preBackupActionController) processPreBackupAction(key string) (bool, error) {
	log := p.logger.WithField("key", key)

	log.Infof("processing prebackupaction")

	return false, nil
}
