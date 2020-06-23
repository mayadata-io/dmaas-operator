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
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
	"github.com/mayadata-io/dmaas-operator/pkg/builder"
	"github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned/fake"
	informers "github.com/mayadata-io/dmaas-operator/pkg/generated/informers/externalversions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apimachineryclock "k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/cache"
)

type fakeBackupper struct {
	mock.Mock
}

func (b *fakeBackupper) Execute(dbkp *v1alpha1.DMaaSBackup, logger logrus.FieldLogger) (bool, error) {
	args := b.Called(dbkp, logger)
	return false, args.Error(0)
}

func (b *fakeBackupper) Delete(dbkp *v1alpha1.DMaaSBackup, logger logrus.FieldLogger) error {
	args := b.Called(dbkp, logger)
	return args.Error(0)
}

func TestProcessBackup(t *testing.T) {
	dbkpbuilder := func(state v1alpha1.DMaasBackupState) *builder.DMaaSBackupBuilder {
		return builder.ForDMaaSBackup("ns", "name").State(state)
	}
	pPhase := func(phase v1alpha1.DMaaSBackupPhase) *v1alpha1.DMaaSBackupPhase {
		return &phase
	}
	tests := []struct {
		name              string
		key               string
		shouldErrored     bool
		dmaasbackup       *v1alpha1.DMaaSBackup
		expectedPhase     *v1alpha1.DMaaSBackupPhase
		backupperErrorMsg string
	}{
		{
			name:          "invalid key",
			key:           "invalid/key/value",
			shouldErrored: false,
		},
		{
			name:          "missing dmaasbackup should return early without an error",
			key:           "missing/dmaasbackup",
			shouldErrored: false,
		},
		{
			name:          "should skip completed dmaasbackup",
			dmaasbackup:   dbkpbuilder("").Phase(v1alpha1.DMaaSBackupPhaseCompleted).Result(),
			shouldErrored: false,
		},
		{
			name:          "should skip paused dmaasbackup",
			dmaasbackup:   dbkpbuilder(v1alpha1.DMaaSBackupStatePaused).Phase(v1alpha1.DMaaSBackupPhasePaused).Result(),
			shouldErrored: false,
		},
		{
			name:          "should changes dmaasbackup phase to in-progress",
			dmaasbackup:   dbkpbuilder("").Result(),
			shouldErrored: false,
			expectedPhase: pPhase(v1alpha1.DMaaSBackupPhaseInProgress),
		},
		{
			name:              "should update reason on error",
			dmaasbackup:       dbkpbuilder("").Result(),
			shouldErrored:     false,
			backupperErrorMsg: "should update reason",
			expectedPhase:     pPhase(v1alpha1.DMaaSBackupPhaseInProgress),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				client   = fake.NewSimpleClientset()
				informer = informers.NewSharedInformerFactory(client, 0)
			)
			var (
				err error
			)

			backupper := new(fakeBackupper)
			d := NewDMaaSBackupController(
				"namespace",
				client,
				informer.Mayadata().V1alpha1().DMaaSBackups(),
				backupper,
				logrus.New(),
				apimachineryclock.NewFakeClock(time.Time{}),
				1,
			)
			c, ok := d.(*dmaasBackupController)
			assert.True(t, ok, "failed to parse dmaasbackup controller")

			bkpper := backupper.On("Execute", mock.Anything, mock.Anything)
			if test.backupperErrorMsg != "" {
				bkpper.Return(errors.New(test.backupperErrorMsg))
			} else {
				bkpper.Return(nil)
			}

			if test.dmaasbackup != nil {
				_ = informer.Mayadata().V1alpha1().DMaaSBackups().Informer().GetStore().Add(test.dmaasbackup)
				_ = client.Tracker().Add(test.dmaasbackup)
				if test.key == "" {
					test.key, err = cache.MetaNamespaceKeyFunc(test.dmaasbackup)
					require.NoError(t, err, "failed to generate key from dmaasbackup err=%v", err)
				}
			}

			_, err = c.processBackup(test.key)
			assert.Equal(t, test.shouldErrored, err != nil, "got error %v", err)

			if test.dmaasbackup != nil {
				updated, err := client.MayadataV1alpha1().
					DMaaSBackups(test.dmaasbackup.Namespace).Get(test.dmaasbackup.Name, metav1.GetOptions{})
				require.NoError(t, err, "failed to get dmaasbackup err=%v", err)
				// verify expectedPhase
				if test.expectedPhase != nil {
					assert.Equal(t, *test.expectedPhase, updated.Status.Phase)
				}

				// verify reason
				if test.backupperErrorMsg != "" {
					assert.Equal(t, test.backupperErrorMsg, updated.Status.Reason)
				}
			}
		})
	}
}

