// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2021 Intel Corporation

package daemon

import (
	"context"
	"errors"

	dh "github.com/rmr-silicom/openshift-operator/common/pkg/drainhelper"

	"github.com/go-logr/logr"
	fpgav1 "github.com/rmr-silicom/openshift-operator/N5010/api/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type FlashConditionReason string

const (
	// FlashCondition flash condition name
	FlashCondition string = "Flashed"

	// Failed indicates that the flashing is in an unknown state
	FlashUnknown FlashConditionReason = "Unknown"
	// FlashInProgress indicates that the flashing process is in progress
	FlashInProgress FlashConditionReason = "InProgress"
	// FlashFailed indicates that the flashing process failed
	FlashFailed FlashConditionReason = "Failed"
	// FlashNotRequested indicates that the flashing process was not requested
	FlashNotRequested FlashConditionReason = "NotRequested"
	// FlashSucceeded indicates that the flashing process succeeded
	FlashSucceeded FlashConditionReason = "Succeeded"
)

type N5010NodeReconciler struct {
	client.Client
	log       logr.Logger
	nodeName  string
	namespace string

	fpga FPGAManager

	drainHelper *dh.DrainHelper
}

func NewN5010NodeReconciler(c client.Client, clientSet *clientset.Clientset, log logr.Logger,
	nodename, namespace string) *N5010NodeReconciler {

	return &N5010NodeReconciler{
		Client:    c,
		log:       log,
		nodeName:  nodename,
		namespace: namespace,
		fpga: FPGAManager{
			Log: log.WithName("fpgaManager"),
		},
		drainHelper: dh.NewDrainHelper(log, clientSet, nodename, namespace),
	}
}

func (r *N5010NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&fpgav1.N5010Node{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				N5010node, ok := e.Object.(*fpgav1.N5010Node)
				if !ok {
					r.log.V(2).Info("Failed to convert e.Object to fpgav1.N5010Node", "e.Object", e.Object)
					return false
				}
				cond := meta.FindStatusCondition(N5010node.Status.Conditions, FlashCondition)
				if cond != nil && cond.ObservedGeneration == e.Object.GetGeneration() {
					r.log.V(4).Info("Created object was handled previously, ignoring")
					return false
				}
				return true

			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				if e.ObjectOld.GetGeneration() == e.ObjectNew.GetGeneration() {
					r.log.V(4).Info("Update ignored, generation unchanged")
					return false
				}
				return true
			},
		}).
		Complete(r)
}

// CreateEmptyN5010NodeIfNeeded creates empty CR to be Reconciled in near future and filled with Status.
// If invoked before manager's Start, it'll need a direct API client
// (Manager's/Controller's client is cached and cache is not initialized yet).
func (r *N5010NodeReconciler) CreateEmptyN5010NodeIfNeeded(c client.Client) error {
	log := r.log.WithName("CreateEmptyN5010NodeIfNeeded").WithValues("name", r.nodeName, "namespace", r.namespace)

	N5010node := &fpgav1.N5010Node{}
	err := c.Get(context.Background(),
		client.ObjectKey{
			Name:      r.nodeName,
			Namespace: r.namespace,
		},
		N5010node)

	if err == nil {
		log.V(4).Info("already exists")
		return nil
	}

	if k8serrors.IsNotFound(err) {
		log.V(2).Info("not found - creating")

		N5010node = &fpgav1.N5010Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.nodeName,
				Namespace: r.namespace,
			},
		}

		return c.Create(context.Background(), N5010node)
	}

	return err
}

func (r *N5010NodeReconciler) getNodeStatus(n *fpgav1.N5010Node) (fpgav1.N5010NodeStatus, error) {
	log := r.log.WithName("getNodeStatus")

	fpgaStatus, err := getFPGAInventory(r.log)
	if err != nil {
		log.Error(err, "Failed to get FPGA inventory")
		return fpgav1.N5010NodeStatus{}, err
	}

	return fpgav1.N5010NodeStatus{
		FPGA: fpgaStatus,
	}, nil
}

func (r *N5010NodeReconciler) updateStatus(n *fpgav1.N5010Node, c []metav1.Condition) error {
	log := r.log.WithName("updateStatus")

	nodeStatus, err := r.getNodeStatus(n)
	if err != nil {
		log.Error(err, "failed to get N5010Node status")
		return err
	}

	for _, condition := range c {
		meta.SetStatusCondition(&nodeStatus.Conditions, condition)
	}
	n.Status = nodeStatus
	if err := r.Status().Update(context.Background(), n); err != nil {
		log.Error(err, "failed to update N5010Node status")
		return err
	}

	return nil
}

