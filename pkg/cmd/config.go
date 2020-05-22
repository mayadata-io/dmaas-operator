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

package cmd

import (
	"os"

	"github.com/spf13/pflag"

	clientset "github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Config interface provides api to access clientset and namespace
type Config interface {
	// BindFlags binds common flags (--kubeconfig, --namespace) to the passed-in FlagSet.
	BindFlags(flags *pflag.FlagSet)
	// Client returns dmaas-operator client.
	Client() (clientset.Interface, error)
	// KubeClient returns Kubernetes client.
	KubeClient() (kubernetes.Interface, error)
	// SetClientQPS sets the Queries Per Second for a client.
	SetClientQPS(float32)
	// SetClientBurst sets the Burst for a client.
	SetClientBurst(int)
	// GetNamespace return dmaas namespace
	GetNamespace() string
}

type config struct {
	kubeconfig  string
	namespace   string
	clientQPS   float32
	clientBurst int
	flags       *pflag.FlagSet
}

// NewConfig return config for clientset
func NewConfig() Config {
	c := &config{
		flags: pflag.NewFlagSet("", pflag.ContinueOnError),
	}

	c.namespace = os.Getenv(envDMaaSNamespaceKey)

	c.flags.StringVar(&c.kubeconfig,
		"kubeconfig",
		"",
		"Path to the kubeconfig file to use to talk to the Kubernetes apiserver. If unset, try the environment variable KUBECONFIG, as well as in-cluster configuration",
	)
	c.flags.StringVarP(&c.namespace,
		"namespace",
		"n",
		c.namespace,
		"The namespace in which dmaas-operator should operate",
	)

	return c
}

// BindFlags binds common flags to the given flagset
func (c *config) BindFlags(flags *pflag.FlagSet) {
	flags.AddFlagSet(c.flags)
}

// ClientConfig return cluster config
func (c *config) ClientConfig() (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = c.kubeconfig
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)

	clientConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "error finding Kubernetes API server config in --kubeconfig, $KUBECONFIG, or in-cluster configuration")
	}

	if c.clientQPS > 0.0 {
		clientConfig.QPS = c.clientQPS
	}

	if c.clientBurst > 0 {
		clientConfig.Burst = c.clientBurst
	}

	return clientConfig, nil
}

// Client returns clientset to access dmaas-operator resources
func (c *config) Client() (clientset.Interface, error) {
	clientConfig, err := c.ClientConfig()
	if err != nil {
		return nil, err
	}

	client, err := clientset.NewForConfig(clientConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return client, nil
}

// KubeClient return clientset to access k8s native resources
func (c *config) KubeClient() (kubernetes.Interface, error) {
	clientConfig, err := c.ClientConfig()
	if err != nil {
		return nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return kubeClient, nil
}

// SetClientQPS set given qps value to config
func (c *config) SetClientQPS(qps float32) {
	c.clientQPS = qps
}

// SetClientBurst set given burst value to config
func (c *config) SetClientBurst(burst int) {
	c.clientBurst = burst
}

// GetNamespace return the namespace of dmaas-operator
func (c *config) GetNamespace() string {
	return c.namespace
}
