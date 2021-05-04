/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backup

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	api "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stash_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1/util"
	stashinformers "stash.appscode.dev/apimachinery/client/informers/externalversions"
	stash_listers "stash.appscode.dev/apimachinery/client/listers/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/stash/pkg/cli"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/robfig/cron/v3"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
	"k8s.io/klog/v2"
	core_util "kmodules.xyz/client-go/core/v1"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
	"kmodules.xyz/client-go/tools/queue"
)

type Options struct {
	Workload         api.LocalTypedReference
	Namespace        string
	ResticName       string
	ScratchDir       string
	PushgatewayURL   string
	NodeName         string
	PodName          string
	SmartPrefix      string
	SnapshotHostname string
	PodLabelsPath    string
	QPS              float64
	Burst            int
	ResyncPeriod     time.Duration
	MaxNumRequeues   int
	RunViaCron       bool
	DockerRegistry   string // image registry for the sidecar, init-container,and job etc.
	StashImage       string // image for the sidecar,init-container,and jobs etc.
	ImageTag         string // image tag for the sidecar,init-container, and jobs etc.
	NumThreads       int
}

type Controller struct {
	k8sClient   kubernetes.Interface
	stashClient cs.Interface
	opt         Options
	locked      chan struct{}
	resticCLI   *cli.ResticWrapper
	cron        *cron.Cron
	recorder    record.EventRecorder

	stashInformerFactory stashinformers.SharedInformerFactory

	// Restic
	rQueue    *queue.Worker
	rInformer cache.SharedIndexInformer
	rLister   stash_listers.ResticLister
}

const (
	CheckRole            = "stash-check"
	BackupEventComponent = "stash-backup"
)

func New(k8sClient kubernetes.Interface, stashClient cs.Interface, opt Options) *Controller {
	return &Controller{
		k8sClient:   k8sClient,
		stashClient: stashClient,
		opt:         opt,
		cron:        cron.New(),
		locked:      make(chan struct{}, 1),
		resticCLI:   cli.New(opt.ScratchDir, true, opt.SnapshotHostname),
		recorder:    eventer.NewEventRecorder(k8sClient, BackupEventComponent),
		stashInformerFactory: stashinformers.NewFilteredSharedInformerFactory(
			stashClient,
			opt.ResyncPeriod,
			opt.Namespace,
			// BUG!!! In 1.8.x, field selectors can't be used with CRDs
			// ref: https://github.com/appscode/voyager/issues/889
			//func(options *metav1.ListOptions) {
			//	options.FieldSelector = fields.OneTermEqualSelector("metadata.name", opt.ResticName).String()
			//},
			nil,
		),
	}
}

func (c *Controller) Backup() error {
	restic, repository, err := c.setup()
	if err != nil {
		err = fmt.Errorf("failed to setup backup. Error: %v", err)
		if restic != nil {
			ref, rerr := reference.GetReference(scheme.Scheme, restic)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					BackupEventComponent,
					ref,
					core.EventTypeWarning,
					eventer.EventReasonFailedSetup,
					err.Error(),
				)
			} else {
				klog.Errorf("Failed to write event on %s %s. Reason: %s", restic.Kind, restic.Name, rerr)
			}
		}
		return err
	}

	if err = c.runResticBackup(restic, repository); err != nil {
		return fmt.Errorf("failed to run backup, reason: %s", err)
	}

	// create check job
	image := docker.Docker{
		Registry: c.opt.DockerRegistry,
		Image:    c.opt.StashImage,
		Tag:      c.opt.ImageTag,
	}

	job := util.NewCheckJob(restic, c.opt.SnapshotHostname, c.opt.SmartPrefix, image)

	// check if check job exists
	if _, err = c.k8sClient.BatchV1().Jobs(restic.Namespace).Get(context.TODO(), job.Name, metav1.GetOptions{}); err != nil && !errors.IsNotFound(err) {
		ref, rerr := reference.GetReference(scheme.Scheme, repository)
		if rerr == nil {
			eventer.CreateEventWithLog(
				c.k8sClient,
				BackupEventComponent,
				ref,
				core.EventTypeWarning,
				eventer.EventReasonFailedCronJob,
				err.Error(),
			)
		} else {
			klog.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
		}
		return err
	}
	if errors.IsNotFound(err) {
		job.Spec.Template.Spec.ServiceAccountName = job.Name

		if job, err = c.k8sClient.BatchV1().Jobs(restic.Namespace).Create(context.TODO(), job, metav1.CreateOptions{}); err != nil {
			err = fmt.Errorf("failed to get check job, reason: %s", err)
			ref, rerr := reference.GetReference(scheme.Scheme, repository)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					BackupEventComponent,
					ref,
					core.EventTypeWarning,
					eventer.EventReasonFailedCronJob,
					err.Error(),
				)
			} else {
				klog.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
			}
			return err
		}

		// create service-account and role-binding
		owner := metav1.NewControllerRef(job, batchv1.SchemeGroupVersion.WithKind(apis.KindJob))
		if err = c.ensureCheckRBAC(job.Namespace, owner); err != nil {
			return fmt.Errorf("error ensuring rbac for check job %s, reason: %s", job.Name, err)
		}

		klog.Infoln("Created check job:", job.Name)
		ref, rerr := reference.GetReference(scheme.Scheme, repository)
		if rerr == nil {
			eventer.CreateEventWithLog(
				c.k8sClient,
				BackupEventComponent,
				ref,
				core.EventTypeNormal,
				eventer.EventReasonCheckJobCreated,
				fmt.Sprintf("Created check job: %s", job.Name),
			)
		} else {
			klog.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
		}
	} else {
		klog.Infoln("Check job already exists, skipping creation:", job.Name)
		ref, rerr := reference.GetReference(scheme.Scheme, repository)
		if rerr == nil {
			eventer.CreateEventWithLog(
				c.k8sClient,
				BackupEventComponent,
				ref,
				core.EventTypeNormal,
				eventer.EventReasonCheckJobCreated,
				fmt.Sprintf("Check job already exists, skipping creation: %s", job.Name),
			)
		} else {
			klog.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
		}
	}
	return nil
}

