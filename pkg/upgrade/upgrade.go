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

// Package upgrade implements upgrade logic for the CLI.
package upgrade

import (
	"context"
	"errors"
	"net/url"
	"os"

	"github.com/AlecAivazis/survey/v2"
	goversion "github.com/hashicorp/go-version"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/percona/percona-everest-cli/data"
	"github.com/percona/percona-everest-cli/pkg/kubernetes"
)

type (
	// Config defines configuration required for upgrade command.
	Config struct {
		// Name of the Kubernetes Cluster
		Name string
		// Namespace defines the namespace operators shall be installed to.
		Namespace string
		// KubeconfigPath is a path to a kubeconfig
		KubeconfigPath string `mapstructure:"kubeconfig"`
		// UpgradeOLM defines do we need to upgrade OLM or not.
		UpgradeOLM bool `mapstructure:"upgrade-olm"`
		// SkipWizard skips wizard during installation.
		SkipWizard bool `mapstructure:"skip-wizard"`
	}
	// Upgrade struct implements upgrade command.
	Upgrade struct {
		l *zap.SugaredLogger

		config     Config
		kubeClient *kubernetes.Kubernetes
	}
)

// NewUpgrade returns a new Upgrade struct.
func NewUpgrade(c Config, l *zap.SugaredLogger) (*Upgrade, error) {
	cli := &Upgrade{
		config: c,
		l:      l.With("component", "upgrade"),
	}

	k, err := kubernetes.New(c.KubeconfigPath, cli.l)
	if err != nil {
		var u *url.Error
		if errors.As(err, &u) {
			cli.l.Error("Could not connect to Kubernetes. " +
				"Make sure Kubernetes is running and is accessible from this computer/server.")
		}
		return nil, err
	}
	cli.kubeClient = k
	return cli, nil
}

// Run runs the operators installation process.
func (u *Upgrade) Run(ctx context.Context) error {
	if err := u.upgradeOLM(ctx); err != nil {
		return err
	}
	u.l.Info("Upgrading Percona Catalog")
	if err := u.kubeClient.InstallPerconaCatalog(ctx); err != nil {
		return err
	}
	u.l.Info("Percona Catalog has been upgraded")
	u.l.Info("Patching subscriptions")
	if err := u.patchSubscriptions(ctx); err != nil {
		return err
	}
	u.l.Info("Subscriptions have been patched")
	u.l.Info("Upgrading Everest")
	if err := u.kubeClient.InstallEverest(ctx, u.config.Namespace); err != nil {
		return err
	}
	u.l.Info("Everest has been upgraded")
	return nil
}

func (u *Upgrade) patchSubscriptions(ctx context.Context) error {
	subList, err := u.kubeClient.ListSubscriptions(ctx, u.config.Namespace)
	if err != nil {
		return err
	}
	disableTelemetryEnvVar := "DISABLE_TELEMETRY"
	disableTelemetry, ok := os.LookupEnv(disableTelemetryEnvVar)
	if !ok || disableTelemetry != "true" {
		disableTelemetry = "false"
	}
	for _, subscription := range subList.Items {
		subscription := subscription
		subscription.Spec.Config = &v1alpha1.SubscriptionConfig{
			Env: []corev1.EnvVar{
				{
					Name:  disableTelemetryEnvVar,
					Value: disableTelemetry,
				},
			},
		}
		if err := u.kubeClient.ApplyObject(&subscription); err != nil {
			return err
		}
	}
	return nil
}

func (u *Upgrade) upgradeOLM(ctx context.Context) error {
	csv, err := u.kubeClient.GetClusterServiceVersion(ctx, types.NamespacedName{
		Name:      "packageserver",
		Namespace: "olm",
	})
	if err != nil {
		return err
	}
	foundVersion, err := goversion.NewSemver(csv.Spec.Version.String())
	if err != nil {
		return err
	}
	shippedVersion, err := goversion.NewSemver(data.OLMVersion)
	if err != nil {
		return err
	}
	if foundVersion.GreaterThan(shippedVersion) {
		// Nothing to do here. Installed OLM is greater that we ship with the CLI.
		return nil
	}
	if foundVersion.Equal(shippedVersion) {
		// Nothing to do here. OLM is upgraded.
		return nil
	}
	if !u.config.SkipWizard {
		if err := u.runWizard(); err != nil {
			return err
		}
	}
	if u.config.UpgradeOLM {
		u.l.Info("Upgrading OLM")
		if err := u.kubeClient.InstallOLMOperator(ctx, true); err != nil {
			return err
		}
		u.l.Info("OLM has been upgraded")
	}
	return nil
}

// runWizard runs installation wizard.
func (u *Upgrade) runWizard() error {
	pOLM := &survey.Confirm{
		Message: "Do you want to upgrade OLM?",
		Default: u.config.UpgradeOLM,
	}
	return survey.AskOne(pOLM, &u.config.UpgradeOLM)
}
