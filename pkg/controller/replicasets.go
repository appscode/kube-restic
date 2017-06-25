package controller

import (
	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/util"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Controller) WatchReplicaSets() {
	if !util.IsPreferredAPIResource(c.KubeClient, extensions.SchemeGroupVersion.String(), "ReplicaSet") {
		log.Warningf("Skipping watching non-preferred GroupVersion:%s Kind:%s", extensions.SchemeGroupVersion.String(), "ReplicaSet")
		return
	}

	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.KubeClient.ExtensionsV1beta1().ReplicaSets(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.KubeClient.ExtensionsV1beta1().ReplicaSets(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&extensions.ReplicaSet{},
		c.syncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if resource, ok := obj.(*extensions.ReplicaSet); ok {
					log.Infof("ReplicaSet %s@%s added", resource.Name, resource.Namespace)

					restic, err := util.FindRestic(c.StashClient, resource.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching Restic for ReplicaSet %s@%s.", resource.Name, resource.Namespace)
						return
					}
					if restic == nil {
						log.Errorf("No Restic found for ReplicaSet %s@%s.", resource.Name, resource.Namespace)
						return
					}
					c.EnsureReplicaSetSidecar(resource, restic)
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (c *Controller) EnsureReplicaSetSidecar(resource *extensions.ReplicaSet, restic *sapi.Restic) error {
	if name := util.GetString(resource.Annotations, sapi.ConfigName); name != "" {
		log.Infof("Restic sidecar already exists for ReplicaSet %s@%s.", resource.Name, resource.Namespace)
		return nil
	}

	resource.Spec.Template.Spec.Containers = append(resource.Spec.Template.Spec.Containers, util.GetSidecarContainer(restic, c.SidecarImageTag, resource.Name, false))
	resource.Spec.Template.Spec.Volumes = util.AddScratchVolume(resource.Spec.Template.Spec.Volumes)
	resource.Spec.Template.Spec.Volumes = util.AddDownwardVolume(resource.Spec.Template.Spec.Volumes)
	if restic.Spec.Backend.Local != nil {
		resource.Spec.Template.Spec.Volumes = append(resource.Spec.Template.Spec.Volumes, restic.Spec.Backend.Local.Volume)
	}
	if resource.Annotations == nil {
		resource.Annotations = make(map[string]string)
	}
	resource.Annotations[sapi.ConfigName] = restic.Name
	resource.Annotations[sapi.VersionTag] = c.SidecarImageTag

	resource, err := c.KubeClient.ExtensionsV1beta1().ReplicaSets(resource.Namespace).Update(resource)
	if kerr.IsNotFound(err) {
		return nil
	} else if err != nil {
		sidecarFailedToAdd()
		log.Errorf("Failed to add sidecar for ReplicaSet %s@%s.", resource.Name, resource.Namespace)
		return err
	}
	sidecarSuccessfullyAdd()
	return util.RestartPods(c.KubeClient, resource.Namespace, resource.Spec.Selector)
}

func (c *Controller) EnsureReplicaSetSidecarDeleted(resource *extensions.ReplicaSet, restic *sapi.Restic) error {
	resource.Spec.Template.Spec.Containers = util.RemoveContainer(resource.Spec.Template.Spec.Containers, ContainerName)
	resource.Spec.Template.Spec.Volumes = util.RemoveVolume(resource.Spec.Template.Spec.Volumes, util.ScratchDirVolumeName)
	resource.Spec.Template.Spec.Volumes = util.RemoveVolume(resource.Spec.Template.Spec.Volumes, util.PodinfoVolumeName)
	if restic.Spec.Backend.Local != nil {
		resource.Spec.Template.Spec.Volumes = util.RemoveVolume(resource.Spec.Template.Spec.Volumes, restic.Spec.Backend.Local.Volume.Name)
	}
	if resource.Annotations != nil {
		delete(resource.Annotations, sapi.ConfigName)
		delete(resource.Annotations, sapi.VersionTag)
	}

	resource, err := c.KubeClient.ExtensionsV1beta1().ReplicaSets(resource.Namespace).Update(resource)
	if kerr.IsNotFound(err) {
		return nil
	} else if err != nil {
		sidecarFailedToAdd()
		log.Errorf("Failed to add sidecar for ReplicaSet %s@%s.", resource.Name, resource.Namespace)
		return err
	}
	sidecarSuccessfullyDeleted()
	util.RestartPods(c.KubeClient, resource.Namespace, resource.Spec.Selector)
	return nil
}
