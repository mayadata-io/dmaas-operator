package controller

import (
	"time"

	clientset "github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned"
	informers "github.com/mayadata-io/dmaas-operator/pkg/generated/informers/externalversions/mayadata.io/v1alpha1"
	"github.com/sirupsen/logrus"

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

func (p *preBackupActionController) processPreBackupAction(key string) error {
	log := p.logger.WithField("key", key)

	log.Infof("processing prebackupaction")

	return nil
}
