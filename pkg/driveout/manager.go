package driveout

import (
	"context"

	apisv1alpha1 "github.com/hwameistor/hwameistor/pkg/apis/hwameistor/local-storage/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NodeDriveoutHandler
type NodeDriveoutHandler struct {
	client.Client
	NodeDriveout apisv1alpha1.NodeDriveout
}

// NewNodeDriveoutHandler
func NewNodeDriveoutHandler(client client.Client) *NodeDriveoutHandler {
	return &NodeDriveoutHandler{
		Client: client,
	}
}

// CreateNodeDriveout
func (ndHandler *NodeDriveoutHandler) CreateNodeDriveout(ndo *apisv1alpha1.NodeDriveout) error {
	log.Debugf("Create NodeDriveout ndo %+v", ndo)

	return ndHandler.Client.Create(context.Background(), ndo)
}

// ConstructNodeDriveout
func (ndHandler *NodeDriveoutHandler) ConstructNodeDriveout(srcNodeName string) (*apisv1alpha1.NodeDriveout, error) {
	log.Debugf("ConstructNodeDriveout srcNodeName = %+v", srcNodeName)

	var ndo = &apisv1alpha1.NodeDriveout{}

	ndo.TypeMeta = metav1.TypeMeta{
		Kind:       NodeDriveoutKind,
		APIVersion: NodeDriveoutAPIVersion,
	}
	ndo.ObjectMeta = metav1.ObjectMeta{
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		Name:        driveoutTaskPrefix + srcNodeName,
	}

	var nodeDriveoutSpec = apisv1alpha1.NodeDriveoutSpec{}
	nodeDriveoutSpec.NodeName = srcNodeName
	//nodeDriveoutSpec.NodeDriveoutStage = apisv1alpha1.DriveoutStage_Init

	ndo.Spec = nodeDriveoutSpec
	return ndo, nil
}

// ListNodeDriveout
func (ndHandler *NodeDriveoutHandler) ListNodeDriveout() (*apisv1alpha1.NodeDriveoutList, error) {
	list := &apisv1alpha1.NodeDriveoutList{
		TypeMeta: metav1.TypeMeta{
			Kind:       NodeDriveoutKind,
			APIVersion: NodeDriveoutAPIVersion,
		},
	}

	err := ndHandler.List(context.TODO(), list)
	return list, err
}

// GetNodeDriveout
func (ndHandler *NodeDriveoutHandler) GetNodeDriveout(key client.ObjectKey) (*apisv1alpha1.NodeDriveout, error) {
	rd := &apisv1alpha1.NodeDriveout{}
	if err := ndHandler.Get(context.Background(), key, rd); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return rd, nil
}

// UpdateNodeDriveoutStatus
func (ndHandler *NodeDriveoutHandler) UpdateNodeDriveoutStatus(status apisv1alpha1.NodeDriveoutStatus) error {
	log.Debugf("NodeDriveoutHandler UpdateNodeDriveoutStatus status = %v", status)

	ndHandler.NodeDriveout.Status.DriveoutStatus = status.DriveoutStatus
	return ndHandler.Status().Update(context.Background(), &ndHandler.NodeDriveout)
}

// Refresh
func (ndHandler *NodeDriveoutHandler) Refresh() (*NodeDriveoutHandler, error) {
	rd, err := ndHandler.GetNodeDriveout(client.ObjectKey{Name: ndHandler.NodeDriveout.GetName()})
	if err != nil {
		return ndHandler, err
	}
	ndHandler.SetNodeDriveout(*rd.DeepCopy())
	return ndHandler, nil
}

// SetNodeDriveout
func (ndHandler *NodeDriveoutHandler) SetNodeDriveout(nd apisv1alpha1.NodeDriveout) *NodeDriveoutHandler {
	ndHandler.NodeDriveout = nd
	return ndHandler
}

// SetMsg
func (ndHandler *NodeDriveoutHandler) SetMsg(msg string) error {
	ndHandler.NodeDriveout.Status.Message = msg
	return ndHandler.Status().Update(context.Background(), &ndHandler.NodeDriveout)
}

// NodeDriveoutStage
func (ndHandler *NodeDriveoutHandler) NodeDriveoutStage() apisv1alpha1.NodeDriveoutStage {
	return ndHandler.NodeDriveout.Spec.NodeDriveoutStage
}

// NodeDriveoutStatus
func (ndHandler *NodeDriveoutHandler) NodeDriveoutStatus() apisv1alpha1.NodeDriveoutStatus {
	return ndHandler.NodeDriveout.Status
}

// SetNodeDriveoutStage
func (ndHandler *NodeDriveoutHandler) SetNodeDriveoutStage(stage apisv1alpha1.NodeDriveoutStage) *NodeDriveoutHandler {
	ndHandler.NodeDriveout.Spec.NodeDriveoutStage = stage
	return ndHandler
}

// SetDriveoutVolumeNames
func (ndHandler *NodeDriveoutHandler) SetDriveoutVolumeNames(volumeNames []string) *NodeDriveoutHandler {
	ndHandler.NodeDriveout.Status.DriveoutVolumeNames = volumeNames
	return ndHandler
}

// UpdateNodeDriveoutCR
func (ndHandler *NodeDriveoutHandler) UpdateNodeDriveoutCR() error {
	return ndHandler.Update(context.Background(), &ndHandler.NodeDriveout)
}

// SetNodeDriveout
func (ndHandler *NodeDriveoutHandler) CheckNodeDriveoutTaskDestroyed(rd apisv1alpha1.NodeDriveout) bool {
	_, err := ndHandler.GetNodeDriveout(client.ObjectKey{Name: rd.Name, Namespace: rd.Namespace})
	if err != nil {
		if errors.IsNotFound(err) {
			return true
		}
		return false
	}
	return false
}
