package controller

import (
	"fmt"

	rapi "github.com/appscode/stash/api"
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/docker"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (c *Controller) IsPreferredAPIResource(groupVersion, kind string) bool {
	if resourceList, err := c.KubeClient.Discovery().ServerPreferredResources(); err == nil {
		for _, resources := range resourceList {
			if resources.GroupVersion != groupVersion {
				continue
			}
			for _, resource := range resources.APIResources {
				if resources.GroupVersion == groupVersion && resource.Kind == kind {
					return true
				}
			}
		}
	}
	return false
}

func (c *Controller) FindRestic(obj metav1.ObjectMeta) (*sapi.Restic, error) {
	restics, err := c.StashClient.Restics(obj.Namespace).List(metav1.ListOptions{})
	if kerr.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	for _, restic := range restics.Items {
		selector, err := metav1.LabelSelectorAsSelector(&restic.Spec.Selector)
		//return nil, fmt.Errorf("invalid selector: %v", err)
		if err == nil {
			if selector.Matches(labels.Set(obj.Labels)) {
				return &restic, nil
			}
		}
	}
	return nil, nil
}

func LabelSelectorRequirementsAsSelector(selector *metav1.LabelSelector) (labels.Selector, error) {
	result := labels.SelectorFromSet(selector.MatchLabels)
	for _, expr := range selector.MatchExpressions {
		var op selection.Operator
		switch expr.Operator {
		case metav1.LabelSelectorOpIn:
			op = selection.In
		case metav1.LabelSelectorOpNotIn:
			op = selection.NotIn
		case metav1.LabelSelectorOpExists:
			op = selection.Exists
		case metav1.LabelSelectorOpDoesNotExist:
			op = selection.DoesNotExist
		default:
			return nil, fmt.Errorf("%q is not a valid label selector operator", expr.Operator)
		}
		r, err := labels.NewRequirement(expr.Key, op, expr.Values)
		if err != nil {
			return nil, err
		}
		result = result.Add(*r)
	}
	return result, nil
}

func (c *Controller) restartPods(namespace string, selector *metav1.LabelSelector) error {
	// ref: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
	// ref: https://github.com/kubernetes/kubernetes/blob/310ea94b6e0694ab08e3aa6185919be92419e932/pkg/api/helper/helpers.go#L357
	result := labels.SelectorFromSet(selector.MatchLabels)
	for _, expr := range selector.MatchExpressions {
		var op selection.Operator
		switch expr.Operator {
		case metav1.LabelSelectorOpIn:
			op = selection.In
		case metav1.LabelSelectorOpNotIn:
			op = selection.NotIn
		case metav1.LabelSelectorOpExists:
			op = selection.Exists
		case metav1.LabelSelectorOpDoesNotExist:
			op = selection.DoesNotExist
		default:
			return fmt.Errorf("%q is not a valid label selector operator", expr.Operator)
		}
		r, err := labels.NewRequirement(expr.Key, op, expr.Values)
		if err != nil {
			return err
		}
		result = result.Add(*r)
	}
	return c.KubeClient.CoreV1().Pods(namespace).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: result.String(),
	})
}

func getString(m map[string]string, key string) string {
	if m == nil {
		return ""
	}
	return m[key]
}

func (c *Controller) GetSidecarContainer(r *rapi.Restic) apiv1.Container {
	tag := c.SidecarImageTag
	if r.Annotations != nil {
		if v, ok := r.Annotations[sapi.VersionTag]; ok {
			tag = v
		}
	}

	sidecar := apiv1.Container{
		Name:            docker.StashContainer,
		Image:           docker.ImageOperator + ":" + tag,
		ImagePullPolicy: apiv1.PullIfNotPresent,
		Args: []string{
			"crond",
			"--v=3",
			"--namespace=" + r.Namespace,
			"--name=" + r.Name,
		},
		VolumeMounts: []apiv1.VolumeMount{
			{
				Name:      r.Spec.Source.VolumeName,
				MountPath: r.Spec.Source.Path,
			},
		},
	}
	backupVolumeMount := apiv1.VolumeMount{
		Name:      r.Spec.Destination.Volume.Name,
		MountPath: r.Spec.Destination.Path,
	}
	sidecar.VolumeMounts = append(sidecar.VolumeMounts, backupVolumeMount)
	return sidecar
}

func (c *Controller) addAnnotation(r *rapi.Restic) {
	if r.ObjectMeta.Annotations == nil {
		r.ObjectMeta.Annotations = make(map[string]string)
	}
	r.ObjectMeta.Annotations[sapi.VersionTag] = c.SidecarImageTag
}

func removeContainer(c []apiv1.Container, name string) []apiv1.Container {
	for i, v := range c {
		if v.Name == name {
			c = append(c[:i], c[i+1:]...)
			break
		}
	}
	return c
}

func removeVolume(volumes []apiv1.Volume, name string) []apiv1.Volume {
	for i, v := range volumes {
		if v.Name == name {
			volumes = append(volumes[:i], volumes[i+1:]...)
			break
		}
	}
	return volumes
}
