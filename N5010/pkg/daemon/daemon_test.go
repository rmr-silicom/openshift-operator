// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2021 Intel Corporation

package daemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"

	fpgav1 "github.com/rmr-silicom/openshift-operator/N5010/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func setupManagers() {
	fpgaInfoExec = fakeFpgaInfo

	fpgasUpdateExec = fakeFpgasUpdate
	rsuExec = fakeRsu

	fakeFpgaInfoErrReturn = fmt.Errorf("error")

	//flags
	fakeFpgaInfoErrReturn = nil
	fakeFpgasUpdateErrReturn = nil
	fakeRsuUpdateErrReturn = nil
}
func cleanUpHandlers() {

	// Restore original FPGA manager handlers
	fpgaInfoExec = runExec
	fpgasUpdateExec = runExecWithLog
	rsuExec = runExecWithLog
}

var reportErrorIn = 0

func fakeFpgaInfoDelayed(cmd *exec.Cmd, log logr.Logger, dryRun bool) (string, error) {
	fmt.Printf("  ** ** || ** GFGF: fakeFpgaInfoDelayed: reportErrorIn: %d\n", reportErrorIn)
	if reportErrorIn == 0 {
		return "", fmt.Errorf("error")
	}

	reportErrorIn--

	return fakeFpgaInfo(cmd, log, dryRun)
}