func (r *N5010NodeReconciler) updateFlashCondition(n *fpgav1.N5010Node, status metav1.ConditionStatus,
	reason FlashConditionReason, msg string) {
	log := r.log.WithName("updateFlashCondition")
	fc := metav1.Condition{
		Type:               FlashCondition,
		Status:             status,
		Reason:             string(reason),
		Message:            msg,
		ObservedGeneration: n.GetGeneration(),
	}
	if err := r.updateStatus(n, []metav1.Condition{fc}); err != nil {
		log.Error(err, "failed to update N5010Node flash condition")
	}
}

func (r *N5010NodeReconciler) verifySpec(n *fpgav1.N5010Node) error {
	for _, f := range n.Spec.FPGA {
		if f.UserImageURL == "" {
			return errors.New("Missing UserImageURL for PCI: " + f.PCIAddr)
		}
	}

	return nil
}

func (r *N5010NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithName("Reconcile").WithValues("namespace", req.Namespace, "name", req.Name)

	// Lack of update on namespace/name mismatch (like in N5010ClusterReconciler):
	// N5010Node is between Operator & Daemon so we're in control of this communication channel.

	if req.Namespace != r.namespace {
		log.V(4).Info("unexpected namespace - ignoring", "expected namespace", r.namespace)
		return ctrl.Result{}, nil
	}

	if req.Name != r.nodeName {
		log.V(4).Info("CR intended for another node - ignoring", "expected name", r.nodeName)
		return ctrl.Result{}, nil
	}

	N5010node := &fpgav1.N5010Node{}
	if err := r.Client.Get(ctx, req.NamespacedName, N5010node); err != nil {
		if k8serrors.IsNotFound(err) {
			log.V(2).Info("reconciled N5010node not found")
			return ctrl.Result{}, r.CreateEmptyN5010NodeIfNeeded(r.Client)
		}
		log.Error(err, "Get(N5010node) failed")
		return ctrl.Result{}, err
	}

	err := r.verifySpec(N5010node)
	if err != nil {
		log.Error(err, "verifySpec error")
		r.updateFlashCondition(N5010node, metav1.ConditionFalse, FlashFailed, err.Error())
		return ctrl.Result{}, nil
	}

	if N5010node.Spec.FPGA == nil {
		log.V(4).Info("Nothing to do")
		r.updateFlashCondition(N5010node, metav1.ConditionFalse, FlashNotRequested, "Inventory up to date")
		return ctrl.Result{}, nil
	}

	// Update current condition to reflect that the flash started
	currentCondition := meta.FindStatusCondition(N5010node.Status.Conditions, FlashCondition)
	if currentCondition != nil {
		currentCondition.Status = metav1.ConditionFalse
		currentCondition.Reason = string(FlashInProgress)
		currentCondition.Message = "Flash started"
		if err := r.updateStatus(N5010node, []metav1.Condition{*currentCondition}); err != nil {
			log.Error(err, "failed to update current N5010Node flash condition")
			return ctrl.Result{}, err
		}
	}

	if N5010node.Spec.FPGA != nil {
		err := r.fpga.verifyPreconditions(N5010node)
		if err != nil {
			r.updateFlashCondition(N5010node, metav1.ConditionFalse, FlashFailed, err.Error())
			return ctrl.Result{}, nil
		}
	}

	var flashErr error
	err = r.drainHelper.Run(func(c context.Context) bool {
		if N5010node.Spec.FPGA != nil {
			err := r.fpga.ProgramFPGAs(N5010node)
			if err != nil {
				log.Error(err, "Unable to flash FPGA")
				flashErr = err
				return true
			}
		}

		return true
	}, !N5010node.Spec.DrainSkip)

	if err != nil {
		// some kind of error around leader election / node (un)cordon / node drain
		r.updateFlashCondition(N5010node, metav1.ConditionUnknown, FlashUnknown, err.Error())
		return ctrl.Result{}, nil
	}

	if flashErr != nil {
		r.updateFlashCondition(N5010node, metav1.ConditionFalse, FlashFailed, flashErr.Error())
	} else {
		r.updateFlashCondition(N5010node, metav1.ConditionTrue, FlashSucceeded, "Flashed successfully")
	}

	log.V(2).Info("Reconciled")
	return ctrl.Result{}, nil
}
