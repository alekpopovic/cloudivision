package kube

import (
	"context"
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateStatusWithRetry writes obj.Status and retries Kubernetes resource
// version conflicts by reloading the latest object before each attempt.
func UpdateStatusWithRetry(ctx context.Context, c client.Client, obj client.Object) error {
	if c == nil {
		return fmt.Errorf("kubernetes client is nil")
	}
	if obj == nil {
		return fmt.Errorf("object is nil")
	}
	key := client.ObjectKeyFromObject(obj)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest, err := newObjectLike(obj)
		if err != nil {
			return err
		}
		if err := c.Get(ctx, key, latest); err != nil {
			return err
		}
		if err := copyStatus(latest, obj); err != nil {
			return err
		}
		if err := c.Status().Update(ctx, latest); err != nil {
			return err
		}
		return copyStatus(obj, latest)
	})
}

func newObjectLike(obj client.Object) (client.Object, error) {
	t := reflect.TypeOf(obj)
	if t == nil {
		return nil, fmt.Errorf("object type is nil")
	}
	if t.Kind() != reflect.Pointer {
		return nil, fmt.Errorf("object %T is not a pointer", obj)
	}
	created, ok := reflect.New(t.Elem()).Interface().(client.Object)
	if !ok {
		return nil, fmt.Errorf("object %T does not implement client.Object", obj)
	}
	return created, nil
}

func copyStatus(dst, src client.Object) error {
	dstStatus, err := statusField(dst)
	if err != nil {
		return err
	}
	srcStatus, err := statusField(src)
	if err != nil {
		return err
	}
	desiredStatus := reflect.New(srcStatus.Type()).Elem()
	desiredStatus.Set(srcStatus)
	mergeConditions(dstStatus, desiredStatus)
	dstStatus.Set(desiredStatus)
	return nil
}

// mergeConditions keeps condition types written by another actor while the
// status update was retried. Desired conditions win by type; unrelated latest
// conditions survive the retry.
func mergeConditions(latestStatus, desiredStatus reflect.Value) {
	if latestStatus.Kind() != reflect.Struct || desiredStatus.Kind() != reflect.Struct {
		return
	}
	latest := latestStatus.FieldByName("Conditions")
	desired := desiredStatus.FieldByName("Conditions")
	conditionSliceType := reflect.TypeOf([]metav1.Condition{})
	if !latest.IsValid() || !desired.IsValid() || latest.Type() != conditionSliceType || desired.Type() != conditionSliceType || !desired.CanSet() {
		return
	}
	merged := append([]metav1.Condition(nil), latest.Interface().([]metav1.Condition)...)
	for _, condition := range desired.Interface().([]metav1.Condition) {
		replaced := false
		for i := range merged {
			if merged[i].Type == condition.Type {
				merged[i] = condition
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, condition)
		}
	}
	desired.Set(reflect.ValueOf(merged))
}

func statusField(obj client.Object) (reflect.Value, error) {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return reflect.Value{}, fmt.Errorf("object %T is not a non-nil pointer", obj)
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("object %T does not point to a struct", obj)
	}
	status := elem.FieldByName("Status")
	if !status.IsValid() {
		return reflect.Value{}, fmt.Errorf("object %T has no Status field", obj)
	}
	if !status.CanSet() {
		return reflect.Value{}, fmt.Errorf("object %T Status field cannot be set", obj)
	}
	return status, nil
}
