/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	k8sframework "k8s.io/kubernetes/pkg/scheduler/framework"
)

// TaskStatus defines the status of a task/pod.
type TaskStatus int

const (
	// Pending means the task is pending in the apiserver.
	Pending TaskStatus = 1 << iota

	// Allocated means the scheduler assigns a host to it.
	Allocated

	// Pipelined means the scheduler assigns a host to wait for releasing resource.
	Pipelined

	// Binding means the scheduler send Bind request to apiserver.
	Binding

	// Bound means the task/Pod bounds to a host.
	Bound

	// Running means a task is running on the host.
	Running

	// Releasing means a task/pod is deleted.
	Releasing

	// Succeeded means that all containers in the pod have voluntarily terminated
	// with a container exit code of 0, and the system is not going to restart any of these containers.
	Succeeded

	// Failed means that all containers in the pod have terminated, and at least one container has
	// terminated in a failure (exited with a non-zero exit code or was stopped by the system).
	Failed

	// Unknown means the status of task/pod is unknown to the scheduler.
	Unknown
)

func (ts TaskStatus) String() string {
	switch ts {
	case Pending:
		return "Pending"
	case Allocated:
		return "Allocated"
	case Pipelined:
		return "Pipelined"
	case Binding:
		return "Binding"
	case Bound:
		return "Bound"
	case Running:
		return "Running"
	case Releasing:
		return "Releasing"
	case Succeeded:
		return "Succeeded"
	case Failed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// NodePhase defines the phase of node
type NodePhase int

const (
	// Ready means the node is ready for scheduling
	Ready NodePhase = 1 << iota
	// NotReady means the node is not ready for scheduling
	NotReady
)

func (np NodePhase) String() string {
	switch np {
	case Ready:
		return "Ready"
	case NotReady:
		return "NotReady"
	}

	return "Unknown"
}

// validateStatusUpdate validates whether the status transfer is valid.
func validateStatusUpdate(oldStatus, newStatus TaskStatus) error {
	return nil
}

// LessFn is the func declaration used by sort or priority queue.
type LessFn func(interface{}, interface{}) bool

// CompareFn is the func declaration used by sort or priority queue.
type CompareFn func(interface{}, interface{}) int

// ValidateFn is the func declaration used to check object's status.
type ValidateFn func(interface{}) bool

// ValidateResult is struct to which can used to determine the result
type ValidateResult struct {
	Pass    bool
	Reason  string
	Message string
}

// ValidateExFn is the func declaration used to validate the result.
type ValidateExFn func(interface{}) *ValidateResult

// VoteFn is the func declaration used to check object's complicated status.
type VoteFn func(interface{}) int

// JobEnqueuedFn is the func declaration used to call after job enqueued.
type JobEnqueuedFn func(interface{})

// PredicateFn is the func declaration used to predicate node for task.
type PredicateFn func(*TaskInfo, *NodeInfo) error

// PrePredicateFn is the func declaration used to pre-predicate node for task.
type PrePredicateFn func(*TaskInfo) error

// BestNodeFn is the func declaration used to return the nodeScores to plugins.
type BestNodeFn func(*TaskInfo, map[float64][]*NodeInfo) *NodeInfo

// EvictableFn is the func declaration used to evict tasks.
type EvictableFn func(*TaskInfo, []*TaskInfo) ([]*TaskInfo, int)

// NodeOrderFn is the func declaration used to get priority score for a node for a particular task.
type NodeOrderFn func(*TaskInfo, *NodeInfo) (float64, error)

// BatchNodeOrderFn is the func declaration used to get priority score for ALL nodes for a particular task.
type BatchNodeOrderFn func(*TaskInfo, []*NodeInfo) (map[string]float64, error)

// NodeMapFn is the func declaration used to get priority score for a node for a particular task.
type NodeMapFn func(*TaskInfo, *NodeInfo) (float64, error)

// NodeReduceFn is the func declaration used to reduce priority score for a node for a particular task.
type NodeReduceFn func(*TaskInfo, k8sframework.NodeScoreList) error

// NodeOrderMapFn is the func declaration used to get priority score of all plugins for a node for a particular task.
type NodeOrderMapFn func(*TaskInfo, *NodeInfo) (map[string]float64, float64, error)

// NodeOrderReduceFn is the func declaration used to reduce priority score of all nodes for a plugin for a particular task.
type NodeOrderReduceFn func(*TaskInfo, map[string]k8sframework.NodeScoreList) (map[string]float64, error)

// TargetJobFn is the func declaration used to select the target job satisfies some conditions
type TargetJobFn func([]*JobInfo) *JobInfo

// ReservedNodesFn is the func declaration used to select the reserved nodes
type ReservedNodesFn func()

// VictimTasksFn is the func declaration used to select victim tasks
type VictimTasksFn func([]*TaskInfo) []*TaskInfo

// AllocatableFn is the func declaration used to check whether the task can be allocated
type AllocatableFn func(*QueueInfo, *TaskInfo) bool

// CustomResourceKind specify the kind for the custom resource
type CustomResourceKind string

// CustomResourceKey specify the index for the custom resource
type CustomResourceKey string

// CustomResource interface is the resource that could be cached into scheduler cache
type CustomResource interface {
	DeepCopy() CustomResource
	Update(CustomResource) error
}

type GenericCode int

const (
	// GenericSuccess the result if success
	GenericSuccess GenericCode = iota
	// GenericFailed the result is failed
	GenericFailed
	// GenericSkip the result should be safely discard
	GenericSkip
)

// GenericResult is the result of the result of the generic functions
type GenericResult struct {
	code    GenericCode
	reasons []string
	err     error
}

// NewGenericResult makes a result out of the given arguments and returns its pointer.
func NewGenericResult(code GenericCode, reasons ...string) *GenericResult {
	s := &GenericResult{
		code:    code,
		reasons: reasons,
	}
	if code == GenericFailed {
		s.err = errors.New(strings.Join(reasons, ","))
	}
	return s
}

func (gr *GenericResult) Failed() bool {
	return gr.code == GenericFailed
}

func (gr *GenericResult) Success() bool {
	return gr == nil || gr.code == GenericSuccess
}

// AsError returns nil if the result is a success; otherwise returns an "error" object
// with a concatenated message on reasons of the result.
func (gr *GenericResult) AsError() error {
	if !gr.Failed() {
		return nil
	}
	if gr.err != nil {
		return gr.err
	}
	return errors.New(strings.Join(gr.reasons, ", "))
}

// AsGenericResult cast normal error into genericResult
func AsGenericResult(err error) *GenericResult {
	return &GenericResult{
		code:    GenericFailed,
		reasons: []string{err.Error()},
		err:     err,
	}
}

// GenericParamCheck checks the parameters number and types to make sure right predication called
func GenericParamCheck(types []reflect.Type, i ...interface{}) *GenericResult {
	if len(types) != len(i) {
		return NewGenericResult(GenericSkip, fmt.Sprintf("parameter number mismatch expect: %d, actual: %d", len(types), len(i)))
	}
	for j, t := range types {
		if t != reflect.TypeOf(i[j]) {
			return NewGenericResult(GenericSkip, fmt.Sprintf("parameter type mismatch expect: %T, actual: %T", t, i[j]))
		}
	}
	return NewGenericResult(GenericSuccess)
}

// GenericPredicateFn is a generic filter function that accept any resources as parameters
type GenericPredicateFn func(...interface{}) *GenericResult

// GenericOrderFn is a generic filter function that accept any resources as parameters
type GenericOrderFn func(...interface{}) (float64, *GenericResult)
