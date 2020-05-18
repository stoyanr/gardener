// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"context"

	"github.com/pkg/errors"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	maxRetries = 3
)

var systemMetadataFields = []string{"uid", "resourceVersion", "generation", "selfLink", "creationTimestamp", "deletionTimestamp", "deletionGracePeriodSeconds", "managedFields"}

// GetObjectByRef returns the object with the given reference and namespace using the given client.
// The full content of the object is returned as map[string]interface{}, except for system metadata fields.
// This function can be combined with runtime.DefaultUnstructuredConverter.FromUnstructured to get the object content
// as runtime.RawExtension.
func GetObjectByRef(ctx context.Context, c client.Client, ref *autoscalingv1.CrossVersionObjectReference, namespace string) (map[string]interface{}, error) {
	gvk, err := gvkFromCrossVersionObjectReference(ref)
	if err != nil {
		return nil, err
	}
	return GetObject(ctx, c, gvk, ref.Name, namespace)
}

// GetObjectByRef returns the object with the given GVK, name, and namespace as a map using the given client.
// The full content of the object is returned as map[string]interface{}, except for system metadata fields.
// This function can be combined with runtime.DefaultUnstructuredConverter.FromUnstructured to get the object content
// as runtime.RawExtension.
func GetObject(ctx context.Context, c client.Client, gvk schema.GroupVersionKind, name, namespace string) (map[string]interface{}, error) {
	// Initialize the object
	key := client.ObjectKey{Namespace: namespace, Name: name}
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	// Get the object
	if err := c.Get(ctx, key, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "could not get object %s %s", gvk, key)
		}

		// The object was not found
		return nil, nil
	}

	return filterMetadata(obj.UnstructuredContent(), systemMetadataFields...), nil
}

// CreateOrUpdateObjectByRef creates or updates the object with the given reference and namespace using the given client.
// The object is created or updated with the given content, except for system metadata fields.
// This function can be combined with runtime.DefaultUnstructuredConverter.ToUnstructured to create or update an object
// from runtime.RawExtension.
func CreateOrUpdateObjectByRef(ctx context.Context, c client.Client, ref *autoscalingv1.CrossVersionObjectReference, namespace string, content map[string]interface{}) error {
	gvk, err := gvkFromCrossVersionObjectReference(ref)
	if err != nil {
		return err
	}
	return CreateOrUpdateObject(ctx, c, gvk, ref.Name, namespace, content)
}

// CreateOrUpdateObject creates or updates the object with the given GVK, name, and namespace using the given client.
// The object is created or updated with the given content, except for system metadata fields, namespace, and name.
// This function can be combined with runtime.DefaultUnstructuredConverter.ToUnstructured to create or update an object
// from runtime.RawExtension.
func CreateOrUpdateObject(ctx context.Context, c client.Client, gvk schema.GroupVersionKind, name, namespace string, content map[string]interface{}) error {
	// Initialize the object
	key := client.ObjectKey{Namespace: namespace, Name: name}
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetName(name)
	obj.SetNamespace(namespace)

	// Create or update the object with retries for optimistic concurrency
	for retries, done := 0, false; !done; retries++ {
		// Check if the object already exists
		found := true
		if err := c.Get(ctx, key, obj); err != nil {
			if !apierrors.IsNotFound(err) {
				return errors.Wrapf(err, "could not get object %s %s", gvk, key)
			}

			// The object was not found
			found = false
		}

		// Set object content
		if content != nil {
			obj.SetUnstructuredContent(mergeObjectContents(obj.UnstructuredContent(),
				filterMetadata(content, add(systemMetadataFields, "namespace", "name")...)))
		}

		// Create or update the object
		var err error
		if !found {
			err = c.Create(ctx, obj)
		} else {
			err = c.Update(ctx, obj)
		}
		if err != nil && (!apierrors.IsConflict(err) || retries == maxRetries) {
			return errors.Wrapf(err, "could not create or update object %s %s", gvk, key)
		}

		done = err == nil
	}
	return nil
}

// DeleteObjectByRef deletes the object with the given reference and namespace using the given client.
func DeleteObjectByRef(ctx context.Context, c client.Client, ref *autoscalingv1.CrossVersionObjectReference, namespace string) error {
	gvk, err := gvkFromCrossVersionObjectReference(ref)
	if err != nil {
		return err
	}
	return DeleteObject(ctx, c, gvk, ref.Name, namespace)
}

// DeleteObject deletes the object with the given GVK, name, and namespace using the given client.
func DeleteObject(ctx context.Context, c client.Client, gvk schema.GroupVersionKind, name, namespace string) error {
	// Initialize the object
	key := client.ObjectKey{Namespace: namespace, Name: name}
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	// Delete the object with retries for optimistic concurrency
	for retries, done := 0, false; !done; retries++ {
		// Get the object
		if err := c.Get(ctx, key, obj); err != nil {
			if !apierrors.IsNotFound(err) {
				return errors.Wrapf(err, "could not get object %s %s", gvk, key)
			}

			// The object was not found
			return nil
		}

		// Delete the object
		err := c.Delete(ctx, obj)
		if err != nil && (!apierrors.IsConflict(err) || retries == maxRetries) {
			return errors.Wrapf(err, "could not delete object %s %s", gvk, key)
		}

		done = err == nil
	}
	return nil
}

func gvkFromCrossVersionObjectReference(ref *autoscalingv1.CrossVersionObjectReference) (schema.GroupVersionKind, error) {
	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return schema.GroupVersionKind{}, errors.Wrap(err, "invalid API version in object reference")
	}
	return schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    ref.Kind,
	}, nil
}

func mergeObjectContents(dest, src map[string]interface{}) map[string]interface{} {
	// Merge metadata
	srcMetadata, srcMetadataOK := src["metadata"].(map[string]interface{})
	if srcMetadataOK {
		destMetadata, destMetadataOK := dest["metadata"].(map[string]interface{})
		if destMetadataOK {
			dest["metadata"] = MergeMaps(destMetadata, srcMetadata)
		} else {
			dest["metadata"] = srcMetadata
		}
	}

	// Take spec and data from the source
	for _, key := range []string{"spec", "data", "stringData"} {
		srcSpec, srcSpecOK := src[key]
		if srcSpecOK {
			dest[key] = srcSpec
		} else {
			delete(dest, key)
		}
	}

	return dest
}

func filterMetadata(content map[string]interface{}, fields ...string) map[string]interface{} {
	// Copy content to result
	result := make(map[string]interface{})
	for key, value := range content {
		result[key] = value
	}

	// Delete specified fields from result
	if metadata, ok := result["metadata"].(map[string]interface{}); ok {
		for _, field := range fields {
			delete(metadata, field)
		}
	}
	return result
}

func add(s []string, elems ...string) []string {
	result := make([]string, len(s))
	copy(result, s)
	return append(result, elems...)
}
