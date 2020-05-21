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
	"strings"

	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	// defaultOpenebsNS default openebs namespace
	defaultOpenebsNS string = "openebs"

	// defaultVeleroNs default velero namespace
	defaultVeleroNs string = "velero"

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
	return &serverOpts{
		openebsNamespace: defaultOpenebsNS,
		veleroNamespace:  defaultVeleroNs,
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

// NewCmdServer returns the command for server
func NewCmdServer(c Config) *cobra.Command {
	opts := newServerOpts()

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run dmaas-operator",

		Run: func(cmd *cobra.Command, args []string) {
			logger := newLogger(opts)


			logger.Infof("DMaaS operator started")
		},
	}

	opts.BindFlags(cmd.PersistentFlags())
	c.BindFlags(cmd.PersistentFlags())

	return cmd
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
	}

	logger.Formatter = new(logrus.JSONFormatter)

	return logger
}
