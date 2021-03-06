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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/mayadata-io/dmaas-operator/pkg/apis/mayadata.io/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// PreBackupActionLister helps list PreBackupActions.
type PreBackupActionLister interface {
	// List lists all PreBackupActions in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.PreBackupAction, err error)
	// PreBackupActions returns an object that can list and get PreBackupActions.
	PreBackupActions(namespace string) PreBackupActionNamespaceLister
	PreBackupActionListerExpansion
}

// preBackupActionLister implements the PreBackupActionLister interface.
type preBackupActionLister struct {
	indexer cache.Indexer
}

// NewPreBackupActionLister returns a new PreBackupActionLister.
func NewPreBackupActionLister(indexer cache.Indexer) PreBackupActionLister {
	return &preBackupActionLister{indexer: indexer}
}

// List lists all PreBackupActions in the indexer.
func (s *preBackupActionLister) List(selector labels.Selector) (ret []*v1alpha1.PreBackupAction, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.PreBackupAction))
	})
	return ret, err
}

// PreBackupActions returns an object that can list and get PreBackupActions.
func (s *preBackupActionLister) PreBackupActions(namespace string) PreBackupActionNamespaceLister {
	return preBackupActionNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// PreBackupActionNamespaceLister helps list and get PreBackupActions.
type PreBackupActionNamespaceLister interface {
	// List lists all PreBackupActions in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.PreBackupAction, err error)
	// Get retrieves the PreBackupAction from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.PreBackupAction, error)
	PreBackupActionNamespaceListerExpansion
}

// preBackupActionNamespaceLister implements the PreBackupActionNamespaceLister
// interface.
type preBackupActionNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all PreBackupActions in the indexer for a given namespace.
func (s preBackupActionNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.PreBackupAction, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.PreBackupAction))
	})
	return ret, err
}

// Get retrieves the PreBackupAction from the indexer for a given namespace and name.
func (s preBackupActionNamespaceLister) Get(name string) (*v1alpha1.PreBackupAction, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("prebackupaction"), name)
	}
	return obj.(*v1alpha1.PreBackupAction), nil
}
