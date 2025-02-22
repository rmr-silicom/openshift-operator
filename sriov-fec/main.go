// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2021 Intel Corporation

/*


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

package main

import (
	"context"
	"flag"
	"os"
	"strings"
	"time"

	secv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/open-ness/openshift-operator/common/pkg/assets"
	sriovfecv1 "github.com/open-ness/openshift-operator/sriov-fec/api/v1"
	"github.com/open-ness/openshift-operator/sriov-fec/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme                 = runtime.NewScheme()
	setupLog               = ctrl.Log.WithName("setup")
	operatorDeploymentName string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(secv1.AddToScheme(scheme))
	utilruntime.Must(sriovfecv1.AddToScheme(scheme))

	n := os.Getenv("NAME")
	operatorDeploymentName = n[:strings.LastIndex(n[:strings.LastIndex(n, "-")], "-")]
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var healthProbeAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&healthProbeAddr, "health-probe-bind-address", ":8081", "The address the controller binds to for serving health probes.")
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	config := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: healthProbeAddr,
		Port:                   9443,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "98e78623.intel.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.SriovFecClusterConfigReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SriovFecClusterConfig"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SriovFecClusterConfig")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "failed to create client")
		os.Exit(1)
	}

	namespace := os.Getenv("SRIOV_FEC_NAMESPACE")
	owner := &appsv1.Deployment{}
	err = c.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      operatorDeploymentName,
	}, owner)
	if err != nil {
		setupLog.Error(err, "Unable to get operator deployment")
		os.Exit(1)
	}

	if err := (&assets.Manager{
		Client:    c,
		Log:       ctrl.Log.WithName("asset_manager").WithName("sriov-fec"),
		EnvPrefix: "SRIOV_FEC_",
		Scheme:    scheme,
		Owner:     owner,
		Assets: []assets.Asset{
			{
				Path: "assets/100-labeler.yaml",
			},
			{
				Path: "assets/200-device-plugin.yaml",
			},
			{
				Path:              "assets/300-daemon.yaml",
				BlockingReadiness: assets.ReadinessPollConfig{Retries: 30, Delay: 20 * time.Second},
			},
		},
	}).LoadAndDeploy(context.Background(), false); err != nil {
		setupLog.Error(err, "failed to deploy the assets")
		os.Exit(1)
	}

	setupLog.V(2).Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
