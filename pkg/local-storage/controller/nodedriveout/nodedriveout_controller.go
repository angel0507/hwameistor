package nodedriveout

import (
	"context"
	apis "github.com/hwameistor/hwameistor/pkg/apis/hwameistor/local-storage"
	apisv1alpha1 "github.com/hwameistor/hwameistor/pkg/apis/hwameistor/local-storage/v1alpha1"
	"github.com/hwameistor/hwameistor/pkg/driveout"
	"github.com/hwameistor/hwameistor/pkg/local-storage/member"
	logr "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new NodeDriveout Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNodeDriveout{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("nodedriveout-controller"),
		// storageMember is a global variable
		storageMember: member.Member(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("NodeDriveout-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource NodeDriveout
	err = c.Watch(&source.Kind{Type: &apisv1alpha1.NodeDriveout{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileNodeDriveout implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNodeDriveout{}

// ReconcileNodeDriveout reconciles a NodeDriveout object
type ReconcileNodeDriveout struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	Recorder      record.EventRecorder
	storageMember apis.LocalStorageMember
}

// Reconcile reads that state of the cluster for a NodeDriveout object and makes changes based on the state read
// and what is in the NodeDriveout.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNodeDriveout) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logr.Debug("ReconcileNodeDriveout  Reconcile start")

	instance := &apisv1alpha1.NodeDriveout{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	logr.Debug("ReconcileNodeDriveout  Reconcile instance = %v", instance)

	ndohandler := driveout.NewNodeDriveoutHandler(r.client)
	ndohandler = ndohandler.SetNodeDriveout(*instance)
	ndohandler, err = ndohandler.Refresh()
	if err != nil {
		logr.Error(err, "ReconcileNodeDriveout Refresh failed")
		return reconcile.Result{Requeue: true}, err
	}
	ndoStatus := ndohandler.NodeDriveoutStatus()
	logr.Debug("ReconcileNodeDriveout  instance.Spec.NodeDriveoutStage = %v, ndoStatus = %v", instance.Spec.NodeDriveoutStage, ndoStatus)

	switch instance.Spec.NodeDriveoutStage {
	case "":
		logr.Debug("ndohandler.NodeDriveoutStage() = %v, ndohandler.NodeDriveoutStatus() = %v", ndohandler.NodeDriveoutStage(), ndohandler.NodeDriveoutStatus())
		ndohandler = ndohandler.SetNodeDriveoutStage(apisv1alpha1.DriveoutStage_Init)
		err := ndohandler.UpdateNodeDriveoutCR()
		if err != nil {
			logr.Error(err, "UpdateNodeDriveoutCR SetNodeDriveoutStage DriveoutStage_Init failed")
			return reconcile.Result{Requeue: true}, nil
		}
		ndoStatus.DriveoutStatus = apisv1alpha1.Driveout_Init
		if err := ndohandler.UpdateNodeDriveoutStatus(ndoStatus); err != nil {
			logr.Error(err, "DriveoutStage_Init UpdateReplaceDiskStatus ReplaceDisk_Init,ReplaceDisk_Init failed")
			return reconcile.Result{Requeue: true}, nil
		}
		return reconcile.Result{Requeue: true}, nil

	case apisv1alpha1.DriveoutStage_Init:
		logr.Debug("DriveoutStage_Init ndohandler.NodeDriveoutStage() = %v, ndohandler.NodeDriveoutStatus() = %v", ndohandler.NodeDriveoutStage(), ndohandler.NodeDriveoutStatus())

	case apisv1alpha1.DriveoutStage_StopSvc:
		logr.Debug("DriveoutStage_StopSvc ndohandler.NodeDriveoutStage() = %v, ndohandler.NodeDriveoutStatus() = %v", ndohandler.NodeDriveoutStage(), ndohandler.NodeDriveoutStatus())
		ndoStatus.DriveoutStatus = apisv1alpha1.Driveout_WaitStopSvc
		if err := ndohandler.UpdateNodeDriveoutStatus(ndoStatus); err != nil {
			logr.Error(err, "DriveoutStage_StopSvc UpdateNodeDriveoutStatus Driveout_WaitStopSvc failed")
			return reconcile.Result{Requeue: true}, nil
		}

	case apisv1alpha1.DriveoutStage_DrivingOut:
		logr.Debug("DriveoutStage_DrivingOut ndohandler.NodeDriveoutStage() = %v, ndohandler.NodeDriveoutStatus() = %v", ndohandler.NodeDriveoutStage(), ndohandler.NodeDriveoutStatus())
		ndoStatus.DriveoutStatus = apisv1alpha1.Driveout_WaitDriveOut
		if err := ndohandler.UpdateNodeDriveoutStatus(ndoStatus); err != nil {
			logr.Error(err, "DriveoutStage_DrivingOut UpdateNodeDriveoutStatus Driveout_WaitDriveOut failed")
			return reconcile.Result{Requeue: true}, nil
		}

	case apisv1alpha1.DriveoutStage_StartSvc:
		logr.Debug("DriveoutStage_StartSvc ndohandler.NodeDriveoutStage() = %v, ndohandler.NodeDriveoutStatus() = %v", ndohandler.NodeDriveoutStage(), ndohandler.NodeDriveoutStatus())
		ndoStatus.DriveoutStatus = apisv1alpha1.Driveout_WaitStartSvc
		if err := ndohandler.UpdateNodeDriveoutStatus(ndoStatus); err != nil {
			logr.Error(err, "DriveoutStage_StartSvc UpdateNodeDriveoutStatus Driveout_WaitStartSvc failed")
			return reconcile.Result{Requeue: true}, nil
		}

	case apisv1alpha1.DriveoutStage_Succeed:
		logr.Debug("DriveoutStage_Succeed ndohandler.NodeDriveoutStage() = %v, ndohandler.NodeDriveoutStatus() = %v", ndohandler.NodeDriveoutStage(), ndohandler.NodeDriveoutStatus())

	case apisv1alpha1.DriveoutStage_Failed:
		ndoStatus.DriveoutStatus = apisv1alpha1.Driveout_Failed
		if err := ndohandler.UpdateNodeDriveoutStatus(ndoStatus); err != nil {
			logr.Error(err, "DriveoutStage_Failed UpdateNodeDriveoutStatus Driveout_Failed failed")
			return reconcile.Result{Requeue: true}, nil
		}

	default:
		logr.Error(err, "Invalid DriveoutStage")
	}

	logr.Debug("Reconcile ReconcileNodeDriveout start ... ")
	r.storageMember.Node().ReconcileNodeDriveout(instance)

	return reconcile.Result{}, nil
}
