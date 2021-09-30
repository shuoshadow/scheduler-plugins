package podcountbynode

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	pluginConfig "sigs.k8s.io/scheduler-plugins/pkg/apis/config"
)

var _ framework.FilterPlugin = &PodCountByNode{}

const (
	// Name : name of plugin
	Name = "PodCountByNode"
)

// PodCountByNode : scheduler plugin
type PodCountByNode struct {
	handle    framework.Handle
	args      *pluginConfig.PodCountByNodeArgs
}

func New(obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	klog.V(4).Infof("Creating new instance of the PodCountByNode plugin")
	args := getArgs(obj)

	pcn := &PodCountByNode{
		handle:    handle,
		args:      args,
	}
	return pcn, nil
}

func (pcn *PodCountByNode) Filter(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, node *framework.NodeInfo) *framework.Status {
	klog.Infof("PodCountByNode filter, pod: %v, node: %v\n", pod.Name, node.Node().Name)
	// get labels
	nodeLabels := node.Node().Labels
	for _, podLimit := range pcn.args.PodCountLimit {
		selector := podLimit.NodeLabels
		// node 匹配
		if mapContainer(nodeLabels, selector) {
			klog.Infof("PodCountByNode match node: %s", node.Node().Name)
			// 查询pod数量
			namespace := podLimit.PodNamespace
			podCount := 0
			for _, podInfo := range node.Pods {
				if podInfo.Pod.Namespace == namespace {
					podCount++
				}
			}
			if podCount >= podLimit.Count {
				klog.Infof("PodCountByNode filter node:%s, pod counts:%d", node.Node().Name, podCount)
				return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("on node:%s, pod in namespace:%s >= %d", node.Node().Name, namespace, podLimit.Count))
			}
		} else {
			klog.Infof("PodCountByNode not match, nodeLabels:%v, selector:%v", nodeLabels, selector)
		}
	}

	return framework.NewStatus(framework.Success)
}

// Name : name of plugin
func (pcn *PodCountByNode) Name() string {
	return Name
}

// getArgs : get configured args
func getArgs(obj runtime.Object) *pluginConfig.PodCountByNodeArgs {
	// cast object into plugin arguments object
	args, ok := obj.(*pluginConfig.PodCountByNodeArgs)
	if !ok {
		klog.Errorf("want args to be of type PodCountByNodeArgs, got %T, using defaults", obj)
		args = &pluginConfig.PodCountByNodeArgs{
			PodCountLimit: []pluginConfig.PodCountLimitSpec{
				pluginConfig.PodCountLimitSpec{
					map[string]string{"node-role.kubernetes.io/master": ""},
					"dev",
					3,
				},
			},
		}
		return args
	}
	return args
}

func mapContainer(x, y map[string]string) bool {
	// x nodelabel
	// y config label
	if len(x) < len(y) {
		return false
	}
	for k, v := range y {
		if _, ok := x[k]; ok {
			if x[k] != v {
				return false
			}
		} else {
			return false
		}
	}
	return true
}
