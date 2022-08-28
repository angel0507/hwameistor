package driveout

import (
	"context"
	"fmt"
	apisv1alpha1 "github.com/hwameistor/hwameistor/pkg/apis/hwameistor/local-storage/v1alpha1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	runtimecache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"
	"sync"
	"time"

	"github.com/hwameistor/hwameistor/pkg/local-storage/common"
	mykube "github.com/hwameistor/hwameistor/pkg/utils/kubernetes"
	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	informerstoragev1 "k8s.io/client-go/informers/storage/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	hwameiStorSuffix          = "hwameistor.io"
	podScheduledNodeNameIndex = "assignedNode"
	AnnotationKeyNodeDriveOut = "lvm.hwameistor.io/driveout"
	driveoutTaskPrefix        = "node-driveout-"
)

// Assistant interface
type Assistant interface {
	Run(stopCh <-chan struct{}) error
}

type driveoutAssistant struct {
	clientset *kubernetes.Clientset

	apiClient client.Client

	Recorder record.EventRecorder

	nodeCoolDownDuration     time.Duration
	nodeInformer             informercorev1.NodeInformer
	podInformer              informercorev1.PodInformer
	pvInformer               informercorev1.PersistentVolumeInformer
	volumeAttachmentInformer informerstoragev1.VolumeAttachmentInformer

	//localVolumeInformer informerlocalstorage.LocalVolumeInformer

	driveoutQueue *common.TaskQueue

	driveoutNodesCache common.Cache

	informersCache runtimecache.Cache

	migrateCtr Controller

	ndohandler *NodeDriveoutHandler

	logger *log.Entry
}

// New an assistant instance
func New(driveoutCoolDownDuration time.Duration, mgr manager.Manager) Assistant {
	return &driveoutAssistant{
		driveoutQueue:        common.NewTaskQueue("DriveoutTask", 0),
		driveoutNodesCache:   common.NewCache(),
		informersCache:       mgr.GetCache(),
		nodeCoolDownDuration: driveoutCoolDownDuration,
		migrateCtr:           NewController(mgr),
		apiClient:            mgr.GetClient(),
		Recorder:             mgr.GetEventRecorderFor("DriveoutTask"),
		ndohandler:           NewNodeDriveoutHandler(mgr.GetClient()),
		logger:               log.WithField("Module", "Driveout"),
	}
}

func (fa *driveoutAssistant) Run(stopCh <-chan struct{}) error {
	clientset, err := mykube.NewClientSet()
	if err != nil {
		log.WithError(err).Error("Failed to build cluster clientset")
		return err
	}
	fa.clientset = clientset
	factory := informers.NewSharedInformerFactory(clientset, 0)

	fa.nodeInformer = factory.Core().V1().Nodes()
	fa.nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    fa.onNodeAdd,
		UpdateFunc: fa.onNodeUpdate,
	})

	// index: pod.spec.nodename
	podScheduledNodeNameIndexFunc := func(obj interface{}) ([]string, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok || pod == nil {
			return []string{}, fmt.Errorf("wrong Pod resource")
		}
		return []string{pod.Spec.NodeName}, nil
	}
	fa.podInformer = factory.Core().V1().Pods()
	fa.podInformer.Informer().AddIndexers(cache.Indexers{podScheduledNodeNameIndex: podScheduledNodeNameIndexFunc})

	log.Debug("start informer factory")
	factory.Start(stopCh)
	for _, v := range factory.WaitForCacheSync(stopCh) {
		if !v {
			log.Error("Timed out waiting for cache to sync")
			return fmt.Errorf("timed out waiting for cache to sync")
		}
	}

	log.Debug("start driveout worker")
	go fa.startDriveoutWorker(stopCh)

	return nil
}

func (fa *driveoutAssistant) handleVolumeCRDUpdatedEvent(oldObj, newObj interface{}) {
	instance, _ := oldObj.(*apisv1alpha1.LocalVolume)
	fa.logger.WithFields(log.Fields{"volume": instance.Name, "spec": instance.Spec, "status": instance.Status}).Info("Observed a Volume CRD deletion...")
}

func (fa *driveoutAssistant) onNodeAdd(obj interface{}) {
	node, _ := obj.(*corev1.Node)
	fa.checkNodeForDriveout(node)
}

func (fa *driveoutAssistant) onNodeUpdate(oldObj, newObj interface{}) {
	node, _ := newObj.(*corev1.Node)
	fa.checkNodeForDriveout(node)
}

