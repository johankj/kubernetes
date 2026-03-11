/*
Copyright The Kubernetes Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
)

// KubernetesEvictionInterceptorName is the name identifying a core in-tree interceptor.
type KubernetesEvictionInterceptorName string

const (
	// EvictionInterceptorImperativeEviction is a default interceptor that will evict pods using the imperative
	// Eviction API (/evict endpoint) with a backoff.
	EvictionInterceptorImperativeEviction KubernetesEvictionInterceptorName = "imperative-eviction.k8s.io"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:prerelease-lifecycle-gen:introduced=1.36
// +k8s:supportsSubresource="/status"

// EvictionRequest defines a request that should ideally result in a graceful eviction of a
// .spec.target (e.g. termination of a pod).
//
// `.spec.requesters` field should be set and kept updated to preserve the eviction request.
//
// If the target is a pod, the .status.targetInterceptors is populated from Pod's
// .spec.evictionInterceptors.
//
// Interceptors should observe and communicate through the .status to help with the eviction
// of the target when they see their name present in .status.activeInterceptors. InterceptorStatus
// struct should then be periodically updated to indicate the progress or completion of the eviction
// process by each interceptor in .status.intercerptors. If .status.interceptors[].heartbeatTime is
// not updated within 20 minutes, the eviction request is passed over to the next interceptor.
//
// If there are no other interceptor and the target is a pod, the last default
// imperative-eviction.k8s.io interceptor will evict the pod using the imperative Eviction API
// (/evict endpoint).
type EvictionRequest struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is the standard object metadata; More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata.
	// .metadata.name is required and should match the .metadata.uid of the pod being evicted.
	// .metadata.generateName is not supported.
	// The labels of the eviction request object are synchronized with .metadata.labels of the
	// eviction request's target. The labels of the target have a preference.
	// +optional
	// +k8s:subfield(name)=+k8s:format=k8s-uuid
	// +k8s:alpha(since: "1.36")=+k8s:subfield(generateName)=+k8s:forbidden
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// spec defines the eviction request specification.
	// https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +required
	Spec EvictionRequestSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

	// status represents the most recently observed status of the eviction request.
	// Populated by interceptors and eviction request controller.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Status EvictionRequestStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// EvictionRequestSpec is a specification of an EvictionRequest.
type EvictionRequestSpec struct {
	// target contains a reference to an object (e.g. a pod) that should be evicted.
	// Target UID must be the same as the EvictionRequest's .metadata.name.
	// This field is required and immutable.
	// +required
	// +k8s:beta(since: "1.36")=+k8s:immutable
	Target EvictionTarget `json:"target" protobuf:"bytes,1,opt,name=target"`

	// requesters allow you to identify entities, that requested the eviction of the target.
	// At least one requester is required when creating an eviction request.
	// A requester is also required for the eviction request to be processed.
	// An empty list indicates that the eviction request should be canceled.
	//
	// Requester controllers are expected to reconcile this field and not overwrite any changes
	// made by other controllers in order to prevent conflicts and manage ownership.
	//
	// The maximum length of the requesters list is 100.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=name
	// +k8s:optional
	// +k8s:listType=map
	// +k8s:listMapKey=name
	// +k8s:maxItems=100
	Requesters []Requester `json:"requesters,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=requesters"`
}

// EvictionTarget contains a reference to an object that should be evicted.
// +union
type EvictionTarget struct {
	// pod references a pod that is subject to eviction/termination.
	// Pods that are part of a PodGroup (.spec.schedulingGroup is set) are not supported.
	// +optional
	// +k8s:optional
	// +oneOf=TargetSelection
	// +k8s:unionMember
	// +k8s:subfield(name)=+k8s:format=k8s-long-name
	Pod *LocalTargetReference `json:"pod,omitempty" protobuf:"bytes,1,opt,name=pod"`
}

// LocalTargetReference contains enough information to locate the referenced target inside the same namespace.
type LocalTargetReference struct {
	// name of the target.
	// This field is required.
	// +required
	// +k8s:required
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// uid of the target.
	// It can be found in .spec.metadata.uid of the target and is a lowercase UUID in 8-4-4-4-12 format.
	// This field is required.
	// +required
	// +k8s:required
	// +k8s:format=k8s-uuid
	UID apimachinerytypes.UID `json:"uid" protobuf:"bytes,2,opt,name=uid,casttype=k8s.io/kubernetes/pkg/types.UID"`
}

// Requester allows you to identify the entity, that requested the eviction of the target.
// +structType=atomic
type Requester struct {
	// name allows you to identify the entity, that requested the eviction of the target.
	//
	// It must be a fully qualified domain name of at most 253 characters in length, consisting
	// only of lowercase alphanumeric characters, periods and hyphens (e.g. foo.example.com).
	// This field must be unique for each requester.
	// This field is required.
	// +required
	// +k8s:required
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
}

// EvictionRequestStatus represents the last observed status of the eviction request.
type EvictionRequestStatus struct {
	// conditions contain information about the eviction request.
	//
	// Eviction request specific conditions are: Evicted or Canceled (managed by Kubernetes),
	//
	// - Failed means that the eviction request is no longer being processed
	//   by any eviction interceptor. This can happen if the request is canceled or if no interceptor
	//   managee to evict the target (e.g. terminate or delete a pod).
	// - Evicted means that the target has been evicted (e.g. a pod has been terminated or deleted).
	//
	// 	The maximum length of the conditions list is 1000.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	// +k8s:optional
	// +k8s:listType=map
	// +k8s:listMapKey=type
	// +k8s:maxItems=1000
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// observedGeneration is EvictionRequest's .metadata.generation observed by the eviction request controller.
	// The observed generation value cannot be negative and can only be incremented.
	// This field is managed by Kubernetes.
	// +optional
	// +k8s:optional
	// +k8s:minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,2,opt,name=observedGeneration"`

	// targetInterceptors reference interceptors that should eventually respond to this eviction
	// request to help with the graceful eviction of a target. These interceptors are selected
	// sequentially, in the order in which they appear in the list and are added to the
	// .status.activeInterceptors in a rolling fashion.
	//
	// If the target is a pod, the field is populated from Pod's .spec.evictionInterceptors. Default
	// interceptors may be added to the list according to the target.
	//
	// Default interceptors:
	// - imperative-eviction.k8s.io interceptor is appended to the end of the list if the target is
	//   a pod. It will call the /evict API endpoint. This call may not succeed due to
	//   PodDisruptionBudgets, which may block the pod termination. It will update the interceptor
	//   message and try again with a backoff.
	//
	// The maximum length of the interceptors list is 16.
	// This field is immutable once set.
	// This field is managed by Kubernetes.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=name
	// +k8s:optional
	// +k8s:listType=map
	// +k8s:listMapKey=name
	// +k8s:maxItems=16
	TargetInterceptors []TargetInterceptor `json:"targetInterceptors,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,3,rep,name=targetInterceptors"`

	// activeInterceptors store a list of interceptors that should currently interact with the
	// eviction process by updating .status.interceptors[], where .name is the active interceptor
	// name. InterceptorStatus fields should be periodically updated to indicate the progress or
	// completion of the eviction process. If .status.interceptors[].heartbeatTime field is not
	// updated within 20 minutes, the eviction request is passed over to the next interceptor.
	//
	// The maximum allowed number of active interceptors is 1. An active interceptor is removed from
	// this list when .status.interceptors[].completionTime is set.
	// This field is managed by Kubernetes.
	// +optional
	// +listType=atomic
	// +k8s:optional
	// +k8s:listType=set
	// +k8s:maxItems=1
	ActiveInterceptors []string `json:"activeInterceptors" protobuf:"bytes,4,opt,name=activeInterceptors"`

	// processedInterceptors store a list of interceptors that have previously been selected
	// and added to ActiveInterceptors. These interceptors may have reached completion, been
	// canceled, failed to start or failed to update .interceptors[].heartbeatTime` in a timely
	// manner.
	// Please refer to the InterceptorStatus for each interceptor for more details.
	// This field is managed by Kubernetes.
	// +optional
	// +listType=atomic
	// +k8s:optional
	// +k8s:listType=set
	// +k8s:maxItems=16
	ProcessedInterceptors []string `json:"processedInterceptors,omitempty" protobuf:"bytes,5,opt,name=processedInterceptors"`

	// interceptors represents the eviction process status of each declared interceptor. Only
	// ActiveInterceptors should update the interceptor statuses.
	//
	// The interceptor list should be the same length and have the same .name fields as
	// .status.targetInterceptors. Only interceptors with .name that are included in
	// .status.activeInterceptors can be mutated. First initialization of the list is allowed.
	//
	// Each InterceptorStatus is managed by the designated interceptor.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=name
	// +k8s:optional
	// +k8s:listType=map
	// +k8s:listMapKey=name
	// +k8s:maxItems=16
	Interceptors []InterceptorStatus `json:"interceptors,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,6,rep,name=interceptors"`
}

type EvictionRequestConditionType string

// These are built-in conditions of an eviction request.
const (
	// EvictionRequestConditionFailed means that the eviction request is no longer being processed
	// by any eviction interceptor. This can happen if the request is canceled or if no interceptor
	// managee to evict the target (e.g. terminate or delete a pod).
	EvictionRequestConditionFailed EvictionRequestConditionType = "Failed"

	// EvictionRequestConditionEvicted means that the target has been evicted (e.g. a pod has been
	// terminated or deleted).
	EvictionRequestConditionEvicted EvictionRequestConditionType = "Evicted"
)

type EvictionRequestConditionReason string

// These are built-in condition reasons of an eviction request.
const (
	// EvictionRequestConditionReasonEvictionRequestInvalid means that the EvictionRequest is not accepted because the
	// initial configuration is not valid.
	// This reason is set for the Failed condition.
	EvictionRequestConditionReasonEvictionRequestInvalid EvictionRequestConditionReason = "EvictionRequestInvalid"
	// EvictionRequestConditionReasonCanceledDueToNoRequesters means that the EvictionRequest is canceled because it has
	// no requesters.
	// This reason is set for the Failed condition.
	EvictionRequestConditionReasonCanceledDueToNoRequesters EvictionRequestConditionReason = "CanceledDueToNoRequesters"
	// EvictionRequestConditionReasonNoFurtherInterceptor means that the EvictionRequest interceptors failed to evict
	// the target and that no further interceptor is available.
	// This reason is set for the Failed condition.
	EvictionRequestConditionReasonNoFurtherInterceptor EvictionRequestConditionReason = "NoFurtherInterceptor"
	// EvictionRequestConditionReasonPodDeleted means that the target pod has been deleted.
	// This reason is set for the Evicted condition.
	EvictionRequestConditionReasonPodDeleted EvictionRequestConditionReason = "PodDeleted"
	// EvictionRequestConditionReasonPodTerminal means that the target pod has reached a terminal state.
	// This reason is set for the Evicted condition.
	EvictionRequestConditionReasonPodTerminal EvictionRequestConditionReason = "PodTerminal"
)

// TargetInterceptor allows you to specify the interceptor responding to the EvictionRequest.
// Interceptors should observe and communicate through the EvictionRequest API to help with
// the graceful eviction of a target (e.g. termination of a pod).
// +structType=atomic
type TargetInterceptor struct {
	// name allows you to identify the interceptor responding to the EvictionRequest.
	//
	// It must be a fully qualified domain name of at most 253 characters in length, consisting
	// only of lowercase alphanumeric characters, periods and hyphens (e.g. bar.example.com).
	// This field must be unique for each interceptor.
	// This field is required.
	// +required
	// +k8s:required
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
}

// InterceptorStatus represents the last observed status of the eviction process of the interceptor.
// It should be only updated by the designated interceptor whose name is .name field.
// +structType=granular
type InterceptorStatus struct {
	// name must be a fully qualified domain name of at most 253 characters in length, consisting
	// only of lowercase alphanumeric characters, periods and hyphens (e.g. bar.example.com).
	// This field is initialized by Kubernetes and must be unique for each interceptor.
	// This field is required.
	// +required
	// +k8s:required
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// startTime tracks the time at which this interceptor was designated as active and should start
	// processing the eviction request.
	// It should reflect the present time when set.
	// This field is initialized by Kubernetes when this interceptor becomes active.
	// This field becomes immutable once set.
	// +optional
	// +k8s:optional
	// +k8s:beta(since: "1.36")=+k8s:update=NoModify
	// +k8s:beta(since: "1.36")=+k8s:update=NoUnset
	StartTime *metav1.Time `json:"startTime,omitempty" protobuf:"bytes,2,opt,name=startTime"`

	// heartbeatTime is the last time at which the eviction process was reported to be in progress
	// by the interceptor.
	// It should reflect the present time when set.
	// Interceptors should avoid heartbeats more frequent than 20 seconds to avoid overloading the
	// control-plane.
	// +optional
	// +k8s:optional
	HeartbeatTime *metav1.Time `json:"heartbeatTime,omitempty" protobuf:"bytes,3,opt,name=heartbeatTime"`

	// expectedCompletionTime is the time at which the eviction process step is expected to end for the
	// interceptor.
	// The time cannot be set to the past.
	// May be omitted if no estimate can be made.
	// +optional
	// +k8s:optional
	ExpectedCompletionTime *metav1.Time `json:"expectedCompletionTime,omitempty" protobuf:"bytes,4,opt,name=expectedCompletionTime"`

	// completionTime tracks the time at which the Interceptor stopped processing the eviction request.
	// Completion means that the interceptors has either fully or partially completed the
	// eviction process, which may have resulted in target eviction (e.g. pod termination).
	// It should reflect the present time when set.
	// This field becomes immutable once set.
	// +optional
	// +k8s:optional
	// +k8s:beta(since: "1.36")=+k8s:update=NoModify
	// +k8s:beta(since: "1.36")=+k8s:update=NoUnset
	CompletionTime *metav1.Time `json:"completionTime,omitempty" protobuf:"bytes,5,opt,name=completionTime"`

	// message provides human-readable details about the state of the interceptor and the eviction
	// process.
	// Maximum length is 4000 characters. The string is truncated if it exceeds this limit.
	// +optional
	// +k8s:optional
	// +k8s:beta(since: "1.36")=+k8s:maxLength=4000
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:prerelease-lifecycle-gen:introduced=1.36

// EvictionRequestList contains a list of EvictionRequests resources.
type EvictionRequestList struct {
	metav1.TypeMeta `json:",inline"`
	// metadata is the standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// items is the list of EvictionRequests.
	Items []EvictionRequest `json:"items" protobuf:"bytes,2,rep,name=items"`
}
