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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"time"

	v1alpha1 "github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
	scheme "github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// PreBackupActionsGetter has a method to return a PreBackupActionInterface.
// A group's client should implement this interface.
type PreBackupActionsGetter interface {
	PreBackupActions(namespace string) PreBackupActionInterface
}

// PreBackupActionInterface has methods to work with PreBackupAction resources.
type PreBackupActionInterface interface {
	Create(*v1alpha1.PreBackupAction) (*v1alpha1.PreBackupAction, error)
	Update(*v1alpha1.PreBackupAction) (*v1alpha1.PreBackupAction, error)
	UpdateStatus(*v1alpha1.PreBackupAction) (*v1alpha1.PreBackupAction, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.PreBackupAction, error)
	List(opts v1.ListOptions) (*v1alpha1.PreBackupActionList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.PreBackupAction, err error)
	PreBackupActionExpansion
}

// preBackupActions implements PreBackupActionInterface
type preBackupActions struct {
	client rest.Interface
	ns     string
}

// newPreBackupActions returns a PreBackupActions
func newPreBackupActions(c *MayadataV1alpha1Client, namespace string) *preBackupActions {
	return &preBackupActions{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the preBackupAction, and returns the corresponding preBackupAction object, and an error if there is any.
func (c *preBackupActions) Get(name string, options v1.GetOptions) (result *v1alpha1.PreBackupAction, err error) {
	result = &v1alpha1.PreBackupAction{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("prebackupactions").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of PreBackupActions that match those selectors.
func (c *preBackupActions) List(opts v1.ListOptions) (result *v1alpha1.PreBackupActionList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.PreBackupActionList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("prebackupactions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested preBackupActions.
func (c *preBackupActions) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("prebackupactions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a preBackupAction and creates it.  Returns the server's representation of the preBackupAction, and an error, if there is any.
func (c *preBackupActions) Create(preBackupAction *v1alpha1.PreBackupAction) (result *v1alpha1.PreBackupAction, err error) {
	result = &v1alpha1.PreBackupAction{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("prebackupactions").
		Body(preBackupAction).
		Do().
		Into(result)
	return
}

// Update takes the representation of a preBackupAction and updates it. Returns the server's representation of the preBackupAction, and an error, if there is any.
func (c *preBackupActions) Update(preBackupAction *v1alpha1.PreBackupAction) (result *v1alpha1.PreBackupAction, err error) {
	result = &v1alpha1.PreBackupAction{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("prebackupactions").
		Name(preBackupAction.Name).
		Body(preBackupAction).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *preBackupActions) UpdateStatus(preBackupAction *v1alpha1.PreBackupAction) (result *v1alpha1.PreBackupAction, err error) {
	result = &v1alpha1.PreBackupAction{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("prebackupactions").
		Name(preBackupAction.Name).
		SubResource("status").
		Body(preBackupAction).
		Do().
		Into(result)
	return
}

// Delete takes name of the preBackupAction and deletes it. Returns an error if one occurs.
func (c *preBackupActions) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("prebackupactions").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *preBackupActions) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("prebackupactions").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched preBackupAction.
func (c *preBackupActions) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.PreBackupAction, err error) {
	result = &v1alpha1.PreBackupAction{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("prebackupactions").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
