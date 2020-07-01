package dmaasbackup

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/client-go/testing"

	"k8s.io/client-go/tools/cache"

	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
	"github.com/mayadata-io/dmaas-operator/pkg/builder"
	"github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned/fake"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	velerobuilder "github.com/vmware-tanzu/velero/pkg/builder"
	velerofake "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/fake"
	veleroinformer "github.com/vmware-tanzu/velero/pkg/generated/informers/externalversions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryclock "k8s.io/apimachinery/pkg/util/clock"
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
		return builder.ForDMaaSBackup("ns", "name")
	}

	generateSchedule := func(creationTime, cronSchedule string) velerov1api.Schedule {
		testTime, err := time.Parse("2006-01-02 15:04:05", creationTime)
		require.NoError(t, err, "Failed to parse schedule time: %v", err)

		schedule := velerobuilder.ForSchedule("ns", "name-"+testTime.Format("20060102150405")).
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
				PeriodicConfig("*/8", 1).
				Schedule(newTestScheduleSpec()).
				Result(),
			fakeTime:      "2020-06-20 06:20:00",
			shouldErrored: true,
		},
		{
			name: "should reserve name if veleroschedule is empty",
			backup: dbkpbuilder().
				PeriodicConfig("*/8 * * * *", 1).
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
				PeriodicConfig("*/8 * * * *", 1).
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
				PeriodicConfig("*/8 * * * *", 1).
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
				PeriodicConfig("*/8 * * * *", 1).
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
				PeriodicConfig("*/8 * * * *", 1).
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
				PeriodicConfig("*/8 * * * *", 1).
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
				PeriodicConfig("*/8 * * * *", 1).
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

			scheduleInventory := map[string]bool{}

			testTime, err := time.Parse("2006-01-02 15:04:05", test.fakeTime)
			require.NoError(t, err, "Failed to parse fake time: %v", err)

			veleroclient.PrependReactor("create", "schedules", func(action core.Action) (bool, runtime.Object, error) {
				obj := action.(core.CreateAction).GetObject().(*velerov1api.Schedule)

				if test.scheduleCreateError {
					// inject error
					return true, nil, &apierrors.StatusError{
						ErrStatus: metav1.Status{Reason: metav1.StatusReasonInvalid},
					}
				}
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

				_, exists := scheduleInventory[obj.Name]
				if exists {
					return true, obj, &apierrors.StatusError{
						ErrStatus: metav1.Status{Reason: metav1.StatusReasonAlreadyExists},
					}
				}

				// add schedule to scheduleInventory
				scheduleInventory[obj.Name] = true

				return true, obj, nil
			})

			if len(test.existingSchedules) != 0 {
				for _, schedule := range test.existingSchedules {
					pinnedSchedule := schedule
					_, err = veleroclient.VeleroV1().Schedules("ns").Create(&pinnedSchedule)
					require.NoError(t, err, "failed to setup existing schedule err=%v", err)
				}
			}

			veleroclient.PrependReactor("delete", "schedules", func(action core.Action) (bool, runtime.Object, error) {
				scheduleName := action.(core.DeleteAction).GetName()

				if test.scheduleDeleteError {
					// inject error
					return true, nil, &apierrors.StatusError{
						ErrStatus: metav1.Status{Reason: metav1.StatusReasonInvalid},
					}
				}
				_, exists := scheduleInventory[scheduleName]
				if exists {
					delete(scheduleInventory, scheduleName)
				} else {
					return true, nil, &apierrors.StatusError{
						ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound},
					}
				}

				return true, nil, nil
			})

			backupper := NewDMaaSBackupper(
				"ns",
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
	dbkpbuilder := func(retentionCount int) *builder.DMaaSBackupBuilder {
		return builder.ForDMaaSBackup("ns", "name").
			PeriodicConfig("*/8 * * * *", retentionCount)
	}

	// generateBackup will return backup object for the given schedule time
	generateBackup := func(scheduleTime, prefix string) velerov1api.Backup {
		testTime, err := time.Parse("2006-01-02 15:04:05", scheduleTime)
		require.NoError(t, err, "Failed to parse schedule time: %v", err)

		return *velerobuilder.ForBackup("ns", "name-"+testTime.Format("20060102150405")+"-"+prefix).
			FromSchedule(&velerov1api.Schedule{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Name:      "name-" + testTime.Format("20060102150405"),
				},
				Spec: velerov1api.ScheduleSpec{},
			}).
			ObjectMeta(
				velerobuilder.WithLabels(
					// add label using key, value
					v1alpha1.DMaaSBackupLabelKey, "name",
				),
			).
			Result()
	}

	tests := []struct {
		name                   string
		dmaasbackup            *v1alpha1.DMaaSBackup
		existingBackups        []velerov1api.Backup
		backupDeleteError      bool
		expectedVeleroSchedule []v1alpha1.VeleroScheduleDetails // compare only status and name, excluding reserved one
		expectedBackups        []velerov1api.Backup
	}{
		{
			name: "should not delete backup if required schedule count is not reached",
			dmaasbackup: dbkpbuilder(3).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				Result(),
			existingBackups: []velerov1api.Backup{
				// we are having two veleroschedules
				// add 1 backups for active schedule
				generateBackup("2020-06-20 06:20:00", "b1"),

				// add 1 backups for deleted schedule
				generateBackup("2020-06-20 06:10:00", "b1"),
			},
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
			},
			expectedBackups: []velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1"),
				generateBackup("2020-06-20 06:10:00", "b1"),
			},
		},
		{
			name: "should delete backup if retentionCount limit is reached",
			dmaasbackup: dbkpbuilder(1).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false)).
				Result(),
			existingBackups: []velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1"),
				generateBackup("2020-06-20 06:10:00", "b1"),
				generateBackup("2020-06-20 06:00:00", "b1"),
			},
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
				veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false),
			},
			expectedBackups: []velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1"),
				generateBackup("2020-06-20 06:10:00", "b1"),
				//	generateBackup("2020-06-20 06:00:00", "b1"),
			},
		},
		{
			name: "should update last schedule status as Erased if all backups are deleted",
			dmaasbackup: dbkpbuilder(1).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false)).
				Result(),
			existingBackups: []velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1"),
				generateBackup("2020-06-20 06:10:00", "b1"),
			},
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
				veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Erased, false),
			},
			expectedBackups: []velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1"),
				generateBackup("2020-06-20 06:10:00", "b1"),
			},
		},
		{
			name: "should not delete backups for schedule if schedule is not deleted",
			dmaasbackup: dbkpbuilder(1).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:00:00", "", false)).
				Result(),
			existingBackups: []velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1"),
				generateBackup("2020-06-20 06:10:00", "b1"),
				generateBackup("2020-06-20 06:00:00", "b1"),
			},
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
				veleroScheduleDetails(t, "2020-06-20 06:00:00", "", false),
			},
			expectedBackups: []velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1"),
				generateBackup("2020-06-20 06:10:00", "b1"),
				generateBackup("2020-06-20 06:00:00", "b1"),
			},
		},
		{
			name: "should not delete backup if creation of deletionrequest failed",
			dmaasbackup: dbkpbuilder(1).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false)).
				WithVeleroSchedules(veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false)).
				Result(),
			existingBackups: []velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1"),
				generateBackup("2020-06-20 06:10:00", "b1"),
				generateBackup("2020-06-20 06:00:00", "b1"),
			},
			backupDeleteError: true,
			expectedVeleroSchedule: []v1alpha1.VeleroScheduleDetails{
				veleroScheduleDetails(t, "2020-06-20 06:20:00", v1alpha1.Active, false),
				veleroScheduleDetails(t, "2020-06-20 06:10:00", v1alpha1.Deleted, false),
				veleroScheduleDetails(t, "2020-06-20 06:00:00", v1alpha1.Deleted, false),
			},
			expectedBackups: []velerov1api.Backup{
				generateBackup("2020-06-20 06:20:00", "b1"),
				generateBackup("2020-06-20 06:10:00", "b1"),
				generateBackup("2020-06-20 06:00:00", "b1"),
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
				err := veleroclient.VeleroV1().Backups("ns").Delete(backupName, &metav1.DeleteOptions{})
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
					pinnedBackup := backup
					_, err := veleroclient.VeleroV1().Backups("ns").Create(&pinnedBackup)
					require.NoError(t, err, "creating existing backup failed, err=%v", err)
				}
			}

			go veleroFakeInformer.Start(ctx.Done())
			cache.WaitForCacheSync(ctx.Done(), veleroFakeInformer.Velero().V1().Backups().Informer().HasSynced)

			backupper := NewDMaaSBackupper(
				"ns",
				client,
				veleroclient.VeleroV1(),
				veleroFakeInformer.Velero().V1(),
				apimachineryclock.NewFakeClock(time.Time{}),
			)

			sort.Sort(sort.Reverse(ScheduleByCreationTimestamp(test.dmaasbackup.Status.VeleroSchedules)))
			bkpper, ok := backupper.(*dmaasBackup)
			assert.True(t, ok, "failed to parse dmaasbackupper")

			bkpper.logger = logrus.New()

			err := bkpper.cleanupPeriodicSchedule(test.dmaasbackup)
			require.NoError(t, err, "cleanup returned error err=%v", err)

			// verify schedule list
			matchError := compareVeleroSchedule(test.dmaasbackup.Status.VeleroSchedules, test.expectedVeleroSchedule)
			assert.NoError(t, matchError, "veleroschedule not matched")

			// verify shouldRequeue should be false
			assert.Equal(t, false, bkpper.shouldRequeue, "Requeue validation failed")

			wg.Wait()
			require.NoError(t, verifyBackups(veleroclient, test.expectedBackups), "backup verification failed")
		})
	}
}

func verifyBackups(clientset *velerofake.Clientset, expectedBackups []velerov1api.Backup) error {
	existingBackups, err := clientset.VeleroV1().Backups("ns").List(metav1.ListOptions{})
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
