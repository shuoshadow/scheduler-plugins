package nodeactualload

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	pluginConfig "sigs.k8s.io/scheduler-plugins/pkg/apis/config"
	"sigs.k8s.io/scheduler-plugins/pkg/apis/config/v1beta1"
)

var _ framework.FilterPlugin = &NodeActualLoad{}

const (
	// Name : name of plugin
	Name = "NodeActualLoad"
)

// NodeActualLoad : scheduler plugin
type NodeActualLoad struct {
	handle    framework.Handle
	collector *Collector
	args      *pluginConfig.NodeActualLoadArgs
}

func New(obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	klog.V(4).Infof("Creating new instance of the NodeActualLoad plugin")
	args := getArgs(obj)
	collector, err := newCollector(obj)
	if err != nil {
		return nil, err
	}

	nal := &NodeActualLoad{
		handle:    handle,
		collector: collector,
		args:      args,
	}
	return nal, nil
}

func (nal *NodeActualLoad) Filter(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, node *framework.NodeInfo) *framework.Status {
	klog.Infof("NodeActualLoad filter, pod: %v, node: %v\n", pod.Name, node.Node().Name)
	// get node metrics
	nodeName := node.Node().Name
	metrics := nal.collector.getNodeMetrics(nodeName)
	if metrics == nil {
		klog.Warningf("failure getting metrics for node %q, filter node", nodeName)
		return framework.NewStatus(framework.Error, "Node:"+nodeName+" "+"failure getting metrics this node")
	}
	for _, metric := range metrics {
		if metric.Operator == string(nal.args.CalculateType) && metric.Type == "CPU" {
			klog.V(6).Infof("node: %s, metric cpu value: %f, target cpu rate: %f", nodeName, metric.Value, nal.args.TargetCpuRate)
			if nal.args.TargetCpuRate != 0 &&  metric.Value > nal.args.TargetCpuRate {
				return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("cpu useage too hight, now: %f", metric.Value))
			}
		}
		if metric.Operator == string(nal.args.CalculateType) && metric.Type == "Memory" {
			klog.V(6).Infof("node: %s, metric memory value: %f, target memory rate: %f", nodeName, metric.Value, nal.args.TargetMemoryRate)
			if nal.args.TargetMemoryRate != 0 && metric.Value > nal.args.TargetMemoryRate {
				return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("memory useage too hight, now: %f", metric.Value))
			}
		}
	}
	return framework.NewStatus(framework.Success)
}

// Name : name of plugin
func (nal *NodeActualLoad) Name() string {
	return Name
}

// getArgs : get configured args
func getArgs(obj runtime.Object) *pluginConfig.NodeActualLoadArgs {
	// cast object into plugin arguments object
	args, ok := obj.(*pluginConfig.NodeActualLoadArgs)
	if !ok {
		klog.Errorf("want args to be of type NodeActualLoadArgs, got %T, using defaults", obj)
		args = &pluginConfig.NodeActualLoadArgs{
			MetricProvider: pluginConfig.MetricProviderSpec{
				Type: v1beta1.DefaultMetricProviderType,
			},
			CalculateType:    "avg",
			TargetCpuRate:    0,
			TargetMemoryRate: 0,
		}
		return args
	}
	return args
}
