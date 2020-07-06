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

package dmaasbackup

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	velerobuilder "github.com/vmware-tanzu/velero/pkg/builder"
	velerofake "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/fake"
	velerov1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	veleroinformer "github.com/vmware-tanzu/velero/pkg/generated/informers/externalversions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	apimachineryclock "k8s.io/apimachinery/pkg/util/clock"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
	"github.com/mayadata-io/dmaas-operator/pkg/builder"
	"github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned/fake"
)

const (
	// namespace for dmaas-operator resources
	dmaasNamespace = "ns"

	// namespace for velero resources
	veleroNamespace = "ns"
)

func newTestScheduleSpec() velerov1api.ScheduleSpec {
	return velerov1api.ScheduleSpec{
		Schedule: "*/2 * * * *",
	}
}

func veleroScheduleDetails(
	t *testing.T,
	scheduleTime string,
	status v1alpha1.VeleroScheduleStatus,
	isReserved bool,
) v1alpha1.VeleroScheduleDetails {
	testTime, err := time.Parse("2006-01-02 15:04:05", scheduleTime)
	require.NoError(t, err, "Failed to parse schedule time: %v", err)

	var creationTime metav1.Time

	if !isReserved {
		creationTime.Time = testTime
	} else {
		// for reserved veleroschedule, set status empty explicitly
		status = ""
	}

	return v1alpha1.VeleroScheduleDetails{
		ScheduleName:      "name-" + testTime.Format("20060102150405"),
		CreationTimestamp: &creationTime,
		Status:            status,
	}
}

func compareVeleroSchedule(given, expected []v1alpha1.VeleroScheduleDetails) error {
	sort.Sort(sort.Reverse(ScheduleByCreationTimestamp(expected)))
	if (given == nil) != (expected == nil) {
		return errors.New("either of schedule is empty")
	}

	if len(given) != len(expected) {
		return errors.Errorf("Length mismatched given-len=%d expected-len=%d", len(given), len(expected))
	}

	for i := range given {
		if !reflect.DeepEqual(given[i], expected[i]) {
			return errors.Errorf("given schedule=%v, expected=%v", given[i], expected[i])
		}
	}

	return nil
}

