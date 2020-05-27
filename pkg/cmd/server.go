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
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/mayadata-io/dmaas-operator/pkg/controller"
	clientset "github.com/mayadata-io/dmaas-operator/pkg/generated/clientset/versioned"
	informers "github.com/mayadata-io/dmaas-operator/pkg/generated/informers/externalversions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	velero "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/kubernetes"
)

const (
	// envDMaaSNamespaceKey environment variable for dmaas-operator namespace
	envDMaaSNamespaceKey string = "NAMESPACE"

	// envOpenebsNamespaceKey environment variable for openebs namespace
	envOpenebsNamespaceKey string = "OPENEBS_NAMESPACE"

	// envVeleroNamespaceKey environment variable for velero namespace
	envVeleroNamespaceKey string = "VELERO_NAMESPACE"

	// defaultClientQPS for dmaas-operator
	defaultClientQPS float32 = 20.0

	// defaultClientBurst for dmaas-operator
	defaultClientBurst int = 30
)

type serverOpts struct {
	openebsNamespace string
	veleroNamespace  string
	clientQPS        float32
	clientBurst      int
	logLevel         string
}

func newServerOpts() *serverOpts {
	openebsNs := os.Getenv(envOpenebsNamespaceKey)
	veleroNs := os.Getenv(envVeleroNamespaceKey)

	return &serverOpts{
		openebsNamespace: openebsNs,
		veleroNamespace:  veleroNs,
		clientBurst:      defaultClientBurst,
		clientQPS:        defaultClientQPS,
		logLevel:         logrus.InfoLevel.String(),
	}
}

func (s *serverOpts) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&s.openebsNamespace, "openebs-namespace",
		s.openebsNamespace,
		"namespace in which openebs is installed")
	flags.StringVar(&s.veleroNamespace, "velero-namespace",
		s.veleroNamespace,
		"namespace in which velero is installed")
	flags.IntVar(&s.clientBurst, "kube-api-burst",
		s.clientBurst,
		"maximum number of requests by the server to the Kubernetes API in a short period of time")
	flags.Float32Var(&s.clientQPS, "kube-api-qps",
		s.clientQPS,
		"maximum number of requests per second by the server to the Kubernetes API once the burst limit has been reached")
	flags.StringVar(&s.logLevel, "log-level",
		s.logLevel,
		fmt.Sprintf("logging level. valid values are %s.", strings.Join(allowedLoggingLevel(), ", ")))
}

type server struct {
	namespace             string
	openebsNamespace      string
	veleroNamespace       string
	kubeClient            kubernetes.Interface
	dmaasClient           clientset.Interface
	veleroClient          velero.Interface
	sharedInformerFactory informers.SharedInformerFactory
	ctx                   context.Context
	cancelFunc            context.CancelFunc
	logger                logrus.FieldLogger
	clock                 clock.Clock
	opts                  *serverOpts
}

// NewCmdServer returns the command for server
func NewCmdServer(c Config) *cobra.Command {
	opts := newServerOpts()

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run dmaas-operator",

		Run: func(cmd *cobra.Command, args []string) {
			logger := newLogger(opts)

			logger.Infof("DMaaS operator started")

			CheckError(validateConfig(c, opts, logger))

			s, err := newServer(c, opts, logger)
			CheckError(err)

			controllerList := s.getSupportedControllers()
			CheckError(s.startController(controllerList))
		},
	}

	opts.BindFlags(cmd.PersistentFlags())
	c.BindFlags(cmd.PersistentFlags())

	return cmd
}

// validateConfig validate the config and server options
func validateConfig(cfg Config, opts *serverOpts, logger *logrus.Logger) error {
	if cfg.GetNamespace() == "" {
		return errors.New("dmaas-operator namespace can not be empty")
	}

	if opts.openebsNamespace == "" {
		logger.Infof("Openebs namespace is empty, using namespace=%s for Openebs", cfg.GetNamespace())
		opts.openebsNamespace = cfg.GetNamespace()
	}

	if opts.veleroNamespace == "" {
		logger.Infof("Velero namespace is empty, using namespace=%s for Velero", cfg.GetNamespace())
		opts.veleroNamespace = cfg.GetNamespace()
	}

	return nil
}

