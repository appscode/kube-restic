package framework

import (
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kutil "kmodules.xyz/client-go"
)

func (fi *Invocation) ReplicationController(pvcName string) core.ReplicationController {
	labels := map[string]string{
		"app":  fi.app,
		"kind": "replicationcontroller",
	}
	podTemplate := fi.PodTemplate(labels, pvcName)
	return core.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels: map[string]string{
				"app": fi.app,
			},
		},
		Spec: core.ReplicationControllerSpec{
			Replicas: types.Int32P(1),
			Template: &podTemplate,
		},
	}
}

func (f *Framework) CreateReplicationController(obj core.ReplicationController) (*core.ReplicationController, error) {
	return f.KubeClient.CoreV1().ReplicationControllers(obj.Namespace).Create(&obj)
}

func (f *Framework) DeleteReplicationController(meta metav1.ObjectMeta) error {
	err := f.KubeClient.CoreV1().ReplicationControllers(meta.Namespace).Delete(meta.Name, deleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyReplicationController(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *core.ReplicationController {
		obj, err := f.KubeClient.CoreV1().ReplicationControllers(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

func (f *Invocation) WaitUntilRCReadyWithSidecar(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := f.KubeClient.CoreV1().ReplicationControllers(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
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

func (f *Invocation) WaitUntilRCReadyWithInitContainer(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := f.KubeClient.CoreV1().ReplicationControllers(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
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

func (f *Invocation) DeployReplicationController(name string, replica int32) *core.ReplicationController {
	// Create PVC for ReplicationController
	pvc := f.CreateNewPVC(name)
	// Generate ReplicationController definition
	rc := f.ReplicationController(pvc.Name)
	rc.Spec.Replicas = &replica
	rc.Name = name

	By("Deploying ReplicationController: " + rc.Name)
	createdRC, err := f.CreateReplicationController(rc)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(createdRC)

	By("Waiting for ReplicationController to be ready")
	err = util.WaitUntilRCReady(f.KubeClient, createdRC.ObjectMeta)
	Expect(err).NotTo(HaveOccurred())
	// check that we can execute command to the pod.
	// this is necessary because we will exec into the pods and create sample data
	f.EventuallyPodAccessible(createdRC.ObjectMeta).Should(BeTrue())

	return createdRC
}