func (fa *driveoutAssistant) checkNodeForDriveout(node *corev1.Node) {
	fa.logger.Debug("checkNodeForDriveout start node.Name = %v", node.Name)
	if fa.shouldDriveoutForNode(node) {
		if _, expired := fa.driveoutNodesCache.Get(node.Name); expired {
			log.WithField("node", node.Name).Debug("Add node into driveout queue")
			fa.driveoutQueue.Add(node.Name)
			fa.driveoutNodesCache.Set(node.Name, node, fa.nodeCoolDownDuration)
		}
	} else {
		fa.driveoutQueue.Forget(node.Name)
		fa.driveoutQueue.Done(node.Name)
	}
}

func (fa *driveoutAssistant) createNodeDriveoutForNode(nodeName string) (*apisv1alpha1.NodeDriveout, error) {
	tmpNodeDriveout := &apisv1alpha1.NodeDriveout{}
	driveoutName := driveoutTaskPrefix + nodeName
	err := fa.apiClient.Get(context.TODO(), types.NamespacedName{Name: driveoutName}, tmpNodeDriveout)
	tmpNodeDriveoutStage := tmpNodeDriveout.Spec.NodeDriveoutStage
	fa.logger.Debug("createNodeDriveoutForNode tmpNodeDriveout = %v", tmpNodeDriveout)

	if err == nil {
		if tmpNodeDriveoutStage != apisv1alpha1.DriveoutStage_Failed && tmpNodeDriveoutStage != apisv1alpha1.DriveoutStage_Succeed {
			return tmpNodeDriveout, nil
		}
		if delErr := fa.apiClient.Delete(context.TODO(), tmpNodeDriveout); delErr != nil {
			fa.logger.WithError(err).Error("Failed to delete NodeDriveout")
			return tmpNodeDriveout, delErr
		}
	} else {
		fa.logger.WithError(err).Error("Failed to get NodeDriveout")
	}
	ndo, err := fa.ndohandler.ConstructNodeDriveout(nodeName)
	if err != nil {
		fa.logger.WithError(err).Error("Failed to ConstructNodeDriveout")
		return ndo, err
	}
	err = fa.ndohandler.CreateNodeDriveout(ndo)
	if err != nil {
		fa.logger.WithError(err).Error("Failed to CreateNodeDriveout")
		return ndo, err
	}
	return ndo, nil
}

func (fa *driveoutAssistant) shouldDriveoutForNode(node *corev1.Node) bool {
	fa.logger.Debug("shouldDriveoutForNode start node.Name = %v", node.Name)
	if node == nil {
		return false
	}
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			if cond.Status == corev1.ConditionUnknown {
				return true
			}
			break
		}
	}
	_, exists := node.Annotations[AnnotationKeyNodeDriveOut]
	if !exists {
		fa.logger.Error("shouldDriveoutForNode AnnotationKeyNodeDriveOut not exists = %v, nodeName = %v", exists, node.Name)
		return false
	}
	fa.logger.Debug("shouldDriveoutForNode AnnotationKeyNodeDriveOut exists = %v, nodeName = %v", exists, node.Name)
	return true
}

func (fa *driveoutAssistant) startDriveoutWorker(stopCh <-chan struct{}) {
	log.Debug("startDriveoutWorker is working now")
	go func() {
		for {
			time.Sleep(15 * time.Second)
			task, shutdown := fa.driveoutQueue.Get()
			if shutdown {
				log.WithFields(log.Fields{"task": task}).Debug("Stop the Driveout worker")
				break
			}
			if err := fa.driveoutForNode(task); err != nil {
				log.WithFields(log.Fields{"task": task, "error": err.Error()}).Error("Failed to process Node Driveout task, retry later")
				fa.driveoutQueue.AddRateLimited(task)
			} else {
				log.WithFields(log.Fields{"task": task}).Debug("Completed a Node Driveout task.")
				fa.driveoutQueue.Forget(task)
			}
			fa.driveoutQueue.Done(task)
		}
	}()

	<-stopCh
	fa.driveoutQueue.Shutdown()
}

func (fa *driveoutAssistant) listHwameiStorPodsForNode(nodeName string) ([]*corev1.Pod, error) {
	pods, err := fa.podInformer.Informer().GetIndexer().ByIndex(podScheduledNodeNameIndex, nodeName)
	if err != nil {
		fa.logger.WithError(err).Error("listPodsForNode: Failed to get pods on the node")
		return nil, err
	}
	var hwameiPods []*corev1.Pod
	for i := range pods {
		pod, _ := pods[i].(*corev1.Pod)
		if !fa.isHwameiStorPod(pod) {
			continue
		}
		fa.logger.Debug("listHwameiStorPodsForNode pod.Name = %v", pod.Name)
		hwameiPods = append(hwameiPods, pod)
	}
	return hwameiPods, nil
}

