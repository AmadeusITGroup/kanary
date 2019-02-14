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

package enqueue

import (
	"github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ handler.EventHandler = &RequestForKanaryLabel{}

// RequestForKanaryLabel enqueues Requests for the KanaryDeployment corresponding to the label value.
type RequestForKanaryLabel struct {
}

// Create implements EventHandler
func (e *RequestForKanaryLabel) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	add(evt.Meta, q)
}

// Update implements EventHandler
func (e *RequestForKanaryLabel) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	add(evt.MetaOld, q)
	add(evt.MetaNew, q)
}

// Delete implements EventHandler
func (e *RequestForKanaryLabel) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	add(evt.Meta, q)
}

// Generic implements EventHandler
func (e *RequestForKanaryLabel) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	add(evt.Meta, q)
}

func add(meta metav1.Object, q workqueue.RateLimitingInterface) {
	value, ok := meta.GetLabels()[v1alpha1.KanaryDeploymentKanaryNameLabelKey]
	if ok {
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: meta.GetNamespace(),
				Name:      value,
			},
		}
		q.Add(req)
	}
}
