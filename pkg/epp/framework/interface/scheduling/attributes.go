/*
Copyright 2025 The Kubernetes Authors.

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

package scheduling

import (
	"sync"

	"github.com/llm-d/llm-d-router/pkg/epp/framework/interface/plugin"
)

// PutAttribute stores value at key in the request's attribute store.
// The backing store is lazily allocated on first write.
// Callers must not write concurrently to the same request from multiple goroutines.
//
// New code should call PutAttributeKey with a declared DataKey so the call
// site carries the producer's key. The string form remains for plugins that
// synthesize keys outside the data-attribute registry (tests, plugins that
// publish user-configured keys).
func (r *InferenceRequest) PutAttribute(key string, value any) {
	if r.attributes == nil {
		r.attributes = &sync.Map{}
	}
	r.attributes.Store(key, value)
}

// PutAttributeKey stores value at dk.String() in the request's attribute
// store. The DataKey form is the natural counterpart to *datalayer.Slot[T]
// for the request-side store, which is any-backed rather than Cloneable-
// backed; the slot's compile-time type still catches mismatches at the
// assignment boundary.
func (r *InferenceRequest) PutAttributeKey(dk plugin.DataKey, value any) {
	if r.attributes == nil {
		r.attributes = &sync.Map{}
	}
	r.attributes.Store(dk.String(), value)
}

// GetAttribute returns the value stored at key, or nil and false if absent.
// Prefer ReadRequestAttribute for type-safe access.
func (r *InferenceRequest) GetAttribute(key string) (any, bool) {
	if r.attributes == nil {
		return nil, false
	}
	return r.attributes.Load(key)
}

// AttributeKeys returns the keys currently present in the request's attribute store.
// The order is unspecified.
func (r *InferenceRequest) AttributeKeys() []string {
	keys := make([]string, 0)
	if r.attributes == nil {
		return keys
	}
	r.attributes.Range(func(k, _ any) bool {
		if s, ok := k.(string); ok {
			keys = append(keys, s)
		}
		return true
	})
	return keys
}

// ReadRequestAttribute returns the value stored at key, type-asserted to T.
// It returns the zero value of T and false if the key is missing or the value
// is not of type T.
func ReadRequestAttribute[T any](r *InferenceRequest, key string) (T, bool) {
	var zero T
	v, ok := r.GetAttribute(key)
	if !ok {
		return zero, false
	}
	t, ok := v.(T)
	if !ok {
		return zero, false
	}
	return t, true
}