func (fa *driveoutAssistant) driveoutForNode(nodeName string) error {
	fa.logger.Debug("Start to driveoutForNode")
	// step1: createNodeDriveout CR
	ndo, err := fa.createNodeDriveoutForNode(nodeName)
	if err != nil {
		fa.logger.WithError(err).Error("Failed to createNodeDriveoutForNode")
		return err
	}
	fa.ndohandler = fa.ndohandler.SetNodeDriveout(*ndo)

	// Step2: check driveoutnode event canceled or not
	node, err := fa.nodeInformer.Lister().Get(nodeName)
	if err != nil {
		fa.logger.WithError(err).Error("Failed to get node info")
		return err
	}
	if !fa.shouldDriveoutForNode(node) {
		// double check in case that nodes'status is changed back to normal shortly
		fa.logger.Info("Cancel driveout for node")
		return nil
	}

	fa.ndohandler, err = fa.ndohandler.Refresh()
	if err != nil {
		fa.logger.Error(err, "driveoutForNode Refresh failed")
		return err
	}

	pods, err := fa.listHwameiStorPodsForNode(nodeName)
	if err != nil {
		fa.logger.WithError(err).Error("Failed to listHwameiStorPodsForNode on the node")
		return err
	}
	var migrateLocalVolumes []string
	for _, pod := range pods {
		lvs := fa.getAssociatedVolumes(pod)
		for _, lv := range lvs {
			migrateLocalVolumes = append(migrateLocalVolumes, lv.Name)
			fa.logger.Debug("getAssociatedVolumes lv.Name = %v", lv.Name)
			break
		}
	}

	fa.logger.Debug("driveoutForNode migrateLocalVolumes = %v", migrateLocalVolumes)
	fa.logger.Debug("driveoutForNode ndohandler.NodeDriveoutStatus().DriveoutVolumeNames 1 = %v", fa.ndohandler.NodeDriveoutStatus().DriveoutVolumeNames)

	driveoutStage := fa.ndohandler.NodeDriveoutStage()
	fa.logger.Debug("driveoutForNode driveoutStage = %v", driveoutStage)

	switch driveoutStage {
	case "":
		fa.ndohandler, err = fa.ndohandler.Refresh()
		if err != nil {
			fa.logger.Error(err, "driveoutForNode Refresh failed")
			return err
		}
		if len(migrateLocalVolumes) != 0 {
			fa.ndohandler = fa.ndohandler.SetDriveoutVolumeNames(migrateLocalVolumes)
			fa.ndohandler.UpdateNodeDriveoutStatus(fa.ndohandler.NodeDriveout.Status)
			if err != nil {
				fa.logger.Error(err, "driveoutForNode UpdateNodeDriveoutStatus failed")
				return err
			}
		}

		return errors.NewBadRequest("driveoutStage is empty, waiting ... ")
	case apisv1alpha1.DriveoutStage_Init:
		fa.logger.Info("driveoutForNode len(migrateLocalVolumes) = %v, err = %v", len(migrateLocalVolumes), err)
		// Step3: caculate all hwameistor volumes which should be drivedout
		fa.ndohandler, err = fa.ndohandler.Refresh()
		if err != nil {
			fa.logger.Error(err, "DriveoutStage_Init driveoutForNode Refresh failed")
			return err
		}
		tobemigratedLocalVolumes := fa.ndohandler.NodeDriveout.Status.DriveoutVolumeNames
		if len(tobemigratedLocalVolumes) == 0 {
			fa.ndohandler = fa.ndohandler.SetNodeDriveoutStage(apisv1alpha1.DriveoutStage_Failed)
			err = fa.ndohandler.UpdateNodeDriveoutCR()
			if err != nil {
				fa.logger.Error(err, "driveoutForNode UpdateNodeDriveoutCR DriveoutStage_Failed failed")
				return err
			}
			return nil
		} else {
			// Step4: update nodeDriveout CR stage DriveoutStage_Init --> DriveoutStage_StopSvc
			fa.ndohandler = fa.ndohandler.SetNodeDriveoutStage(apisv1alpha1.DriveoutStage_StopSvc)
			err = fa.ndohandler.UpdateNodeDriveoutCR()
			if err != nil {
				fa.logger.Error(err, "driveoutForNode UpdateNodeDriveoutCR DriveoutStage_StopSvc failed")
				return err
			}
		}
		fallthrough

	case apisv1alpha1.DriveoutStage_StopSvc:
		// Step5: waitHwameistorPodStopForNode
		fa.ndohandler, err = fa.ndohandler.Refresh()
		if err != nil {
			fa.logger.Error(err, "DriveoutStage_StopSvc driveoutForNode Refresh failed")
			return err
		}
		err = fa.waitHwameistorPodStopForNode(nodeName)
		if err != nil {
			fa.logger.WithError(err).Error("Failed to waitHwameistorPodStop")
			return err
		}
		// Step6: update nodeDriveout CR stage DriveoutStage_StopSvc --> DriveoutStage_DrivingOut
		fa.ndohandler = fa.ndohandler.SetNodeDriveoutStage(apisv1alpha1.DriveoutStage_DrivingOut)
		err = fa.ndohandler.UpdateNodeDriveoutCR()
		if err != nil {
			fa.logger.Error(err, "driveoutForNode UpdateNodeDriveoutCR DriveoutStage_DrivingOut failed")
			return err
		}
		fallthrough

	case apisv1alpha1.DriveoutStage_DrivingOut:
		// Step7: start node hwameistor volumes driveout
		fa.ndohandler, err = fa.ndohandler.Refresh()
		if err != nil {
			fa.logger.Error(err, "DriveoutStage_DrivingOut driveoutForNode Refresh failed")
			return err
		}
		var localVolumeList []apisv1alpha1.LocalVolume
		var tobemigratedLocalVolumes = fa.ndohandler.NodeDriveout.Status.DriveoutVolumeNames
		for _, localVolumeName := range tobemigratedLocalVolumes {
			vol := &apisv1alpha1.LocalVolume{}
			if err := fa.apiClient.Get(context.TODO(), types.NamespacedName{Name: localVolumeName}, vol); err != nil {
				if !errors.IsNotFound(err) {
					log.Errorf("Failed to get Volume from cache, retry it later ...")
					return err
				}
				log.Printf("Not found the Volume from cache, should be deleted already.")
				return err
			}

			err := fa.createMigrateTaskByLocalVolume(*vol, nodeName)
			if err != nil {
				log.Error(err, "driveoutForNode: createMigrateTaskByLocalVolume failed")
				return err
			}
			localVolumeList = append(localVolumeList, *vol)
		}

		err = fa.waitMigrateTaskByLocalVolumeDone(localVolumeList)
		if err != nil {
			fa.logger.WithError(err).Error("driveoutForNode: Failed to waitMigrateTaskByLocalVolumeDone")
			return err
		}
		// Step8: update nodeDriveout CR statge DriveoutStage_DrivingOut --> DriveoutStage_StartSvc
		fa.ndohandler = fa.ndohandler.SetNodeDriveoutStage(apisv1alpha1.DriveoutStage_StartSvc)
		err = fa.ndohandler.UpdateNodeDriveoutCR()
		if err != nil {
			fa.logger.Error(err, "driveoutForNode SetNodeDriveoutStage DriveoutStage_StartSvc failed")
			return err
		}
		fallthrough

	case apisv1alpha1.DriveoutStage_StartSvc:
	}

	fa.logger.Debug("driveoutForNode end ... ")

	return nil
}

