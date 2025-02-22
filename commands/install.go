// percona-everest-cli
// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package commands ...
package commands

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/percona/percona-everest-cli/pkg/install"
	"github.com/percona/percona-everest-cli/pkg/output"
)

func newInstallCmd(l *zap.SugaredLogger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "install",
		Run: func(cmd *cobra.Command, args []string) {
			initInstallViperFlags(cmd)
			c := &install.Config{}
			err := viper.Unmarshal(c)
			if err != nil {
				os.Exit(1)
			}

			op, err := install.NewInstall(*c, l)
			if err != nil {
				l.Error(err)
				os.Exit(1)
			}

			if err := op.Run(cmd.Context()); err != nil {
				output.PrintError(err, l)
				os.Exit(1)
			}
		},
	}
	initInstallFlags(cmd)

	return cmd
}

func initInstallFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("kubeconfig", "k", "~/.kube/config", "Path to a kubeconfig")
	cmd.Flags().StringP("name", "n", "", "Kubernetes cluster name")
	cmd.Flags().String("namespace", "percona-everest", "Namespace into which Percona Everest components are deployed to")
	cmd.Flags().Bool("skip-wizard", false, "Skip installation wizard")

	cmd.Flags().Bool("operator.mongodb", true, "Install MongoDB operator")
	cmd.Flags().Bool("operator.postgresql", true, "Install PostgreSQL operator")
	cmd.Flags().Bool("operator.xtradb-cluster", true, "Install XtraDB Cluster operator")

	cmd.Flags().String("channel.everest", "stable-v0", "Channel for Everest operator")
	cmd.Flags().String("channel.victoria-metrics", "stable-v0", "Channel for VictoriaMetrics operator")
	cmd.Flags().String("channel.xtradb-cluster", "stable-v1", "Channel for XtraDB Cluster operator")
	cmd.Flags().String("channel.mongodb", "stable-v1", "Channel for MongoDB operator")
	cmd.Flags().String("channel.postgresql", "fast-v2", "Channel for PostgreSQL operator")
}

func initInstallViperFlags(cmd *cobra.Command) {
	viper.BindPFlag("skip-wizard", cmd.Flags().Lookup("skip-wizard")) //nolint:errcheck,gosec

	viper.BindEnv("kubeconfig")                                     //nolint:errcheck,gosec
	viper.BindPFlag("kubeconfig", cmd.Flags().Lookup("kubeconfig")) //nolint:errcheck,gosec
	viper.BindPFlag("name", cmd.Flags().Lookup("name"))             //nolint:errcheck,gosec
	viper.BindPFlag("namespace", cmd.Flags().Lookup("namespace"))   //nolint:errcheck,gosec

	viper.BindPFlag("operator.mongodb", cmd.Flags().Lookup("operator.mongodb"))               //nolint:errcheck,gosec
	viper.BindPFlag("operator.postgresql", cmd.Flags().Lookup("operator.postgresql"))         //nolint:errcheck,gosec
	viper.BindPFlag("operator.xtradb-cluster", cmd.Flags().Lookup("operator.xtradb-cluster")) //nolint:errcheck,gosec

	viper.BindPFlag("channel.victoria-metrics", cmd.Flags().Lookup("channel.victoria-metrics")) //nolint:errcheck,gosec
	viper.BindPFlag("channel.xtradb-cluster", cmd.Flags().Lookup("channel.xtradb-cluster"))     //nolint:errcheck,gosec
	viper.BindPFlag("channel.mongodb", cmd.Flags().Lookup("channel.mongodb"))                   //nolint:errcheck,gosec
	viper.BindPFlag("channel.postgresql", cmd.Flags().Lookup("channel.postgresql"))             //nolint:errcheck,gosec
	viper.BindPFlag("channel.everest", cmd.Flags().Lookup("channel.everest"))                   //nolint:errcheck,gosec
}
