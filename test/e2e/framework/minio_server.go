package framework

import (
	"fmt"
	"net"
	"path/filepath"

	"github.com/appscode/go/types"
	. "github.com/onsi/gomega"
	"gomodules.xyz/cert"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apps_util "kmodules.xyz/client-go/apps/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

const (
	MINIO_PUBLIC_CRT_NAME  = "public.crt"
	MINIO_PRIVATE_KEY_NAME = "private.key"

	MINIO_ACCESS_KEY_ID     = "not@id"
	MINIO_SECRET_ACCESS_KEY = "not@secret"

	MINIO_CERTS_MOUNTPATH = "/root/.minio/certs"
	StandardStorageClass  = "standard"

	MinioServer         = "minio-server"
	MinioServerSecret   = "minio-server-secret"
	MinioPVCStorage     = "minio-pvc-storage"
	MinioNodePortServic = "minio-nodeport-service"
)

var (
	mcred   core.Secret
	mpvc    core.PersistentVolumeClaim
	mdeploy apps.Deployment
	msrvc   core.Service
)

func (f *Framework) CreateMinioServer(tls bool, ips []net.IP) (string, error) {
	//creating secret for minio server
	mcred = f.SecretForMinioServer(ips)
	err := f.CreateSecret(mcred)
	if err != nil {
		return "", err
	}

	//creating deployment for minio server
	mdeploy = f.DeploymentForMinioServer()
	if !tls { // if tls not enabled then don't mount secret for cacerts
		mdeploy.Spec.Template.Spec.Containers = f.RemoveSecretVolumeMount(mdeploy.Spec.Template.Spec.Containers)
	}
	err = f.CreateDeploymentForMinioServer(mdeploy)
	if err != nil {
		return "", err
	}

	//creating pvc for minio server
	mpvc = f.PVCForMinioServer()
	err = f.CreatePersistentVolumeClaimForMinioServer(mpvc)
	if err != nil {
		return "", nil
	}

	//creating service for minio server
	msrvc = f.ServiceForMinioServer()
	_, err = f.CreateServiceForMinioServer(msrvc)
	if err != nil {
		return "", err
	}
	err = apps_util.WaitUntilDeploymentReady(f.KubeClient, mdeploy.ObjectMeta)
	if err != nil {
		return "", err
	}
	return f.MinioServiceAddres(), nil
}

func (f *Framework) SecretForMinioServer(ips []net.IP) core.Secret {
	crt, key, err := f.CertStore.NewServerCertPairBytes(f.MinioServerSANs(ips))
	Expect(err).NotTo(HaveOccurred())

	return core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MinioServerSecret,
			Namespace: f.namespace,
		},
		Data: map[string][]byte{
			MINIO_PUBLIC_CRT_NAME:  []byte(string(crt) + "\n" + string(f.CertStore.CACertBytes())),
			MINIO_PRIVATE_KEY_NAME: key,
		},
	}
}

func (f *Framework) PVCForMinioServer() core.PersistentVolumeClaim {
	return core.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MinioPVCStorage,
			Namespace: f.namespace,
			Labels: map[string]string{
				// this label will be used to mount this pvc as volume in minio server container
				"app": "minio-storage-claim",
			},
		},
		Spec: core.PersistentVolumeClaimSpec{
			AccessModes: []core.PersistentVolumeAccessMode{
				core.ReadWriteOnce,
			},
			Resources: core.ResourceRequirements{
				Requests: core.ResourceList{
					core.ResourceName(core.ResourceStorage): resource.MustParse("2Gi"),
				},
			},
			StorageClassName: types.StringP(StandardStorageClass),
		},
	}
}