func TestProcessPeriodicConfigSchedule(t *testing.T) {
	dbkpbuilder := func() *builder.DMaaSBackupBuilder {
		return builder.ForDMaaSBackup(dmaasNamespace, "name")
	}

	generateSchedule := func(creationTime, cronSchedule string) velerov1api.Schedule {
		testTime, err := time.Parse("2006-01-02 15:04:05", creationTime)
		require.NoError(t, err, "Failed to parse schedule time: %v", err)

		schedule := velerobuilder.ForSchedule(veleroNamespace, "name-"+testTime.Format("20060102150405")).
			CronSchedule(cronSchedule).
			Result()
		schedule.CreationTimestamp = metav1.Time{Time: testTime}
		return *schedule
	}

	tests := []struct {
		name                   string
		backup                 *v1alpha1.DMaaSBackup
		fakeTime               string
		shouldErrored          bool
		expectedVeleroSchedule []v1alpha1.VeleroScheduleDetails // compare only status and name, excluding reserved one
		shouldRequeue          bool
		existingSchedules      []velerov1api.Schedule
		scheduleCreateError    bool
		scheduleDeleteError    bool
	}{
		{
			name: "should return error on invalid crontime",
			backup: dbkpbuilder().
				PeriodicConfig("*/8", 1, false).
				Schedule(newTestScheduleSpec()).
				Result(),
			fakeTime:      "2020-06-20 06:20:00",
			shouldErrored: true,
		},
		{
			name: "should reserve name if veleroschedule is empty",
			backup: dbkpbuilder().
				PeriodicConfig("*/8 * * * *", 1, false).
				Schedule(newTestScheduleSpec()).
				Result(),
			fakeTime: "2020-06-20 06:20:00",
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", "", true),
			},
			shouldRequeue: true,
		},
		{
			name: "should not reserve name if schedule not due",
			backup: dbkpbuilder().
				PeriodicConfig("*/8 * * * *", 1, false).
				Schedule(newTestScheduleSpec()).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				Result(),
			fakeTime: "2020-06-20 06:21:00",
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
			},
		},
		{
			name: "should create schedule if schedule name is reserved",
			backup: dbkpbuilder().
				PeriodicConfig("*/8 * * * *", 1, false).
				Schedule(newTestScheduleSpec()).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", "", true)).
				Result(),
			fakeTime: "2020-06-20 06:21:00",
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
			},
		},
		{
			name: "should ignore alreadyExists error on schedule creation",
			backup: dbkpbuilder().
				PeriodicConfig("*/8 * * * *", 1, false).
				Schedule(newTestScheduleSpec()).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", "", true)).
				Result(),
			existingSchedules: []velerov1api.Schedule{
				generateSchedule("2020-06-20 06:20:00", "*/2 * * * *"),
			},
			fakeTime: "2020-06-20 06:21:00",
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
			},
		},
		{
			name: "should not delete old active schedule on schedule creation failure",
			backup: dbkpbuilder().
				PeriodicConfig("*/8 * * * *", 1, false).
				Schedule(newTestScheduleSpec()).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", "", true)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:12:00", v1alpha1.Active, false)).
				Result(),
			fakeTime:            "2020-06-20 06:21:00",
			scheduleCreateError: true,
			shouldErrored:       true,
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", "", true),
				veleroScheduleDetails(t, "2020-06-20 06:12:00", v1alpha1.Active, false),
			},
		},

		{
			name: "should delete existing active schedule on new schedule creation",
			backup: dbkpbuilder().
				PeriodicConfig("*/8 * * * *", 1, false).
				Schedule(newTestScheduleSpec()).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", "", true)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:12:00", v1alpha1.Active, false)).
				Result(),
			fakeTime: "2020-06-20 06:21:00",
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:12:00", v1alpha1.Deleted, false),
			},
		},
		{
			name: "should not update old active schedule status on deletion failure",
			backup: dbkpbuilder().
				PeriodicConfig("*/8 * * * *", 1, false).
				Schedule(newTestScheduleSpec()).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", "", true)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:12:00", v1alpha1.Active, false)).
				Result(),
			fakeTime:            "2020-06-20 06:21:00",
			scheduleDeleteError: true,
			shouldErrored:       true,
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:12:00", v1alpha1.Active, false),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				client             = fake.NewSimpleClientset()
				veleroclient       = velerofake.NewSimpleClientset()
				veleroFakeInformer = veleroinformer.NewSharedInformerFactory(veleroclient, 0)
			)

			testTime, err := time.Parse("2006-01-02 15:04:05", test.fakeTime)
			require.NoError(t, err, "Failed to parse fake time: %v", err)

			// add reactor to assign timestamp for new schedules
			veleroclient.PrependReactor("create", "schedules", func(action core.Action) (bool, runtime.Object, error) {
				obj := action.(core.CreateAction).GetObject().(*velerov1api.Schedule)

				// set creationTimestamp
				setCreationTime := func(obj *velerov1api.Schedule) error {
					var ntime time.Time

					nslice := strings.Split(obj.Name, "-")
					if len(nslice) != 2 {
						return errors.New("invalid name")
					}
					ntime, err = time.Parse("20060102150405", nslice[1])
					if err != nil {
						return errors.New("invalid name format")
					}
					obj.CreationTimestamp.Time = ntime
					return nil
				}

				err = setCreationTime(obj)
				if err != nil {
					return false, obj, err
				}

				return true, obj, nil
			})

			// add existing schedules
			for _, schedule := range test.existingSchedules {
				pinnedSchedule := schedule
				_, err = veleroclient.VeleroV1().Schedules(veleroNamespace).Create(&pinnedSchedule)
				require.NoError(t, err, "failed to setup existing schedule err=%v", err)
			}

			// add reactor for backup create error
			veleroclient.PrependReactor("create", "schedules", func(action core.Action) (bool, runtime.Object, error) {
				if test.scheduleCreateError {
					// inject error
					return true, nil, &apierrors.StatusError{
						ErrStatus: metav1.Status{Reason: metav1.StatusReasonInvalid},
					}
				}
				return false, nil, nil
			})

			// add reactor for backup delete error
			veleroclient.PrependReactor("delete", "schedules", func(action core.Action) (bool, runtime.Object, error) {
				if test.scheduleDeleteError {
					// inject error
					return true, nil, &apierrors.StatusError{
						ErrStatus: metav1.Status{Reason: metav1.StatusReasonInvalid},
					}
				}

				return true, nil, nil
			})

			backupper := NewDMaaSBackupper(
				veleroNamespace,
				client,
				veleroclient.VeleroV1(),
				veleroFakeInformer.Velero().V1(),
				apimachineryclock.NewFakeClock(testTime),
			)

			sort.Sort(sort.Reverse(ScheduleByCreationTimestamp(test.backup.Status.VeleroSchedules)))
			bkpper, ok := backupper.(*dmaasBackup)
			assert.True(t, ok, "failed to parse dmaasbackupper")

			bkpper.logger = logrus.New()

			err = bkpper.processPeriodicConfigSchedule(test.backup)

			// verify error
			if test.shouldErrored {
				require.NotNil(t, err, "execute return nil error")
			} else {
				require.NoError(t, err, "execute return error err=%v", err)
			}

			// verify schedule list
			matchError := compareVeleroSchedule(test.backup.Status.VeleroSchedules, test.expectedVeleroSchedule)
			assert.NoError(t, matchError, "veleroschedule not matched")

			// verify shouldRequeue
			assert.Equal(t, test.shouldRequeue, bkpper.shouldRequeue, "Requeue validation failed")
		})
	}
}

