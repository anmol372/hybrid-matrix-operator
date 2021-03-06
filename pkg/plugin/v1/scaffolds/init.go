/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scaffolds

import (
	"fmt"
	"os"

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin/scaffold"

	"github.com/joelanford/helm-operator/pkg/internal/kubebuilder/machinery"
	"github.com/joelanford/helm-operator/pkg/plugin/internal/chartutil"
	"github.com/joelanford/helm-operator/pkg/plugin/v1/scaffolds/internal/templates"
	"github.com/joelanford/helm-operator/pkg/plugin/v1/scaffolds/internal/templates/manager"
	"github.com/joelanford/helm-operator/pkg/plugin/v1/scaffolds/internal/templates/metricsauth"
	"github.com/joelanford/helm-operator/pkg/plugin/v1/scaffolds/internal/templates/prometheus"
)

const (
	// KustomizeVersion is the kubernetes-sigs/kustomize version to be used in the project
	KustomizeVersion = "v3.5.4"

	imageName = "controller:latest"
)

var _ scaffold.Scaffolder = &initScaffolder{}

type initScaffolder struct {
	config        *config.Config
	apiScaffolder scaffold.Scaffolder
}

// NewInitScaffolder returns a new Scaffolder for project initialization operations
func NewInitScaffolder(config *config.Config, apiScaffolder scaffold.Scaffolder) scaffold.Scaffolder {
	return &initScaffolder{
		config:        config,
		apiScaffolder: apiScaffolder,
	}
}

func (s *initScaffolder) newUniverse() *model.Universe {
	return model.NewUniverse(
		model.WithConfig(s.config),
	)
}

// Scaffold implements Scaffolder
func (s *initScaffolder) Scaffold() error {
	switch {
	case s.config.IsV3():
		if err := s.scaffold(); err != nil {
			return err
		}
		if s.apiScaffolder != nil {
			return s.apiScaffolder.Scaffold()
		}
		return nil
	default:
		return fmt.Errorf("unknown project version %v", s.config.Version)
	}
}

func (s *initScaffolder) scaffold() error {
	if err := os.MkdirAll(chartutil.HelmChartsDir, 0755); err != nil {
		return err
	}
	return machinery.NewScaffold().Execute(
		s.newUniverse(),
		&templates.GitIgnore{},
		&templates.AuthProxyRole{},
		&templates.AuthProxyRoleBinding{},
		&metricsauth.AuthProxyPatch{},
		&metricsauth.AuthProxyService{},
		&metricsauth.ClientClusterRole{},
		&manager.Config{Image: imageName},
		&templates.Makefile{
			Image:            imageName,
			KustomizeVersion: KustomizeVersion,
		},
		&templates.Dockerfile{},
		&templates.Kustomize{},
		&templates.ManagerRoleBinding{},
		&templates.LeaderElectionRole{},
		&templates.LeaderElectionRoleBinding{},
		&templates.KustomizeRBAC{},
		&templates.Watches{},
		&manager.Kustomization{},
		&prometheus.Kustomization{},
		&prometheus.ServiceMonitor{},
	)
}
