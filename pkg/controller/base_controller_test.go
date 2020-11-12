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
	"testing"
	"time"

	"github.com/mayadata-io/dmaas-operator/pkg/builder"
	"github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned/fake"
	informers "github.com/mayadata-io/dmaas-operator/pkg/generated/informers/externalversions"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apimachineryclock "k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/watch"
	clientgotest "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestRun(t *testing.T) {
	initExecuted := false

	dmaasbackup := builder.ForDMaaSBackup("ns", "name").Result()

	client := fake.NewSimpleClientset()

	w := watch.NewFake()
	defer w.Stop()
	client.PrependWatchReactor("dmaasbackups", clientgotest.DefaultWatchReactor(w, nil))

	informer := informers.NewSharedInformerFactory(client, 0)

	output := make(chan string)

	d := NewDMaaSBackupController(
		"ns",
		client,
		informer.Mayadata().V1alpha1().DMaaSBackups(),
		nil,
		logrus.New(),
		apimachineryclock.NewFakeClock(time.Time{}),
		1,
	)
	c, ok := d.(*dmaasBackupController)
	assert.True(t, ok, "failed to parse dmaasbackup controller")

	c.reconcile = func(key string) (bool, error) {
		output <- key
		return false, nil
	}
	c.sync = nil
	c.init = func() error {
		initExecuted = true
		return nil
	}

	expectedKey, err := cache.MetaNamespaceKeyFunc(dmaasbackup)
	require.NoError(t, err, "failed to generate key from dmaasbackup err=%v", err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go informer.Start(ctx.Done())
	go func() {
		_ = c.Run(ctx)
	}()

	// verify AddFunc
	w.Add(dmaasbackup)
	select {
	case <-ctx.Done():
		t.Fatal("test timed out for AddFunc")
	case key := <-output:
		assert.Equal(t, expectedKey, key)
	}

	// verify UpdateFunc
	dmaasbackup.Status.Message = "new message"
	w.Modify(dmaasbackup)

	select {
	case <-ctx.Done():
		t.Fatal("test timed out for UpdateFunc")
	case key := <-output:
		assert.Equal(t, expectedKey, key)
	}

	// verify DeleteFunc
	w.Delete(dmaasbackup)

	select {
	case <-ctx.Done():
		t.Fatal("test timed out for DeleteFunc")
	case key := <-output:
		assert.Equal(t, expectedKey, key)
	}
	assert.Equal(t, true, initExecuted)
}

func TestSync(t *testing.T) {
	dmaasbackup := builder.ForDMaaSBackup("ns", "name").Result()

	client := fake.NewSimpleClientset()
	informer := informers.NewSharedInformerFactory(client, 0)

	output := make(chan string)

	d := NewDMaaSBackupController(
		"ns",
		client,
		informer.Mayadata().V1alpha1().DMaaSBackups(),
		nil,
		logrus.New(),
		apimachineryclock.NewFakeClock(time.Time{}),
		1,
	)
	c, ok := d.(*dmaasBackupController)
	assert.True(t, ok, "failed to parse dmaasbackup controller")

	c.reconcile = func(key string) (bool, error) {
		output <- key
		return false, nil
	}

	expectedKey, err := cache.MetaNamespaceKeyFunc(dmaasbackup)
	require.NoError(t, err, "failed to generate key from dmaasbackup err=%v", err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go informer.Start(ctx.Done())

	_ = client.Tracker().Add(dmaasbackup)
	go func() {
		_ = c.Run(ctx)
	}()

	// verify syncFunc
	select {
	case <-ctx.Done():
		t.Fatal("test timed out for syncFunc")
	case key := <-output:
		assert.Equal(t, expectedKey, key)
	}
}
