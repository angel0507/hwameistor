package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeDriveoutSpec defines the desired state of NodeDriveout
type NodeDriveoutSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// *** custom section of the operations ***

	// NodeName nodeName of the driveout
	NodeName string `json:"nodeName,omitempty"`

	// Init StopSvc DriveOut StartSvc Completed
	NodeDriveoutStage NodeDriveoutStage `json:"nodeDriveoutStage,omitempty"`
}

// NodeDriveoutStatus defines the observed state of NodeDriveout
type NodeDriveoutStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// State of the operation, e.g. submitted, started, completed, abort, ...
	DriveoutStatus DriveoutStatus `json:"driveoutStatus,omitempty"`
	// error message to describe some status
	Message string `json:"message,omitempty"`
	// DriveoutVolumeNames driveoutVolumeNames of the node driveout
	DriveoutVolumeNames []string `json:"driveoutVolumeNames,omitempty"`
}

// DriveoutStatus defines the observed status of driveout
type DriveoutStatus string

const (
	// Driveout_Init represents that the hwameistor pods which should be drivedout at present locates Init status.
	Driveout_Init DriveoutStatus = "Init"

	// Driveout_WaitStopSvc represents that hwameistor pods which should be drivedout locates WaitStopSvc status.
	Driveout_WaitStopSvc DriveoutStatus = "WaitStopSvc"

	// Driveout_StopingSvc represents that hwameistor pods which should be drivedout at present locates StopingSvc status.
	Driveout_StopingSvc DriveoutStatus = "StopingSvc"

	// Driveout_WaitDriveOut represents that hwameistor pods volumes which should be drivedout locates WaitDriveOut status.
	Driveout_WaitDriveOut DriveoutStatus = "WaitDriveOut"

	// Driveout_DrivingOut represents that hwameistor pods volumes which should be drivedout locates DrivingOut status.
	Driveout_DrivingOut DriveoutStatus = "DrivingOut"

	// Driveout_WaitStartSvc represents that hwameistor pods which volumes has been drivedout locates WaitStartSvc status.
	Driveout_WaitStartSvc DriveoutStatus = "WaitStartSvc"

	// Driveout_StartingSvc represents that hwameistor pods which volumes has been drivedout locates StartingSvc status.
	Driveout_StartingSvc DriveoutStatus = "StartingSvc"

	// Driveout_Succeed represents that hwameistor pods  which volumes has been drivedout locates Succeed status.
	Driveout_Succeed DriveoutStatus = "Succeed"

	// Driveout_Failed represents that hwameistor pods  which volumes has been drivedout locates Failed status.
	Driveout_Failed DriveoutStatus = "Failed"
)

// DriveoutStage defines the observed state of Driveout
type NodeDriveoutStage string

const (
	// DriveoutStage_Init represents that the node driveout at present locates Init stage.
	DriveoutStage_Init NodeDriveoutStage = "Init"

	// DriveoutStage_StopSvc represents that the node driveout at present locates StopSvc stage.
	DriveoutStage_StopSvc NodeDriveoutStage = "StopSvc"

	// DriveoutStage_DrivingOut represents that the node driveout at present locates DrivingOut stage.
	DriveoutStage_DrivingOut NodeDriveoutStage = "DrivingOut"

	// DriveoutStage_StartSvc represents that the node driveout at present locates StartSvc stage.
	DriveoutStage_StartSvc NodeDriveoutStage = "StartSvc"

	// DriveoutStage_Succeed represents that the node driveout at present locates Succeed stage.
	DriveoutStage_Succeed NodeDriveoutStage = "Succeed"

	// DriveoutStage_Failed represents that the node driveout at present locates Failed stage.
	DriveoutStage_Failed NodeDriveoutStage = "Failed"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeDriveout is the Schema for the NodeDriveouts API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=NodeDriveouts,scope=Cluster,shortName=lvmigrate
// +kubebuilder:printcolumn:name="volume",type=string,JSONPath=`.spec.volumeName`,description="Name of the volume to be migrated"
// +kubebuilder:printcolumn:name="node",type=string,JSONPath=`.spec.nodeName`,description="Node name of the volume replica to be migrated"
// +kubebuilder:printcolumn:name="target",type=string,JSONPath=`.status.targetNodeName`,description="Node name of the new volume replica"
// +kubebuilder:printcolumn:name="state",type=string,JSONPath=`.status.state`,description="State of the migration"
// +kubebuilder:printcolumn:name="age",type=date,JSONPath=`.metadata.creationTimestamp`
type NodeDriveout struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeDriveoutSpec   `json:"spec,omitempty"`
	Status NodeDriveoutStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeDriveoutList contains a list of NodeDriveout
type NodeDriveoutList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeDriveout `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeDriveout{}, &NodeDriveoutList{})
}