// Init and/or connect to repo
func (c *Controller) setup() (*api.Restic, *api.Repository, error) {
	// setup scratch-dir
	if err := os.MkdirAll(c.opt.ScratchDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create scratch dir: %s", err)
	}
	if err := ioutil.WriteFile(c.opt.ScratchDir+"/.stash", []byte("test"), 0644); err != nil {
		return nil, nil, fmt.Errorf("no write access in scratch dir: %s", err)
	}

	// check restic
	restic, err := c.stashClient.StashV1alpha1().Restics(c.opt.Namespace).Get(context.TODO(), c.opt.ResticName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	klog.Infof("Found restic %s", restic.Name)
	if err := restic.IsValid(); err != nil {
		return restic, nil, err
	}
	secret, err := c.k8sClient.CoreV1().Secrets(restic.Namespace).Get(context.TODO(), restic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return restic, nil, err
	}
	klog.Infof("Found repository secret %s", secret.Name)

	// setup restic-cli
	prefix := ""
	if prefix, err = c.resticCLI.SetupEnv(restic.Spec.Backend, secret, c.opt.SmartPrefix); err != nil {
		return restic, nil, err
	}
	if err = c.resticCLI.InitRepositoryIfAbsent(); err != nil {
		return restic, nil, err
	}
	repository, err := c.createRepositoryCrdIfNotExist(restic, prefix)
	if err != nil {
		return restic, nil, err
	}
	return restic, repository, nil
}

func (c *Controller) runResticBackup(restic *api.Restic, repository *api.Repository) (err error) {
	if restic.Spec.Paused {
		klog.Infoln("skipped logging since restic is paused.")
		return nil
	}
	startTime := metav1.Now()
	var (
		restic_session_success = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "restic",
			Subsystem: "session",
			Name:      "success",
			Help:      "Indicates if session was successfully completed",
		})
		restic_session_fail = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "restic",
			Subsystem: "session",
			Name:      "fail",
			Help:      "Indicates if session failed",
		})
		restic_session_duration_seconds_total = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "restic",
			Subsystem: "session",
			Name:      "duration_seconds_total",
			Help:      "Total seconds taken to complete restic session",
		})
		restic_session_duration_seconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "restic",
			Subsystem: "session",
			Name:      "duration_seconds",
			Help:      "Total seconds taken to complete restic session",
		}, []string{"filegroup", "op"})
	)

	defer func() {
		endTime := metav1.Now()
		if c.opt.PushgatewayURL != "" {
			if err != nil {
				restic_session_success.Set(0)
				restic_session_fail.Set(1)
			} else {
				restic_session_success.Set(1)
				restic_session_fail.Set(0)
			}
			restic_session_duration_seconds_total.Set(endTime.Sub(startTime.Time).Seconds())

			pusher := push.New(c.opt.PushgatewayURL, c.JobName(restic))
			registry := prometheus.NewRegistry()
			registry.MustRegister(
				restic_session_success,
				restic_session_fail,
				restic_session_duration_seconds_total,
				restic_session_duration_seconds,
			)
			err := pusher.Gatherer(registry).Push()
			if err != nil {
				klog.Errorln(err)
			}
		}
		if err == nil {
			_, err2 := stash_util.UpdateRepositoryStatus(
				context.TODO(),
				c.stashClient.StashV1alpha1(),
				repository.ObjectMeta,
				func(in *api.RepositoryStatus) (types.UID, *api.RepositoryStatus) {
					in.BackupCount++
					in.LastBackupTime = &startTime
					if in.FirstBackupTime == nil {
						in.FirstBackupTime = &startTime
					}
					in.LastBackupDuration = endTime.Sub(startTime.Time).String()
					return repository.UID, in
				},
				metav1.UpdateOptions{},
			)
			if err2 != nil {
				klog.Errorln(err2)
			}
		}
	}()

	for _, fg := range restic.Spec.FileGroups {
		backupOpMetric := restic_session_duration_seconds.WithLabelValues(sanitizeLabelValue(fg.Path), "backup")
		err = c.measure(c.resticCLI.Backup, restic, fg, backupOpMetric)
		if err != nil {
			klog.Errorf("Backup failed for Repository %s/%s, reason: %s", repository.Namespace, repository.Name, err)
			ref, rerr := reference.GetReference(scheme.Scheme, repository)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					BackupEventComponent,
					ref,
					core.EventTypeWarning,
					eventer.EventReasonFailedToBackup,
					fmt.Sprintf("Backup failed, reason: %s", err),
				)
			} else {
				klog.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
			}
			return
		} else {
			hostname, _ := os.Hostname()
			ref, rerr := reference.GetReference(scheme.Scheme, repository)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					BackupEventComponent,
					ref,
					core.EventTypeNormal,
					eventer.EventReasonSuccessfulBackup,
					fmt.Sprintf("Backed up pod: %s, path: %s", hostname, fg.Path),
				)
			} else {
				klog.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
			}
		}

		forgetOpMetric := restic_session_duration_seconds.WithLabelValues(sanitizeLabelValue(fg.Path), "forget")
		err = c.measure(c.resticCLI.Forget, restic, fg, forgetOpMetric)
		if err != nil {
			klog.Errorf("Failed to forget old snapshots for Repository %s/%s, reason: %s", repository.Namespace, repository.Name, err)
			ref, rerr := reference.GetReference(scheme.Scheme, repository)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					BackupEventComponent,
					ref,
					core.EventTypeWarning,
					eventer.EventReasonFailedToRetention,
					fmt.Sprintf("Failed to forget old snapshots, reason: %s", err),
				)
			} else {
				klog.Errorf("Failed to write event on %s %s. Reason: %s", repository.Kind, repository.Name, rerr)
			}
			return
		}
	}
	return
}