func (fa *driveoutAssistant) getReplicasForVolume(volName string) ([]*apisv1alpha1.LocalVolumeReplica, error) {
	// todo
	replicaList := &apisv1alpha1.LocalVolumeReplicaList{}
	if err := fa.apiClient.List(context.TODO(), replicaList); err != nil {
		return nil, err
	}

	var replicas []*apisv1alpha1.LocalVolumeReplica
	for i := range replicaList.Items {
		if replicaList.Items[i].Spec.VolumeName == volName {
			replicas = append(replicas, &replicaList.Items[i])
		}
	}
	return replicas, nil
}

func (fa *driveoutAssistant) isHwameiStorVolume(ns, pvc string) (bool, error) {
	logCtx := log.Fields{"NameSpace": ns, "PVC": pvc}

	// Step 1: get storageclass by pvc
	sc, err := fa.getStorageClassByPVC(ns, pvc)
	if err != nil {
		fa.logger.WithFields(logCtx).WithError(err).Error("failed to get storageclass")
		return false, err
	}

	// skip static volume
	if sc == "" {
		return false, nil
	}

	// Step 2: compare provisioner name with hwameistor.io
	provisioner, err := fa.getProvisionerByStorageClass(sc)
	if err != nil {
		fa.logger.WithFields(logCtx).WithError(err).Error("failed to get provisioner")
		return false, err
	}
	return strings.HasSuffix(provisioner, hwameiStorSuffix), nil
}

