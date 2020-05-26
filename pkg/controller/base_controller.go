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
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type controller struct {
	// name of the controller
	name string

	// numWorker number of worker for controller
	numWorker int

	// workqueue for controller
	workQueue workqueue.RateLimitingInterface

	// logger for controller logging
	logger logrus.FieldLogger

	// init function to be executed before reconcile
	init func() error

	// reconcile is main function, which process the event
	reconcile func(key string) error

	// destroy is function to be executed for deletion of object
	destroy func(key string) error

	// controller syncPeriod
	syncPeriod time.Duration

	cacheSyncWaiters []cache.InformerSynced
}

// queueOperation represents the type of operation on workQueueLoad
type queueOperation string

// Different type of operations on the workQueueLoad
const (
	qOpDestroy queueOperation = "destroy"
	qOpSync    queueOperation = "sync"
)

// workQueueLoad is for storing the key and type of operation before entering workqueue
type workQueueLoad struct {
	key       string // Key is the name of the object
	operation queueOperation
}

func newController(name string, logger logrus.FieldLogger, numWorker int) *controller {
	return &controller{
		name:      name,
		logger:    logger.WithField("controller", name),
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name),
		numWorker: numWorker,
	}
}

func (c *controller) Run(ctx context.Context) error {
	if c.reconcile == nil {
		return errors.Errorf("no reconcile function provided")
	}

	if c.init != nil {
		err := c.init()
		if err != nil {
			c.logger.WithError(err).Errorf("init returned error")
			return err
		}
	}

	c.logger.Infof("Starting controller")
	defer c.logger.Infof("Shutting down controller")

	// wait for caches to sync
	if len(c.cacheSyncWaiters) > 0 {
		c.logger.Infof("Waiting for caches to sync")
		if !cache.WaitForCacheSync(ctx.Done(), c.cacheSyncWaiters...) {
			return errors.New("timed out waiting for caches to sync")
		}
		c.logger.Infof("All caches synced")
	}

	// Waitgroup for starting controller goroutines.
	var wg sync.WaitGroup

	wg.Add(c.numWorker)
	for i := 0; i < c.numWorker; i++ {
		go func() {
			wait.Until(c.runWorker, c.syncPeriod, ctx.Done())
			wg.Done()
		}()
	}

	<-ctx.Done()
	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *controller) processNextWorkItem() bool {
	key, shutdown := c.workQueue.Get()

	if shutdown {
		return false
	}

	// always call Done on this key so the workqueue knows we have finished
	// processing this item. If any error occurs we re-add this key to workqueue
	// with rate-limiting.
	// if we don't want to process this key further then call Forget for this key
	defer c.workQueue.Done(key)

	qObj, ok := key.(workQueueLoad)
	if !ok {
		c.workQueue.Forget(key)
	}

	var err error

	switch qObj.operation {
	case qOpDestroy:
		if c.destroy != nil {
			err = c.destroy(qObj.key)
		}
	case qOpSync:
		err = c.reconcile(qObj.key)
	}

	if err == nil {
		c.workQueue.Forget(key)
		return true
	}

	c.logger.WithError(err).
		WithField("key", qObj.key).
		WithField("operation", qObj.operation).
		Error("Error handling key, re-adding it")
	c.workQueue.AddRateLimited(key)
	return true
}

func (c *controller) enqueue(obj interface{}, op queueOperation) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		c.logger.WithError(errors.WithStack(err)).
			WithField("operation", op).
			Error("Error fetching queue from object, skipping it")
		return
	}

	c.workQueue.Add(workQueueLoad{
		key:       key,
		operation: op,
	})
}