func (c *Controller) measure(f func(*api.Restic, api.FileGroup) error, restic *api.Restic, fg api.FileGroup, g prometheus.Gauge) (err error) {
	startTime := time.Now()
	defer func() {
		g.Set(time.Since(startTime).Seconds())
	}()
	err = f(restic, fg)
	return
}

// use sidecar-cluster-role, service-account and role-binding name same as job name
// set job as owner of service-account and role-binding
func (c *Controller) ensureCheckRBAC(namespace string, owner *metav1.OwnerReference) error {
	// ensure service account
	meta := metav1.ObjectMeta{
		Name:      owner.Name,
		Namespace: namespace,
	}
	_, _, err := core_util.CreateOrPatchServiceAccount(
		context.TODO(),
		c.k8sClient,
		meta,
		func(in *core.ServiceAccount) *core.ServiceAccount {
			core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

			if in.Labels == nil {
				in.Labels = map[string]string{}
			}
			in.Labels[apis.LabelApp] = apis.AppLabelStash
			return in
		},
		metav1.PatchOptions{},
	)
	if err != nil {
		return err
	}

	// ensure role binding
	_, _, err = rbac_util.CreateOrPatchRoleBinding(
		context.TODO(),
		c.k8sClient,
		meta,
		func(in *rbac.RoleBinding) *rbac.RoleBinding {
			core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

			if in.Labels == nil {
				in.Labels = map[string]string{}
			}
			in.Labels[apis.LabelApp] = apis.AppLabelStash

			in.RoleRef = rbac.RoleRef{
				APIGroup: rbac.GroupName,
				Kind:     apis.KindClusterRole,
				Name:     apis.StashSidecarClusterRole,
			}
			in.Subjects = []rbac.Subject{
				{
					Kind:      rbac.ServiceAccountKind,
					Name:      meta.Name,
					Namespace: meta.Namespace,
				},
			}
			return in
		},
		metav1.PatchOptions{},
	)
	return err
}