func TestCleanupPeriodicSchedule(t *testing.T) {
	dbkpbuilder := func(retentionCount int, disableCheck bool) *builder.DMaaSBackupBuilder {
		return builder.ForDMaaSBackup(dmaasNamespace, "name").
			PeriodicConfig("*/8 * * * *", retentionCount, disableCheck)
	}

	// generateBackup will return backup object for the given schedule time with status completed
	generateBackup := func(scheduleTime, prefix string) *velerobuilder.BackupBuilder {
		testTime, err := time.Parse("2006-01-02 15:04:05", scheduleTime)
		require.NoError(t, err, "Failed to parse schedule time: %v", err)

		return velerobuilder.ForBackup(veleroNamespace, "name-"+testTime.Format("20060102150405")+"-"+prefix).
			FromSchedule(&velerov1api.Schedule{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: veleroNamespace,
					Name:      "name-" + testTime.Format("20060102150405"),
				},
				Spec: velerov1api.ScheduleSpec{},
			}).
			ObjectMeta(
				velerobuilder.WithLabels(
					// add label using key, value
					v1alpha1.DMaaSBackupLabelKey, "name",
				),
			).Phase(velerov1api.BackupPhaseCompleted)
	}

	tests := []struct {
		name                   string
		dmaasbackup            *v1alpha1.DMaaSBackup
		existingBackups        []*velerov1api.Backup
		backupDeleteError      bool
		backupListError        bool
		expectedVeleroSchedule []v1alpha1.VeleroScheduleDetails // compare only status and name, excluding reserved one
		expectedBackups        []*velerov1api.Backup
	}{
		{
			name: "should not delete backup if required schedule count is not reached",
			dmaasbackup: dbkpbuilder(3, false).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				Result(),
			existingBackups: []*velerov1api.Backup{
				// we are having two veleroschedules
				// add 1 backups for active schedule
				generateBackup("2020-06-20 06:20:00", "b1").Result(),

				// add 1 backups for deleted schedule
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
			},
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
			},
			expectedBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
			},
		},
		{
			name: "should delete backup if retentionCount limit is reached",
			dmaasbackup: dbkpbuilder(1, false).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false)).
				Result(),
			existingBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
				generateBackup("2020-06-20 06:00:00", "b1").Result(),
			},
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
				veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false),
			},
			expectedBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
			},
		},
		{
			name: "should not delete last successful schedule's backup if retained one doesn't have successful backup",
			dmaasbackup: dbkpbuilder(1, false).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false)).
				Result(),
			existingBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Phase(velerov1api.BackupPhaseFailed).Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Phase(velerov1api.BackupPhaseInProgress).Result(),
				generateBackup("2020-06-20 06:00:00", "b1").Result(),
			},
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
				veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false),
			},
			expectedBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Phase(velerov1api.BackupPhaseFailed).Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Phase(velerov1api.BackupPhaseInProgress).Result(),
				generateBackup("2020-06-20 06:00:00", "b1").Result(),
			},
		},
		{
			name: "should delete last successful schedule's backup if retained one doesn't have successful backup, if DisableSuccessfulBackupRetain set",
			dmaasbackup: dbkpbuilder(1, true).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false)).
				Result(),
			existingBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Phase(velerov1api.BackupPhaseFailed).Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Phase(velerov1api.BackupPhaseInProgress).Result(),
				generateBackup("2020-06-20 06:00:00", "b1").Result(),
			},
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
				veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false),
			},
			expectedBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Phase(velerov1api.BackupPhaseFailed).Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Phase(velerov1api.BackupPhaseInProgress).Result(),
			},
		},
		{
			name: "should update last schedule status as Erased if all backups are deleted",
			dmaasbackup: dbkpbuilder(1, false).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false)).
				Result(),
			existingBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
			},
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
				veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Erased, false),
			},
			expectedBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
			},
		},
		{
			name: "should not delete backups for schedule if schedule is not deleted",
			dmaasbackup: dbkpbuilder(1, false).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:00:00", "", false)).
				Result(),
			existingBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
				generateBackup("2020-06-20 06:00:00", "b1").Result(),
			},
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
				veleroScheduleDetails(t, "2020-06-20 06:00:00", "", false),
			},
			expectedBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
				generateBackup("2020-06-20 06:00:00", "b1").Result(),
			},
		},
		{
			name: "should not account erased scheduled's backup for SuccessfulBackupRetain",
			dmaasbackup: dbkpbuilder(1, false).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Erased, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 05:00:00", v1alpha1.Deleted, false)).
				Result(),
			existingBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Phase(velerov1api.BackupPhaseFailed).Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Phase(velerov1api.BackupPhaseFailed).Result(),
				generateBackup("2020-06-20 06:00:00", "b1").Phase(velerov1api.BackupPhaseCompleted).Result(),
			},
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
				veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Erased, false),
				veleroScheduleDetails(t, "2020-06-20 05:00:00", v1alpha1.Deleted, false),
			},
			expectedBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Phase(velerov1api.BackupPhaseFailed).Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Phase(velerov1api.BackupPhaseFailed).Result(),
				generateBackup("2020-06-20 06:00:00", "b1").Phase(velerov1api.BackupPhaseCompleted).Result(),
			},
		},
		{
			name: "should not delete backup if creation of deletionrequest failed",
			dmaasbackup: dbkpbuilder(1, false).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false)).
				Result(),
			existingBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
				generateBackup("2020-06-20 06:00:00", "b1").Result(),
			},
			backupDeleteError: true,
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
				veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false),
			},
			expectedBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
				generateBackup("2020-06-20 06:00:00", "b1").Result(),
			},
		},
		{
			name: "should not update schedule status if backup list failed",
			dmaasbackup: dbkpbuilder(1, false).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false)).
				Result(),
			existingBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
			},
			backupListError: true,
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
				veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false),
			},
			expectedBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
			},
		},
		{
			name: "should not update schedule status if backup list failed, if DisableSuccessfulBackupRetain set",
			dmaasbackup: dbkpbuilder(1, true).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false)).
				Result(),
			existingBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
			},
			backupListError: true,
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
				veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false),
			},
			expectedBackups: []*velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1").Result(),
				generateBackup("2020-06-20 06:10:00", "b1").Result(),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				client             = fake.NewSimpleClientset()
				veleroclient       = velerofake.NewSimpleClientset()
				veleroFakeInformer = veleroinformer.NewSharedInformerFactory(veleroclient, 0)
			)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var wg sync.WaitGroup

			backupDeleter := func(backupName string) {
				err := veleroclient.VeleroV1().Backups(veleroNamespace).Delete(backupName, &metav1.DeleteOptions{})
				require.NoError(t, err, "backup=%s deletion failed, err=%v", backupName, err)
			}

			veleroclient.PrependReactor("create", "deletebackuprequests", func(action core.Action) (bool, runtime.Object, error) {
				obj := action.(core.CreateAction).GetObject().(*velerov1api.DeleteBackupRequest)

				if test.backupDeleteError {
					return true, nil, &apierrors.StatusError{
						ErrStatus: metav1.Status{Reason: metav1.StatusReasonInvalid},
					}
				}

				wg.Add(1)
				go func() {
					backupDeleter(obj.Spec.BackupName)
					wg.Done()
				}()
				return true, obj, nil
			})

			// Create required backups
			if len(test.existingBackups) != 0 {
				for _, backup := range test.existingBackups {
					_, err := veleroclient.VeleroV1().Backups(veleroNamespace).Create(backup)
					require.NoError(t, err, "creating existing backup failed, err=%v", err)
				}
			}

			go veleroFakeInformer.Start(ctx.Done())
			cache.WaitForCacheSync(ctx.Done(), veleroFakeInformer.Velero().V1().Backups().Informer().HasSynced)

			backupper := NewDMaaSBackupper(
				veleroNamespace,
				client,
				veleroclient.VeleroV1(),
				veleroFakeInformer.Velero().V1(),
				apimachineryclock.NewFakeClock(time.Time{}),
			)

			sort.Sort(sort.Reverse(ScheduleByCreationTimestamp(test.dmaasbackup.Status.VeleroSchedules)))
			bkpper, ok := backupper.(*dmaasBackup)
			assert.True(t, ok, "failed to parse dmaasbackupper")
			bkpper.logger = logrus.New()

			if test.backupListError {
				// assign mock backuplister to inject error on List API
				bkpper.backupLister = &fakeBackupLister{}
			}

			err := bkpper.cleanupPeriodicSchedule(test.dmaasbackup)
			require.NoError(t, err, "cleanup returned error err=%v", err)

			// verify schedule list
			matchError := compareVeleroSchedule(test.dmaasbackup.Status.VeleroSchedules, test.expectedVeleroSchedule)
			assert.NoError(t, matchError, "veleroschedule not matched")

			// verify shouldRequeue should be false
			assert.Equal(t, false, bkpper.shouldRequeue, "Requeue validation failed")

			wg.Wait()
			require.NoError(t,
				verifyBackups(
					veleroclient.VeleroV1().Backups(veleroNamespace),
					test.expectedBackups,
				),
				"backup verification failed")
		})
	}
}