func (f *Framework) CreatePersistentVolumeClaimForMinioServer(obj core.PersistentVolumeClaim) error {
	_, err := f.KubeClient.CoreV1().PersistentVolumeClaims(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeploymentForMinioServer() apps.Deployment {
	labels := map[string]string{
		"app": f.namespace + "minio-server",
	}

	return apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MinioServer,
			Namespace: f.namespace,
		},
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},

			Strategy: apps.DeploymentStrategy{
				Type: apps.RecreateDeploymentStrategyType,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					// minio service will select this pod using this label.
					Labels: labels,
				},
				Spec: core.PodSpec{
					// this volumes will be mounted on minio server container
					Volumes: []core.Volume{
						{
							Name: "minio-storage",
							VolumeSource: core.VolumeSource{
								PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
									ClaimName: MinioPVCStorage,
								},
							},
						},
						{
							Name: "minio-certs",
							VolumeSource: core.VolumeSource{
								Secret: &core.SecretVolumeSource{
									SecretName: MinioServerSecret,
									Items: []core.KeyToPath{
										{
											Key:  MINIO_PUBLIC_CRT_NAME,
											Path: MINIO_PUBLIC_CRT_NAME,
										},
										{
											Key:  MINIO_PRIVATE_KEY_NAME,
											Path: MINIO_PRIVATE_KEY_NAME,
										},
										{
											Key:  MINIO_PUBLIC_CRT_NAME,
											Path: filepath.Join("CAs", MINIO_PUBLIC_CRT_NAME),
										},
									},
								},
							},
						},
					},
					// run this containers in minio server pod
					Containers: []core.Container{
						{
							Name:  "minio-server",
							Image: "minio/minio",
							Args: []string{
								"server",
								"--address",
								":443",
								"/storage",
							},
							Env: []core.EnvVar{
								{
									Name:  "MINIO_ACCESS_KEY",
									Value: MINIO_ACCESS_KEY_ID,
								},
								{
									Name:  "MINIO_SECRET_KEY",
									Value: MINIO_SECRET_ACCESS_KEY,
								},
							},
							Ports: []core.ContainerPort{
								{
									ContainerPort: int32(443),
									HostPort:      int32(443),
								},
							},
							VolumeMounts: []core.VolumeMount{
								{
									Name:      "minio-storage",
									MountPath: "/storage",
								},
								{
									Name:      "minio-certs",
									MountPath: MINIO_CERTS_MOUNTPATH,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (f *Framework) RemoveSecretVolumeMount(containers []core.Container) []core.Container {
	resp := make([]core.Container, 0)
	for _, c := range containers {
		if c.Name == "minio-server" {
			c.VolumeMounts = core_util.EnsureVolumeMountDeleted(c.VolumeMounts, "minio-secret")
		}
		resp = append(resp, c)
	}
	return resp
}

func (f *Framework) CreateDeploymentForMinioServer(obj apps.Deployment) error {
	_, err := f.KubeClient.AppsV1().Deployments(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) ServiceForMinioServer() core.Service {
	return core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MinioNodePortServic,
			Namespace: f.namespace,
		},
		Spec: core.ServiceSpec{
			Type: core.ServiceTypeLoadBalancer,
			Ports: []core.ServicePort{
				{
					Port:       int32(443),
					TargetPort: intstr.FromInt(443),
					Protocol:   core.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": f.namespace + "minio-server",
			},
		},
	}
}

func (f *Framework) CreateServiceForMinioServer(obj core.Service) (*core.Service, error) {
	return f.KubeClient.CoreV1().Services(obj.Namespace).Create(&obj)
}

func (f *Framework) DeleteMinioServer() error {
	if err := f.DeleteSecretForMinioServer(mcred.ObjectMeta); err != nil {
		return err
	}
	if err := f.DeletePVCForMinioServer(mpvc.ObjectMeta); err != nil {
		return err
	}
	if err := f.DeleteDeploymentForMinioServer(mdeploy.ObjectMeta); err != nil {
		return err
	}
	if err := f.DeleteServiceForMinioServer(msrvc.ObjectMeta); err != nil {
		return err
	}
	return nil
}
func (f *Framework) DeleteSecretForMinioServer(meta metav1.ObjectMeta) error {
	err := f.KubeClient.CoreV1().Secrets(meta.Namespace).Delete(meta.Name, deleteInForeground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) DeletePVCForMinioServer(meta metav1.ObjectMeta) error {
	err := f.KubeClient.CoreV1().PersistentVolumeClaims(meta.Namespace).Delete(meta.Name, deleteInForeground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) DeleteDeploymentForMinioServer(meta metav1.ObjectMeta) error {
	err := f.KubeClient.AppsV1().Deployments(meta.Namespace).Delete(meta.Name, deleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) DeleteServiceForMinioServer(meta metav1.ObjectMeta) error {
	err := f.KubeClient.CoreV1().Services(meta.Namespace).Delete(meta.Name, deleteInForeground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) MinioServerSANs(ips []net.IP) cert.AltNames {
	altNames := cert.AltNames{
		DNSNames: []string{f.MinioServiceAddres()},
		IPs:      ips,
	}
	return altNames
}

func (f *Framework) MinioServiceAddres() string {
	return fmt.Sprintf(MinioNodePortServic+".%s.svc", f.namespace)

}
