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

package datalayer

import (
	"sync"

	"github.com/llm-d/llm-d-router/pkg/epp/framework/interface/plugin"
)

// Cloneable types support cloning of the value.
type Cloneable interface {
	Clone() Cloneable
}

// DynamicAttribute wraps a getter function to allow on-demand resolution of attributes.
type DynamicAttribute struct {
	Get func() Cloneable
}

// Clone implements Cloneable. It copies the function pointer.
func (d *DynamicAttribute) Clone() Cloneable {
	if d == nil {
		return nil
	}
	return &DynamicAttribute{Get: d.Get}
}

// AttributeMap is used to store flexible metadata or traits
// across different aspects of an inference server.
// Stored values must be Cloneable.
//
// Put is the canonical write entry point and is what Slot[T].Put calls.
// PutKey on the concrete *Attributes is the DataKey-based convenience for
// callers that hold a DataKey but no Slot (the slot is the recommended path
// because it pins the value type at compile time).
type AttributeMap interface {
	// Put stores val under key. New code should call PutKey on *Attributes
	// or use a *Slot[T] so the call site carries the declared DataKey.
	Put(string, Cloneable)
	Get(string) (Cloneable, bool)
	Keys() []string
	Clone() AttributeMap
}

// Attributes provides a goroutine-safe implementation of AttributeMap.
type Attributes struct {
	data sync.Map // key: attribute name (string), value: attribute value (opaque, Cloneable)
}

// NewAttributes returns a new instance of Attributes.
func NewAttributes() AttributeMap {
	return &Attributes{}
}

// Put adds or updates an attribute in the map.
//
// New code should call PutKey with a declared DataKey so the call site
// carries the producer's key. The string form remains for plugins
// (notably the dynamic-attribute installer and the custom-metrics
// extractor) that synthesize keys outside the data-attribute registry.
func (a *Attributes) Put(key string, value Cloneable) {
	if value != nil {
		a.data.Store(key, value) // TODO: Clone into map to ensure isolation
	}
}

// PutKey stores val under dk.String(). It is the DataKey-based counterpart
// to Put; the Slot writes through this method so the producer-side type
// check (compile-time via Slot[T]) is the single source of value-safety.
func (a *Attributes) PutKey(dk plugin.DataKey, value Cloneable) {
	a.Put(dk.String(), value)
}

// Get retrieves an attribute by key, returning a cloned copy (or resolving it dynamically).
func (a *Attributes) Get(key string) (Cloneable, bool) {
	val, ok := a.data.Load(key)
	if !ok {
		return nil, false
	}

	if dynamic, ok := val.(*DynamicAttribute); ok {
		realVal := dynamic.Get()
		if realVal == nil {
			return nil, false
		}
		return realVal.Clone(), true
	}

	if cloneable, ok := val.(Cloneable); ok {
		return cloneable.Clone(), true
	}
	return nil, false
}

// Keys returns all keys in the attribute map.
func (a *Attributes) Keys() []string {
	var keys []string
	a.data.Range(func(key, _ any) bool {
		if sk, ok := key.(string); ok {
			keys = append(keys, sk)
		}
		return true
	})
	return keys
}

// Clone creates a deep copy of the entire attribute map.
func (a *Attributes) Clone() AttributeMap {
	clone := NewAttributes()
	a.data.Range(func(key, value any) bool {
		if sk, ok := key.(string); ok {
			if v, ok := value.(Cloneable); ok {
				clone.Put(sk, v)
			}
		}
		return true
	})
	return clone
}

// ReadAttribute retrieves attribute with the given key from AttributeMap and asserts it to type T.
// Second return value is 'false' if the key is not found or the type assertion fails.
func ReadAttribute[T Cloneable](attributeMap AttributeMap, key string) (T, bool) {
	var zero T

	raw, ok := attributeMap.Get(key)
	if !ok {
		return zero, false
	}

	val, ok := raw.(T)
	if !ok {
		return zero, false
	}

	return val, true
}
