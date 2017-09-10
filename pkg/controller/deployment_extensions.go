package controller

import (
	"errors"
	"fmt"
	"reflect"

	acrt "github.com/appscode/go/runtime"
	corekutil "github.com/appscode/kutil/core/v1"
	kutil "github.com/appscode/kutil/extensions/v1beta1"
	"github.com/appscode/log"
	sapi "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Controller) WatchDeploymentExtensions() {
	if !util.IsPreferredAPIResource(c.kubeClient, extensions.SchemeGroupVersion.String(), "Deployment") {
		log.Warningf("Skipping watching non-preferred GroupVersion:%s Kind:%s", extensions.SchemeGroupVersion.String(), "Deployment")
		return
	}

	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.kubeClient.ExtensionsV1beta1().Deployments(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.kubeClient.ExtensionsV1beta1().Deployments(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&extensions.Deployment{},
		c.resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if resource, ok := obj.(*extensions.Deployment); ok {
					log.Infof("Deployment %s@%s added", resource.Name, resource.Namespace)

					restic, err := util.FindRestic(c.stashClient, resource.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching Restic for Deployment %s@%s.", resource.Name, resource.Namespace)
						return
					}
					if restic == nil {
						log.Errorf("No Restic found for Deployment %s@%s.", resource.Name, resource.Namespace)
						return
					}
					c.EnsureDeploymentExtensionSidecar(resource, nil, restic)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldObj, ok := old.(*extensions.Deployment)
				if !ok {
					log.Errorln(errors.New("Invalid Deployment object"))
					return
				}
				newObj, ok := new.(*extensions.Deployment)
				if !ok {
					log.Errorln(errors.New("Invalid Deployment object"))
					return
				}
				if !reflect.DeepEqual(oldObj.Labels, newObj.Labels) {
					oldRestic, err := util.FindRestic(c.stashClient, oldObj.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching Restic for Deployment %s@%s.", oldObj.Name, oldObj.Namespace)
						return
					}
					newRestic, err := util.FindRestic(c.stashClient, newObj.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching Restic for Deployment %s@%s.", newObj.Name, newObj.Namespace)
						return
					}
					if util.ResticEqual(oldRestic, newRestic) {
						return
					}
					if newRestic != nil {
						c.EnsureDeploymentExtensionSidecar(newObj, oldRestic, newRestic)
					} else if oldRestic != nil {
						c.EnsureDeploymentExtensionSidecarDeleted(newObj, oldRestic)
					}
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (c *Controller) EnsureDeploymentExtensionSidecar(resource *extensions.Deployment, old, new *sapi.Restic) (err error) {
	if new.Spec.Backend.StorageSecretName == "" {
		err = fmt.Errorf("Missing repository secret name for Restic %s@%s.", new.Name, new.Namespace)
		return
	}
	_, err = c.kubeClient.CoreV1().Secrets(resource.Namespace).Get(new.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if name := util.GetString(resource.Annotations, sapi.ConfigName); name != "" && name != new.Name {
		log.Infof("Restic %s sidecar already added for Deployment %s@%s.", name, resource.Name, resource.Namespace)
		return nil
	}

	_, err = kutil.PatchDeployment(c.kubeClient, resource, func(obj *extensions.Deployment) *extensions.Deployment {
		obj.Spec.Template.Spec.Containers = corekutil.UpsertContainer(obj.Spec.Template.Spec.Containers, util.CreateSidecarContainer(new, c.SidecarImageTag, "Deployment/"+obj.Name))
		obj.Spec.Template.Spec.Volumes = util.UpsertScratchVolume(obj.Spec.Template.Spec.Volumes)
		obj.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(obj.Spec.Template.Spec.Volumes)
		obj.Spec.Template.Spec.Volumes = util.MergeLocalVolume(obj.Spec.Template.Spec.Volumes, old, new)

		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}
		obj.Annotations[sapi.ConfigName] = new.Name
		obj.Annotations[sapi.VersionTag] = c.SidecarImageTag
		return obj
	})
	if err != nil {
		return
	}

	err = kutil.WaitUntilDeploymentReady(c.kubeClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarAdded(c.kubeClient, resource.Namespace, resource.Spec.Selector)
	return err
}

func (c *Controller) EnsureDeploymentExtensionSidecarDeleted(resource *extensions.Deployment, restic *sapi.Restic) (err error) {
	if name := util.GetString(resource.Annotations, sapi.ConfigName); name == "" {
		log.Infof("Restic sidecar already removed for Deployment %s@%s.", resource.Name, resource.Namespace)
		return nil
	}

	_, err = kutil.PatchDeployment(c.kubeClient, resource, func(obj *extensions.Deployment) *extensions.Deployment {
		obj.Spec.Template.Spec.Containers = corekutil.EnsureContainerDeleted(obj.Spec.Template.Spec.Containers, util.StashContainer)
		obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.ScratchDirVolumeName)
		obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.PodinfoVolumeName)
		if restic.Spec.Backend.Local != nil {
			obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.LocalVolumeName)
		}
		if obj.Annotations != nil {
			delete(obj.Annotations, sapi.ConfigName)
			delete(obj.Annotations, sapi.VersionTag)
		}
		return obj
	})
	if err != nil {
		return
	}

	err = kutil.WaitUntilDeploymentReady(c.kubeClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarRemoved(c.kubeClient, resource.Namespace, resource.Spec.Selector)
	return err
}
