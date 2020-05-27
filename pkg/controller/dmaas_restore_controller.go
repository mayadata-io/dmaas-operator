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

func (d *dmaasRestoreController) processRestore(key string) error {
	log := d.logger.WithField("key", key)

	log.Infof("processing restore")

	return nil
}
