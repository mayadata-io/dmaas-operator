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

package builder

import (
	"github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DMaaSBackupBuilder build dmaasbackup object
type DMaaSBackupBuilder struct {
	object *v1alpha1.DMaaSBackup
}

// ForDMaaSBackup is returns dmaasbackup builder
func ForDMaaSBackup(ns, name string) *DMaaSBackupBuilder {
	return &DMaaSBackupBuilder{
		object: &v1alpha1.DMaaSBackup{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       "DMaaSBackup",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
		},
	}
}

// Result return the dmaasbackup resource
func (d *DMaaSBackupBuilder) Result() *v1alpha1.DMaaSBackup {
	return d.object
}

// Phase set the phase of dmaasbackup
func (d *DMaaSBackupBuilder) Phase(phase v1alpha1.DMaaSBackupPhase) *DMaaSBackupBuilder {
	d.object.Status.Phase = phase
	return d
}

// State set the state of dmaasbackup
func (d *DMaaSBackupBuilder) State(state v1alpha1.DMaasBackupState) *DMaaSBackupBuilder {
	d.object.Spec.State = state
	return d
}

// DeletionTimeStamp set the deletion timestamp of the dmaasbackup
func (d *DMaaSBackupBuilder) DeletionTimeStamp(deletionTime *metav1.Time) *DMaaSBackupBuilder {
	d.object.DeletionTimestamp = deletionTime
	return d
}

// Finalizer add the given finalizer to dmaasbackup
func (d *DMaaSBackupBuilder) Finalizer(finalizers ...string) *DMaaSBackupBuilder {
	d.object.SetFinalizers(finalizers)
	return d
}
