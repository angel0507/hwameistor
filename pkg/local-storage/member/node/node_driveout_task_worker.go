package node

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	apisv1alpha1 "github.com/hwameistor/hwameistor/pkg/apis/hwameistor/local-storage/v1alpha1"
)

func (m *manager) startNodeDriveoutTaskWorker(stopCh <-chan struct{}) {
	m.logger.Debug("NodeDriveout Worker is working now")
	go func() {
		for {
			task, shutdown := m.nodeDriveoutTaskQueue.Get()
			if shutdown {
				m.logger.WithFields(log.Fields{"task": task}).Debug("Stop the NodeDriveout worker")
				break
			}
			if err := m.processNodeDriveout(task); err != nil {
				m.logger.WithFields(log.Fields{"task": task, "attempts": m.nodeDriveoutTaskQueue.NumRequeues(task), "error": err.Error()}).Error("Failed to process NodeDriveout task, retry later")
				m.nodeDriveoutTaskQueue.AddRateLimited(task)
			} else {
				m.logger.WithFields(log.Fields{"task": task}).Debug("Completed a NodeDriveout task.")
				m.nodeDriveoutTaskQueue.Forget(task)
			}
			m.nodeDriveoutTaskQueue.Done(task)
		}
	}()

	<-stopCh
	m.nodeDriveoutTaskQueue.Shutdown()
}

func (m *manager) processNodeDriveout(ndNamespacedName string) error {
	logCtx := m.logger.WithFields(log.Fields{"NodeDriveout": ndNamespacedName})
	logCtx.Debug("Working on a NodeDriveout task")

	nodeDriveout := &apisv1alpha1.NodeDriveout{}
	if err := m.apiClient.Get(context.TODO(), types.NamespacedName{Name: ndNamespacedName}, nodeDriveout); err != nil {
		if !errors.IsNotFound(err) {
			logCtx.WithError(err).Error("Failed to get NodeDriveout from cache")
			return err
		}
		logCtx.Info("Not found the NodeDriveout from cache, should be deleted already.")
		return nil
	}

	if nodeDriveout.Spec.NodeName != m.name {
		return nil
	}

	m.ndohandler = m.ndohandler.SetNodeDriveout(*nodeDriveout)
	ndohandler, err := m.ndohandler.Refresh()
	if err != nil {
		log.Error(err, "DriveoutStatus Refresh failed")
		return err
	}
	m.ndohandler = ndohandler

	// log with namespace/name is enough
	logCtx = m.logger.WithFields(log.Fields{"nodeDriveout": nodeDriveout.Name, "spec": nodeDriveout.Spec, "status": nodeDriveout.Status})
	logCtx.Debug("Starting to process a NodeDriveout task")
	switch nodeDriveout.Status.DriveoutStatus {
	case apisv1alpha1.Driveout_Init:
		return nil
	case apisv1alpha1.Driveout_WaitStopSvc:
		warnMsg := fmt.Sprintf("Plz Stop All the Hwameistor Svc on the Node %v !", nodeDriveout.Spec.NodeName)
		err := m.ndohandler.SetMsg(warnMsg)
		if err != nil {
			log.Error(err, "DriveoutStatus Driveout_WaitStopSvc SetMsg failed")
			return err
		}
		return nil

	case apisv1alpha1.Driveout_StopingSvc:
		return nil

	case apisv1alpha1.Driveout_WaitDriveOut:
		ndoStatus := m.ndohandler.NodeDriveoutStatus()
		ndoStatus.DriveoutStatus = apisv1alpha1.Driveout_DrivingOut
		if err := m.ndohandler.UpdateNodeDriveoutStatus(ndoStatus); err != nil {
			logCtx.Error(err, "DriveoutStatus UpdateNodeDriveoutStatus from Driveout_WaitDriveOut into Driveout_DrivingOut failed")
			return err
		}
		return nil

	case apisv1alpha1.Driveout_DrivingOut:
		return nil

	case apisv1alpha1.Driveout_WaitStartSvc:
		warnMsg := fmt.Sprintf("Plz Start All the Hwameistor Svc on the Node %v !", nodeDriveout.Spec.NodeName)
		err := m.ndohandler.SetMsg(warnMsg)
		if err != nil {
			log.Error(err, "DriveoutStatus Driveout_WaitStartSvc SetMsg failed")
			return err
		}
		return nil

	case apisv1alpha1.Driveout_StartingSvc:
		return nil

	case apisv1alpha1.Driveout_Succeed:
		return nil

	case apisv1alpha1.Driveout_Failed:
		warnMsg := fmt.Sprintf("No Hwameistor Svc or No Hwameistor Volumes on the Node %v !", nodeDriveout.Spec.NodeName)
		err := m.ndohandler.SetMsg(warnMsg)
		if err != nil {
			log.Error(err, "DriveoutStatus Driveout_Failed SetMsg failed")
			return err
		}
		return nil
	default:
		logCtx.Error("Invalid state/phase")
	}
	return fmt.Errorf("invalid state/phase")
}
