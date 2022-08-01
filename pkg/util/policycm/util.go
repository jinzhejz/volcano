package policycm

import (
	"context"
	"fmt"
	"os"

	v1 "k8s.io/api/core/v1"

	"k8s.io/klog/v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"volcano.sh/volcano/pkg/kube"
)

type schedulerConfigMonitor struct {
	isDefaultScheduler bool
	configMapInformer  cache.SharedIndexInformer
}

const (
	KubeScheduler    string = "kube-scheduler"
	VolcanoScheduler string = "volcano"

	policyConfigNamespace string = "kube-system"
	policyConfigName      string = "scheduler-config"

	defaultSchedulerKey = "default-scheduler"
)

func NewSchedulerCMMonitor(options kube.ClientOptions) *schedulerConfigMonitor {
	config, err := kube.BuildConfig(options)
	if err != nil {
		panic(fmt.Sprintf("failed build kube config, with err: %v", err))
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(fmt.Sprintf("failed init kubeClient, with err: %v", err))
	}

	isDefaultScheduler := false
	cm, err := kubeClient.CoreV1().ConfigMaps(policyConfigNamespace).Get(context.TODO(), policyConfigName, metav1.GetOptions{})
	if err != nil {
		klog.V(4).Infof("Failed to get configmap kube-system/scheduler-config, use kube-scheduler as default scheduler")
	} else {
		if cm.Data[defaultSchedulerKey] == KubeScheduler {
			isDefaultScheduler = false
		} else if cm.Data[defaultSchedulerKey] == VolcanoScheduler {
			isDefaultScheduler = true
		}
	}

	informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)
	configMapInformerFactory := informerFactory.Core().V1().ConfigMaps().Informer()
	monitor := &schedulerConfigMonitor{
		isDefaultScheduler: isDefaultScheduler,
		configMapInformer:  configMapInformerFactory,
	}

	return monitor
}

func (m *schedulerConfigMonitor) Run(stopCh <-chan struct{}) {
	m.configMapInformer.AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch v := obj.(type) {
				case *v1.ConfigMap:
					return v.Namespace == policyConfigNamespace && v.Name == policyConfigName
				default:
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc:    m.addPolicyConfigMap,
				UpdateFunc: m.updatePolicyConfigMap,
				DeleteFunc: m.deletePolicyConfigMap,
			},
		},
	)
	go m.configMapInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, m.configMapInformer.HasSynced)
	klog.V(2).Infof("scheduler config monitor is running")
}

func (m *schedulerConfigMonitor) addPolicyConfigMap(obj interface{}) {
	klog.V(2).Infof("add scheduler policy configmap")
	cm, ok := obj.(*v1.ConfigMap)
	if !ok {
		klog.Errorf("cannot convert policy configmap to *v1.ConfigMap: %v", obj)
		return
	}

	if !m.isValidData(cm) {
		klog.V(2).Infof("default-scheduler %s in policy config map is invalid, should kube-scheduler or volcano", cm.Data[defaultSchedulerKey])
		return
	}

	if !m.defaultSchedulerIsChanged(cm) {
		klog.V(2).Infof("scheduler config is not changed")
		return
	}

	m.killScheduler()
}

func (m *schedulerConfigMonitor) updatePolicyConfigMap(_, new interface{}) {
	klog.V(2).Infof("update scheduler policy configmap")
	cm, ok := new.(*v1.ConfigMap)
	if !ok {
		klog.Errorf("cannot convert policy configmap to *v1.ConfigMap: %v", new)
		return
	}

	if !m.isValidData(cm) {
		klog.V(2).Infof("default-scheduler %s in policy config map is invalid, should kube-scheduler or volcano", cm.Data[defaultSchedulerKey])
		return
	}

	if !m.defaultSchedulerIsChanged(cm) {
		klog.V(2).Infof("scheduler config is not changed")
		return
	}

	m.killScheduler()
}

func (m *schedulerConfigMonitor) deletePolicyConfigMap(obj interface{}) {
	klog.V(2).Infof("delete scheduler policy configmap")
	_, ok := obj.(*v1.ConfigMap)
	if !ok {
		klog.Errorf("cannot convert policy configmap to *v1.ConfigMap: %v", obj)
		return
	}

	if !m.isDefaultScheduler {
		return
	}

	m.killScheduler()
}

func (m *schedulerConfigMonitor) IsDefaultScheduler() bool {
	return m.isDefaultScheduler
}

func (m *schedulerConfigMonitor) killScheduler() {
	klog.V(1).Infof("scheduler is going to restart in order to update policy")
	klog.Flush()
	os.Exit(0)
}

func (m *schedulerConfigMonitor) isValidData(cm *v1.ConfigMap) bool {
	return cm.Data[defaultSchedulerKey] == KubeScheduler || cm.Data[defaultSchedulerKey] == VolcanoScheduler
}

func (m *schedulerConfigMonitor) defaultSchedulerIsChanged(cm *v1.ConfigMap) bool {
	return (m.isDefaultScheduler && cm.Data[defaultSchedulerKey] == KubeScheduler) || (!m.isDefaultScheduler && cm.Data[defaultSchedulerKey] == VolcanoScheduler)
}