func allowedLoggingLevel() []string {
	var logLevel []string

	for _, l := range logrus.AllLevels {
		logLevel = append(logLevel, l.String())
	}
	return logLevel
}

// newLogger return logger to log on stdout with json format
func newLogger(opts *serverOpts) *logrus.Logger {
	logger := logrus.New()
	logger.Out = os.Stdout

	if lvl, err := logrus.ParseLevel(opts.logLevel); err != nil {
		logrus.Errorf("Invalid log-level value, using level %s", logrus.InfoLevel.String())
		logger.Level = logrus.InfoLevel
	} else {
		logger.Level = lvl
		logger.Infof("Setting log-level=%s", lvl.String())
	}

	logger.Formatter = new(logrus.JSONFormatter)

	return logger
}

// newServer return server
func newServer(cfg Config, opts *serverOpts, logger *logrus.Logger) (*server, error) {
	// update qps and burst to config before accessing clientset
	if err := cfg.SetClientQPS(opts.clientQPS); err != nil {
		return nil, err
	}

	if err := cfg.SetClientBurst(opts.clientBurst); err != nil {
		return nil, err
	}

	kubeClient, err := cfg.KubeClient()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch kubernetes client")
	}

	dmaasClient, err := cfg.Client()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch dmaas client")
	}

	veleroClient, err := cfg.VeleroClient()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch velero client")
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	return &server{
		namespace:             cfg.GetNamespace(),
		openebsNamespace:      opts.openebsNamespace,
		veleroNamespace:       opts.veleroNamespace,
		kubeClient:            kubeClient,
		dmaasClient:           dmaasClient,
		veleroClient:          veleroClient,
		sharedInformerFactory: informers.NewSharedInformerFactoryWithOptions(dmaasClient, 0, informers.WithNamespace(cfg.GetNamespace())),
		ctx:                   ctx,
		logger:                logger,
		cancelFunc:            cancelFunc,
		clock:                 clock.RealClock{},
		opts:                  opts,
	}, nil
}

var (
	// defaultControllerWorker is default value of controller worker
	defaultControllerWorker int = 1
)

// getSupportedControllers returns list of supported controller
func (s *server) getSupportedControllers() []controller.Controller {
	var controllerList []controller.Controller

	dmaasBackupCtrl := controller.NewDMaaSBackupController(
		s.namespace, s.openebsNamespace, s.veleroNamespace,
		s.kubeClient,
		s.dmaasClient,
		s.sharedInformerFactory.Mayadata().V1alpha1().DMaaSBackups(),
		s.veleroClient,
		s.logger,
		s.clock,
		defaultControllerWorker,
	)
	controllerList = append(controllerList, dmaasBackupCtrl)
	// add other controllers in similar way

	return controllerList
}

// startController run all the given controllers
func (s *server) startController(controllers []controller.Controller) error {
	// start the informers
	s.sharedInformerFactory.Start(s.ctx.Done())

	s.logger.Infof("Waiting for caches to sync")
	for informer, ok := range s.sharedInformerFactory.WaitForCacheSync(s.ctx.Done()) {
		if !ok {
			return errors.Errorf("timed out waiting for caches to sync for informer=%v", informer)
		}
		s.logger.WithField("informer", informer).Infof("cache synced")
	}

	s.logger.Infof("All informers cache synced")

	// start all controllers

	s.logger.Info("starting all controllers")

	var wg sync.WaitGroup
	for k := range controllers {
		c := controllers[k]
		wg.Add(1)

		go func() {
			_ = c.Run(s.ctx)
			wg.Done()
		}()
	}

	s.logger.Infof("All controllers started")

	<-s.ctx.Done()

	s.logger.Infof("Waiting for all controllers to stop gracefully")

	wg.Wait()
	return nil
}
