package framework

import (
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kutil "kmodules.xyz/client-go"
	apps_util "kmodules.xyz/client-go/apps/v1"
)

func (fi *Invocation) ReplicaSet(pvcName string) apps.ReplicaSet {
	labels := map[string]string{
		"app":  fi.app,
		"kind": "replicaset",
	}
	return apps.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels:    labels,
		},
		Spec: apps.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas: types.Int32P(1),
			Template: fi.PodTemplate(labels, pvcName),
		},
	}
}

func (f *Framework) CreateReplicaSet(obj apps.ReplicaSet) (*apps.ReplicaSet, error) {
	return f.KubeClient.AppsV1().ReplicaSets(obj.Namespace).Create(&obj)
}

func (f *Framework) DeleteReplicaSet(meta metav1.ObjectMeta) error {
	err := f.KubeClient.AppsV1().ReplicaSets(meta.Namespace).Delete(meta.Name, deleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyReplicaSet(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.ReplicaSet {
		obj, err := f.KubeClient.AppsV1().ReplicaSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

func (f *Invocation) WaitUntilRSReadyWithSidecar(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := f.KubeClient.AppsV1().ReplicaSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if obj.Status.Replicas == obj.Status.ReadyReplicas {
				pods, err := f.GetAllPods(obj.ObjectMeta)
				if err != nil {
					return false, err
				}

				for i := range pods {
					hasSidecar := false
					for _, c := range pods[i].Spec.Containers {
						if c.Name == util.StashContainer {
							hasSidecar = true
						}
					}
					if !hasSidecar {
						return false, nil
					}
				}
				return true, nil
			}
			return false, nil
		}
		return false, nil
	})
}

func (f *Invocation) WaitUntilRSReadyWithInitContainer(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := f.KubeClient.AppsV1().ReplicaSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if obj.Status.Replicas == obj.Status.ReadyReplicas {
				pods, err := f.GetAllPods(obj.ObjectMeta)
				if err != nil {
					return false, err
				}

				for i := range pods {
					hasInitContainer := false
					for _, c := range pods[i].Spec.InitContainers {
						if c.Name == util.StashInitContainer {
							hasInitContainer = true
						}
					}
					if !hasInitContainer {
						return false, nil
					}
				}
				return true, nil
			}
			return false, nil
		}
		return false, nil
	})
}

func (f *Invocation) DeployReplicaSet(name string, replica int32) *apps.ReplicaSet {
	// Create PVC for ReplicaSet
	pvc := f.CreateNewPVC(name)
	// Generate ReplicaSet definition
	rs := f.ReplicaSet(pvc.Name)
	rs.Spec.Replicas = &replica
	rs.Name = name

	By("Deploying ReplicaSet: " + rs.Name)
	createdRS, err := f.CreateReplicaSet(rs)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(createdRS)

	By("Waiting for ReplicaSet to be ready")
	err = apps_util.WaitUntilReplicaSetReady(f.KubeClient, createdRS.ObjectMeta)
	Expect(err).NotTo(HaveOccurred())
	// check that we can execute command to the pod.
	// this is necessary because we will exec into the pods and create sample data
	f.EventuallyPodAccessible(createdRS.ObjectMeta).Should(BeTrue())

	return createdRS
}
