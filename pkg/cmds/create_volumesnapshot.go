package cmds

import (
	"fmt"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	vs "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	vs_cs "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/meta"
	workload_api "kmodules.xyz/webhook-runtime/apis/workload/v1"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"
)

var (
	pvcList = make([]string, 0)
)

type VSoption struct {
	name           string
	namespace      string
	kubeClient     kubernetes.Interface
	stashClient    cs.Interface
	snapshotClient vs_cs.Interface
}

func NewCmdCreateVolumeSnapshot() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = VSoption{
			namespace: meta.Namespace(),
		}
	)

	cmd := &cobra.Command{
		Use:               "create-vs",
		Short:             "Take snapshot of PersistentVolumeClaims",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}
			opt.kubeClient = kubernetes.NewForConfigOrDie(config)
			opt.stashClient = cs.NewForConfigOrDie(config)
			opt.snapshotClient = vs_cs.NewForConfigOrDie(config)

			err = opt.CreateVolumeSnapshot()
			if err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.name, "backupsession.name", "", "Set BackupSession Name")
	return cmd
}

func (opt *VSoption) CreateVolumeSnapshot() error {
	// Start clock to measure total session duration
	startTime := time.Now()
	backupSession, err := opt.stashClient.StashV1beta1().BackupSessions(opt.namespace).Get(opt.name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	backupConfiguration, err := opt.stashClient.StashV1beta1().BackupConfigurations(opt.namespace).Get(backupSession.Spec.BackupConfiguration.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if backupConfiguration == nil || backupConfiguration.Spec.Target == nil {
		return fmt.Errorf("BackupConfiguration or  backupConfiguration target is nil")
	}

	kind := backupConfiguration.Spec.Target.Ref.Kind
	name := backupConfiguration.Spec.Target.Ref.Name
	namespace := backupConfiguration.Namespace

	switch kind {
	case workload_api.KindDeployment:
		deployment, err := opt.kubeClient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		getPVCs(deployment.Spec.Template.Spec.Volumes)

	case workload_api.KindDaemonSet:
		daemon, err := opt.kubeClient.AppsV1().DaemonSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		getPVCs(daemon.Spec.Template.Spec.Volumes)

	case workload_api.KindReplicationController:
		rc, err := opt.kubeClient.CoreV1().ReplicationControllers(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		getPVCs(rc.Spec.Template.Spec.Volumes)

	case workload_api.KindReplicaSet:
		rs, err := opt.kubeClient.AppsV1().ReplicaSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		getPVCs(rs.Spec.Template.Spec.Volumes)

	case workload_api.KindStatefulSet:
		ss, err := opt.kubeClient.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		getPVCsForStatefulset(ss.Spec.VolumeClaimTemplates, ss)

	case apis.KindPersistentVolumeClaim:
		pvcList = []string{name}
	}

	objectMeta := []metav1.ObjectMeta{}

	for _, pvcName := range pvcList {
		volumeSnapshot := opt.getVolumeSnapshotDefinition(backupConfiguration, pvcName, backupSession.CreationTimestamp)
		vs, err := opt.snapshotClient.VolumesnapshotV1alpha1().VolumeSnapshots(namespace).Create(&volumeSnapshot)
		if err != nil {
			return err
		}
		objectMeta = append(objectMeta, vs.ObjectMeta)
	}

	for i, pvcName := range pvcList {
		err = util.WaitUntilVolumeSnapshotReady(opt.snapshotClient, objectMeta[i])
		if err != nil {
			return err
		}
		// Update Backup Session
		o := status.UpdateStatusOptions{
			KubeClient:    opt.kubeClient,
			StashClient:   opt.stashClient.(*cs.Clientset),
			Namespace:     opt.namespace,
			BackupSession: opt.name,
		}
		backupOutput := restic.BackupOutput{
			HostBackupStats: v1beta1.HostBackupStats{
				Hostname: pvcName,
				Phase:    v1beta1.HostBackupSucceeded,
			},
		}
		// Volume Snapshot complete. Read current time and calculate total backup duration.
		endTime := time.Now()
		backupOutput.HostBackupStats.Duration = endTime.Sub(startTime).String()

		err = o.UpdatePostBackupStatus(&backupOutput)
		if err != nil {
			return err
		}
	}
	return nil
}

func (opt *VSoption) getVolumeSnapshotDefinition(backupConfiguration *v1beta1.BackupConfiguration, pvcName string, creationTimestamp metav1.Time) (volumeSnapshot vs.VolumeSnapshot) {

	return vs.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%d-%02d-%02dt%02d-%02d-%02d-stash", pvcName, creationTimestamp.Year(), creationTimestamp.Month(), creationTimestamp.Day(), creationTimestamp.Hour(), creationTimestamp.Minute(), creationTimestamp.Second()),
			Namespace: backupConfiguration.Namespace,
		},
		Spec: vs.VolumeSnapshotSpec{
			VolumeSnapshotClassName: &backupConfiguration.Spec.Target.VolumeSnapshotClassName,
			Source: &corev1.TypedLocalObjectReference{
				Kind: apis.KindPersistentVolumeClaim,
				Name: pvcName,
			},
		},
	}

}

func getPVCs(volList []corev1.Volume) {
	pvcList = []string{}
	for _, list := range volList {
		if list.PersistentVolumeClaim != nil {
			pvcList = append(pvcList, list.PersistentVolumeClaim.ClaimName)
		}
	}
}

func getPVCsForStatefulset(volList []corev1.PersistentVolumeClaim, ss *appsv1.StatefulSet) {
	pvcList = []string{}
	for i := int32(0); i < types.Int32(ss.Spec.Replicas); i++ {
		podName := fmt.Sprintf("%v-%v", ss.Name, i)
		for _, list := range volList {
			pvcList = append(pvcList, fmt.Sprintf("%v-%v", list.Name, podName))
		}
	}

}