func verifyBackups(backupInterface velerov1.BackupInterface, expectedBackups []*velerov1api.Backup) error {
	existingBackups, err := backupInterface.List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to list backups")
	}

	if len(existingBackups.Items) != len(expectedBackups) {
		return errors.Errorf("mismatch in length existingbackup:%d expectedBackups:%d", len(existingBackups.Items), len(expectedBackups))
	}

	// sort existing backup by their name
	sort.SliceStable(existingBackups.Items, func(i, j int) bool {
		return (strings.Compare(existingBackups.Items[i].Name, existingBackups.Items[j].Name) >= 0)
	})

	// sort expectedBackups by their name so that we can compare it with existing one through index
	sort.SliceStable(expectedBackups, func(i, j int) bool {
		return (strings.Compare(expectedBackups[i].Name, expectedBackups[j].Name) >= 0)
	})

	for i, bkp := range existingBackups.Items {
		if bkp.Name != expectedBackups[i].Name {
			return errors.Errorf("Backup Mismatch existing:%s expected:%s\n", bkp.Name, expectedBackups[i].Name)
		}
	}
	return nil
}

// fakeBackupLister is mocking of velero backuplister interface to return error
type fakeBackupLister struct{}

func (*fakeBackupLister) List(selector labels.Selector) ([]*velerov1api.Backup, error) {
	return []*velerov1api.Backup{}, errors.New("internal Error")
}

func (*fakeBackupLister) Get(name string) (*velerov1api.Backup, error) {
	return nil, errors.New("internal Error")
}