var _ = Describe("N5010 Daemon Tests", func() {

	var clusterConfig *fpgav1.N5010Cluster

	var n5010node *fpgav1.N5010Node

	var request ctrl.Request
	var reconciler N5010NodeReconciler

	const tempNamespaceName = "n5010node"
	var namespace = os.Getenv("INTEL_FPGA_NAMESPACE")

	log := klogr.New()
	doDeconf := false
	removeCluster := false

	setupManagers()

	var _ = Describe("Reconciler functionalities", func() {
		BeforeEach(func() {
			cleanFPGA()
			cleanUpHandlers()
			doDeconf = false
			removeCluster = false

			n5010node = &fpgav1.N5010Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gf",
					Namespace: namespace,
				},
			}

			clusterConfig = &fpgav1.N5010Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tempNamespaceName,
					Namespace: namespace,
				},
				Spec: fpgav1.N5010ClusterSpec{
					Nodes: []fpgav1.N5010ClusterNode{
						{
							NodeName: "dummy",
						},
					},
				},
			}

			setupManagers()
		})

		AfterEach(func() {
			var err error
			if doDeconf {
				clusterConfig.Spec = fpgav1.N5010ClusterSpec{
					Nodes: []fpgav1.N5010ClusterNode{},
				}

				err = k8sClient.Update(context.TODO(), clusterConfig)
				Expect(err).NotTo(HaveOccurred())
				_, err = (reconciler).Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
			}

			if removeCluster {
				err = k8sClient.Delete(context.TODO(), clusterConfig)
				Expect(err).ToNot(HaveOccurred())
			}

			// Remove nodes
			nodes := &fpgav1.N5010NodeList{}
			err = k8sClient.List(context.TODO(), nodes)
			Expect(err).ToNot(HaveOccurred())

			for _, nodeToDelete := range nodes.Items {
				err = k8sClient.Delete(context.TODO(), &nodeToDelete)
				Expect(err).ToNot(HaveOccurred())
			}

			cleanUpHandlers()
		})

		var _ = It("check NewN5010NodeReconciler", func() {

			var clientSet clientset.Clientset
			const nodeName = "FakeNodeName"
			const namespaceName = "FakeNamespace"
			log = klogr.New().WithName("N5010NodeReconciler-Test")

			recon := NewN5010NodeReconciler(k8sClient, &clientSet, log, nodeName, namespaceName)

			Expect(recon).ToNot(Equal(nil))
			Expect(recon.nodeName).To(Equal(nodeName))
			Expect(recon.namespace).To(Equal(namespaceName))
			Expect(recon.Client).To(Equal(k8sClient))
		})

		var _ = It("check updateFlashCondition 2", func() {

			n5010node = &fpgav1.N5010Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gfgf",
					Namespace: namespace,
				},
			}
			err := k8sClient.Create(context.Background(), n5010node)
			Expect(err).ToNot(HaveOccurred())
			log = klogr.New().WithName("N5010NodeReconciler-Test")

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log,
				namespace: namespace,
				nodeName:  "dummy",
				fpga: FPGAManager{
					Log: log.WithName("fpgaManager"),
				},
			}

			Expect(reconciler).ToNot(Equal(nil))

			reconciler.updateFlashCondition(n5010node, metav1.ConditionFalse, FlashFailed, "OK")
		})

		var _ = It("check updateFlashCondition no n5010node", func() {

			log = klogr.New().WithName("N5010NodeReconciler-Test")

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log,
				namespace: namespace,
				nodeName:  "dummy",
				fpga: FPGAManager{
					Log: log.WithName("fpgaManager"),
				},
			}

			Expect(reconciler).ToNot(Equal(nil))

			reconciler.updateFlashCondition(n5010node, metav1.ConditionFalse, FlashFailed, "OK")
		})

		var _ = It("check updateFlashCondition", func() {
			err := k8sClient.Create(context.Background(), n5010node)
			Expect(err).ToNot(HaveOccurred())

			log = klogr.New().WithName("N5010NodeReconciler-Test")

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log,
				namespace: namespace,
				nodeName:  "dummy",
				fpga: FPGAManager{
					Log: log.WithName("fpgaManager"),
				},
			}

			Expect(reconciler).ToNot(Equal(nil))

			reconciler.updateFlashCondition(n5010node, metav1.ConditionFalse, FlashFailed, "OK")
		})

		var _ = It("check updateFlashCondition True", func() {

			var err error
			n5010node = &fpgav1.N5010Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gf2",
					Namespace: namespace,
				},
			}

			log = klogr.New().WithName("N5010NodeReconciler-Test")

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log,
				namespace: namespace,
				nodeName:  "dummy",
				fpga: FPGAManager{
					Log: log.WithName("fpgaManager"),
				},
			}

			Expect(reconciler).ToNot(Equal(nil))

			err = (reconciler).CreateEmptyN5010NodeIfNeeded(k8sClient)
			Expect(err).ToNot(HaveOccurred())

			reconciler.updateFlashCondition(n5010node, metav1.ConditionTrue, FlashFailed, "OK")
		})

		var _ = It("check updateFlash failure ", func() {

			n5010node = &fpgav1.N5010Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gfgf",
					Namespace: namespace,
				},
			}
			err := k8sClient.Create(context.Background(), n5010node)
			Expect(err).ToNot(HaveOccurred())
			log = klogr.New().WithName("N5010NodeReconciler-Test")

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log,
				namespace: namespace,
				nodeName:  "dummy",
				fpga: FPGAManager{
					Log: log.WithName("fpgaManager"),
				},
			}

			Expect(reconciler).ToNot(Equal(nil))

			fc := metav1.Condition{
				Type:               FlashCondition,
				Status:             metav1.ConditionFalse,
				Reason:             string(FlashFailed),
				Message:            "message",
				ObservedGeneration: n5010node.GetGeneration(),
			}

			reportErrorIn = 1
			fpgaInfoExec = fakeFpgaInfoDelayed

			// Error reported by FPGA Manager
			err = reconciler.updateStatus(n5010node, []metav1.Condition{fc})
			Expect(err).To(HaveOccurred())
			Expect(reportErrorIn).To(Equal(0))

			// Error reported by Fortville Manager
			err = reconciler.updateStatus(n5010node, []metav1.Condition{fc})
			Expect(err).To(HaveOccurred())

			// restore default value
			fpgaInfoExec = fakeFpgaInfo
		})

		var _ = It("check verifySpec", func() {
			var err error

			reconciler = N5010NodeReconciler{}

			var emptyNode fpgav1.N5010Node
			err = reconciler.verifySpec(&emptyNode)
			Expect(err).ToNot(HaveOccurred())

			var noUserimageUrlNode fpgav1.N5010Node

			noUserimageUrlNode.Spec.FPGA = []fpgav1.N5010Fpga{
				{
					PCIAddr:      "PCI1",
					UserImageURL: "someUrl",
				},
				{
					PCIAddr:      "PCI2",
					UserImageURL: "",
				},
			}
			err = reconciler.verifySpec(&noUserimageUrlNode)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PCI2"))
		})

		var _ = It("will create node config", func() {
			var err error

			err = k8sClient.Create(context.Background(), n5010node)
			Expect(err).ToNot(HaveOccurred())

			// simulate creation of cluster config by the user

			log = klogr.New().WithName("N5010NodeReconciler-Test")
			request = ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      tempNamespaceName,
				},
			}

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log,
				namespace: request.NamespacedName.Namespace,
				nodeName:  "dummy"}

			_, err = (reconciler).Reconcile(context.TODO(), request)
			Expect(err).ToNot(HaveOccurred())
		})

		var _ = It("fail because of node not found", func() {
			var err error

			// simulate creation of cluster config by the user

			log = klogr.New().WithName("N5010NodeReconciler-Test")
			request = ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      "gf",
				},
			}

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log,
				namespace: request.NamespacedName.Namespace,
				nodeName:  "gf"}

			_, err = (reconciler).Reconcile(context.TODO(), request)
			Expect(err).ToNot(HaveOccurred())
		})

		var _ = It("fail because of wrong node name", func() {
			var err error

			// simulate creation of cluster config by the user

			log = klogr.New().WithName("N5010NodeReconciler-Test")

			request = ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      tempNamespaceName,
				},
			}

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log, namespace: request.NamespacedName.Namespace, nodeName: "123NodeName"}

			_, err = (reconciler).Reconcile(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
		})

		var _ = It("fail because of missing node name", func() {
			var err error

			// simulate creation of cluster config by the user

			log = klogr.New().WithName("N5010NodeReconciler-Test")

			request = ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      tempNamespaceName,
				},
			}

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log, namespace: request.NamespacedName.Namespace}

			_, err = (reconciler).Reconcile(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconciler.nodeName).To(Equal(""))
		})

		var _ = It("fail because of wrong namespace, but no error", func() {
			var err error
			// simulate creation of cluster config by the user

			log = klogr.New().WithName("N5010NodeReconciler-Test")
			request = ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      tempNamespaceName,
				},
			}

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log}

			_, err = (reconciler).Reconcile(context.TODO(), request)
			Expect(err).ToNot(HaveOccurred())
			Expect(request.Namespace).ToNot(Equal(reconciler.namespace))
		})

		var _ = It("will fail to create node config because of missing MACS and FPGA", func() {
			var err error

			err = k8sClient.Create(context.Background(), n5010node)
			Expect(err).ToNot(HaveOccurred())

			// simulate creation of cluster config by the user

			log = klogr.New().WithName("N5010NodeReconciler-Test")
			request = ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      "gf",
				},
			}

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log,
				namespace: request.NamespacedName.Namespace,
				nodeName:  "gf",
				fpga: FPGAManager{
					Log: log.WithName("fpgaManager"),
				},
			}

			_, err = (reconciler).Reconcile(context.TODO(), request)
			Expect(err).ToNot(HaveOccurred())
		})

		var _ = It("will fail with wrong FPGA preconditions", func() {
			var err error

			n5010node.Spec.FPGA = []fpgav1.N5010Fpga{
				{
					PCIAddr:      "ffff:ff:01.1",
					UserImageURL: "/tmp/fake.bin",
				},
			}

			err = k8sClient.Create(context.Background(), n5010node)
			Expect(err).ToNot(HaveOccurred())

			// simulate creation of cluster config by the user

			log = klogr.New().WithName("N5010NodeReconciler-Test")
			request = ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      "gf",
				},
			}

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log,
				namespace: request.NamespacedName.Namespace,
				nodeName:  "gf",
				fpga: FPGAManager{
					Log: log.WithName("fpgaManager"),
				},
			}

			_, err = (reconciler).Reconcile(context.TODO(), request)
			Expect(err).ToNot(HaveOccurred())
		})
		var _ = It("will fail because of Flash problem", func() {
			var err error

			n5010node.Spec.FPGA = nil

			err = k8sClient.Create(context.Background(), n5010node)
			Expect(err).ToNot(HaveOccurred())

			// simulate creation of cluster config by the user

			log = klogr.New().WithName("N5010NodeReconciler-Test")
			request = ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      "gf",
				},
			}

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log,
				namespace: request.NamespacedName.Namespace,
				nodeName:  "gf",
				fpga: FPGAManager{
					Log: log.WithName("fpgaManager"),
				},
			}

			_, err = (reconciler).Reconcile(context.TODO(), request)
			Expect(err).ToNot(HaveOccurred())
		})

		var _ = It("check CreateEmptyN5010NodeIfNeeded", func() {
			var err error

			err = k8sClient.Create(context.Background(), n5010node)
			Expect(err).ToNot(HaveOccurred())

			// simulate creation of cluster config by the user

			log = klogr.New().WithName("N5010NodeReconciler-Test")

			reconciler = N5010NodeReconciler{Client: k8sClient, log: log,
				namespace: namespace,
				nodeName:  "gf"}

			nodes := &fpgav1.N5010NodeList{}
			err = k8sClient.List(context.TODO(), nodes)
			Expect(err).ToNot(HaveOccurred())

			err = (reconciler).CreateEmptyN5010NodeIfNeeded(k8sClient)
			Expect(err).ToNot(HaveOccurred())

			err = (reconciler).CreateEmptyN5010NodeIfNeeded(k8sClient)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	var _ = Describe("Reconciler manager", func() {
		var _ = It("setup with invalid manager", func() {
			var m ctrl.Manager
			var reconciler N5010NodeReconciler

			err := reconciler.SetupWithManager(m)
			Expect(err).To(HaveOccurred())
		})
	})
})