// getStorageClassByPVC return sc name if set, else return empty if it is a static volume
func (fa *driveoutAssistant) getStorageClassByPVC(ns, pvcName string) (string, error) {
	pvc, err := fa.clientset.CoreV1().PersistentVolumeClaims(ns).Get(context.Background(), pvcName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	if pvc.Spec.StorageClassName == nil {
		return "", nil
	}
	return *pvc.Spec.StorageClassName, nil
}

func (fa *driveoutAssistant) getProvisionerByStorageClass(scName string) (string, error) {
	sc, err := fa.clientset.StorageV1().StorageClasses().Get(context.Background(), scName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	return sc.Provisioner, nil
}

func (fa *driveoutAssistant) isHwameiStorPod(pod *corev1.Pod) bool {
	for _, vol := range pod.Spec.Volumes {
		if vol.PersistentVolumeClaim == nil {
			continue
		}
		pvc := &corev1.PersistentVolumeClaim{}
		pvc, err := fa.clientset.CoreV1().PersistentVolumeClaims(pod.Namespace).Get(context.Background(), vol.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
		if err != nil {
			fa.logger.WithFields(log.Fields{"namespace": pod.Namespace, "pvc": vol.PersistentVolumeClaim.ClaimName}).WithError(err).Error("Failed to fetch PVC")
			continue
		}
		if fa.isHwameiStorPVC(pvc) {
			return true
		}
	}
	return false
}

func (fa *driveoutAssistant) isHwameiStorPVC(pvc *corev1.PersistentVolumeClaim) bool {
	if pvc.Spec.StorageClassName == nil {
		return false
	}
	sc := &storagev1.StorageClass{}
	sc, err := fa.clientset.StorageV1().StorageClasses().Get(context.Background(), *pvc.Spec.StorageClassName, metav1.GetOptions{})
	if err != nil {
		fa.logger.WithFields(log.Fields{"namespace": pvc.Namespace, "pvc": pvc.Name, "storageclass": *pvc.Spec.StorageClassName}).WithError(err).Error("Failed to fetch storageclass")
		return false
	}
	return sc.Provisioner == apisv1alpha1.CSIDriverName
}

func (fa *driveoutAssistant) getAssociatedVolumes(pod *corev1.Pod) []*apisv1alpha1.LocalVolume {
	lvs := []*apisv1alpha1.LocalVolume{}
	for _, vol := range pod.Spec.Volumes {
		if vol.PersistentVolumeClaim == nil {
			continue
		}
		pvc := &corev1.PersistentVolumeClaim{}
		pvc, err := fa.clientset.CoreV1().PersistentVolumeClaims(pod.Namespace).Get(context.Background(), vol.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
		if err != nil {
			fa.logger.WithFields(log.Fields{"namespace": pod.Namespace, "pvc": vol.PersistentVolumeClaim.ClaimName}).WithError(err).Error("Failed to fetch PVC")
			continue
		}
		lv, err := fa.constructLocalVolumeForMigrate(pvc)
		if err == nil {
			lvs = append(lvs, lv)
		}
	}
	log.Debug("getAssociatedVolumes return ")
	return lvs
}

func (fa *driveoutAssistant) constructLocalVolumeForMigrate(pvc *corev1.PersistentVolumeClaim) (*apisv1alpha1.LocalVolume, error) {
	lv := apisv1alpha1.LocalVolume{}

	if err := fa.apiClient.Get(context.TODO(), types.NamespacedName{Name: pvc.Spec.VolumeName}, &lv); err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed to get Volume from cache, retry it later ...")
			return &lv, err
		}
		log.Printf("Not found the Volume from cache, should be deleted already.")
		return &lv, err
	}

	lv.Name = pvc.Spec.VolumeName
	lv.Namespace = pvc.Namespace

	return &lv, nil
}

func (fa *driveoutAssistant) createMigrateTaskByLocalVolume(vol apisv1alpha1.LocalVolume, sourceNodeName string) error {
	fa.logger.Debug("createMigrateTaskByLocalVolume start ... ")

	localVolumeMigrate, err := fa.migrateCtr.ConstructLocalVolumeMigrate(vol, sourceNodeName)
	if err != nil {
		log.Error(err, "createMigrateTaskByLocalVolume ConstructLocalVolumeMigrate failed")
		return err
	}
	migrate := &apisv1alpha1.LocalVolumeMigrate{}
	if err := fa.apiClient.Get(context.TODO(), types.NamespacedName{Namespace: localVolumeMigrate.Namespace, Name: localVolumeMigrate.Name}, migrate); err == nil {
		fa.logger.Info("createMigrateTaskByLocalVolume localVolumeMigrate %v already exists.", localVolumeMigrate.Name)
		return nil
	}

	err = fa.migrateCtr.CreateLocalVolumeMigrate(*localVolumeMigrate)
	if err != nil {
		log.Error(err, "createMigrateTaskByLocalVolume CreateLocalVolumeMigrate failed")
		return err
	}

	fa.logger.Debug("createMigrateTaskByLocalVolume end ... ")
	return nil
}

func (fa *driveoutAssistant) waitHwameistorPodStopForNode(nodeName string) error {
	fa.logger.Debug("waitHwameistorPodStopForNode start ... ")
	for {
		pods, err := fa.listHwameiStorPodsForNode(nodeName)
		fa.logger.Debug("waitHwameistorPodStopForNode listHwameiStorPodsForNode len(pods) = %v, err = %v", len(pods), err)
		if err != nil {
			fa.logger.WithError(err).Error("Failed to listHwameiStorPodsForNode on the node")
			return err
		}
		var hwameistorlvs []*apisv1alpha1.LocalVolume
		for _, pod := range pods {
			lvs := fa.getAssociatedVolumes(pod)
			hwameistorlvs = append(hwameistorlvs, lvs...)
		}
		if len(hwameistorlvs) == 0 {
			break
		}
	}
	return nil
}

func (fa *driveoutAssistant) waitHwameistorSvcPodStop(pod *corev1.Pod) error {
	fa.logger.Debug("waitHwameistorSvcPodStop start ... ")

	var globalErr error = errors.NewBadRequest("waitHwameistorSvcPodStop: you should stop the pod svc now !")
	var toBeStoppingPod = &corev1.Pod{}
	var err error

	for {
		toBeStoppingPod, err = fa.clientset.CoreV1().Pods(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
		fa.logger.Debug("waitHwameistorSvcPodStop toBeStoppingPod.Name = %v, err = %v", toBeStoppingPod.Name, err)

		if err != nil {
			if !errors.IsNotFound(err) {
				fa.logger.WithError(err).Error("waitHwameistorSvcPodStop: Failed to query pod")
				return err
			} else {
				fa.logger.Info("waitHwameistorSvcPodStop: Not found the pod")
				break
			}
		}
		fa.logger.WithFields(log.Fields{"namespace": toBeStoppingPod.Namespace, "name": toBeStoppingPod.Name}).Warning(globalErr)

		time.Sleep(3 * time.Second)
	}

	return nil
}

// 统计数据迁移情况
func (fa *driveoutAssistant) waitMigrateTaskByLocalVolumeDone(volList []apisv1alpha1.LocalVolume) error {
	fa.logger.Debug("waitMigrateTaskByLocalVolumeDone start ... ")
	var wg sync.WaitGroup
	for _, vol := range volList {
		vol := vol
		wg.Add(1)
		go func() {
			for {
				migrateStatus, err := fa.migrateCtr.GetLocalVolumeMigrateStatusByLocalVolume(vol)
				if err != nil {
					log.Error(err, "waitMigrateTaskByLocalVolumeDone GetLocalVolumeMigrateStatusByLocalVolume failed")
					return
				}
				if migrateStatus.State == apisv1alpha1.OperationStateSubmitted || migrateStatus.State == apisv1alpha1.OperationStateInProgress {
					time.Sleep(2 * time.Second)
					continue
				}
				if migrateStatus.State == apisv1alpha1.OperationStateAborted || migrateStatus.State == apisv1alpha1.OperationStateAborting ||
					migrateStatus.State == apisv1alpha1.OperationStateToBeAborted {
					continue
				}
				break
			}
			wg.Done()
		}()
	}
	wg.Wait()

	fa.logger.Debug("waitMigrateTaskByLocalVolumeDone end ... ")
	return nil
}

func namespacedName(ns string, name string) string {
	return fmt.Sprintf("%s_%s", ns, name)
}