func TestDeleteBackup(t *testing.T) {
	dbkpbuilder := func(deletionTime time.Time) *builder.DMaaSBackupBuilder {
		dtime := &metav1.Time{
			Time: deletionTime,
		}
		return builder.ForDMaaSBackup("ns", "name").DeletionTimeStamp(dtime)
	}
	tests := []struct {
		name                  string
		key                   string
		dmaasbackup           *v1alpha1.DMaaSBackup
		deleteErrorMsg        string
		shouldRemoveFinalizer bool
	}{
		{
			name: "should remove finalizer",
			dmaasbackup: dbkpbuilder(time.Now()).
				Finalizer(v1alpha1.DMaaSFinalizer).
				Result(),
			shouldRemoveFinalizer: true,
		},
		{
			name: "should not remove finalizer",
			dmaasbackup: dbkpbuilder(time.Now()).
				Finalizer("unknown/finalizer").
				Result(),
			shouldRemoveFinalizer: false,
		},
		{
			name: "should fail deletion",
			dmaasbackup: dbkpbuilder(time.Now()).
				Finalizer(v1alpha1.DMaaSFinalizer).
				Result(),
			shouldRemoveFinalizer: false,
			deleteErrorMsg:        "should fail deletion",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				client   = fake.NewSimpleClientset()
				informer = informers.NewSharedInformerFactory(client, 0)
			)
			var (
				err error
			)

			backupper := new(fakeBackupper)
			d := NewDMaaSBackupController(
				"namespace",
				client,
				informer.Mayadata().V1alpha1().DMaaSBackups(),
				backupper,
				logrus.New(),
				apimachineryclock.NewFakeClock(time.Time{}),
				1,
			)
			c, ok := d.(*dmaasBackupController)
			assert.True(t, ok, "failed to parse dmaasbackup controller")

			bkpper := backupper.On("Delete", mock.Anything, mock.Anything)
			if test.deleteErrorMsg != "" {
				bkpper.Return(errors.New(test.deleteErrorMsg))
			} else {
				bkpper.Return(nil)
			}

			_ = informer.Mayadata().V1alpha1().DMaaSBackups().Informer().GetStore().Add(test.dmaasbackup)
			_ = client.Tracker().Add(test.dmaasbackup)
			if test.key == "" {
				test.key, err = cache.MetaNamespaceKeyFunc(test.dmaasbackup)
				require.NoError(t, err, "failed to generate key from dmaasbackup err=%v", err)
			}

			_, err = c.processBackup(test.key)
			require.NoError(t, err, "Failed to process backup returned err=%v", err)

			updated, err := client.MayadataV1alpha1().
				DMaaSBackups(test.dmaasbackup.Namespace).Get(test.dmaasbackup.Name, metav1.GetOptions{})
			require.NoError(t, err, "failed to get dmaasbackup err=%v", err)

			// verify reason
			if test.deleteErrorMsg != "" {
				assert.Equal(t, test.deleteErrorMsg, updated.Status.Reason)
			}
			if test.shouldRemoveFinalizer {
				assert.Equal(t, 0, len(updated.GetFinalizers()))
			}
		})
	}
}
