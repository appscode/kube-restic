package framework

import (
	"database/sql"
	"fmt"
	"time"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"

	"github.com/appscode/go/sets"
	_ "github.com/go-sql-driver/mysql"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	apps_util "kmodules.xyz/client-go/apps/v1"
	appCatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
)

const (
	KeyUser     = "username"
	KeyPassword = "password"
	SuperUser   = "root"

	KeyMySQLRootPassword   = "MYSQL_ROOT_PASSWORD"
	MySQLServingPortName   = "mysql"
	MySQLContainerName     = "mysql"
	MySQLServingPortNumber = 3306
	MySQLBackupTask        = "mysql-backup-8.0.14"
	MySQLRestoreTask       = "mysql-restore-8.0.14"
	MySQLBackupFunction    = "mysql-backup-8.0.14"
	MySQLRestoreFunction   = "mysql-restore-8.0.14"
)

func (f *Invocation) MySQLCredentials() *core.Secret {
	return &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.app,
			Namespace: f.namespace,
		},
		Data: map[string][]byte{
			KeyUser:     []byte(SuperUser),
			KeyPassword: []byte(f.app),
		},
		Type: core.SecretTypeOpaque,
	}
}

func (f *Invocation) MySQLService() *core.Service {
	return &core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.app,
			Namespace: f.namespace,
		},
		Spec: core.ServiceSpec{
			Selector: map[string]string{
				"app": f.app,
			},
			Ports: []core.ServicePort{
				{
					Name: MySQLServingPortName,
					Port: MySQLServingPortNumber,
				},
			},
		},
	}
}

func (f *Invocation) MySQLPVC() *core.PersistentVolumeClaim {
	return &core.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.app,
			Namespace: f.namespace,
		},
		Spec: core.PersistentVolumeClaimSpec{
			AccessModes: []core.PersistentVolumeAccessMode{
				core.ReadWriteOnce,
			},
			Resources: core.ResourceRequirements{
				Requests: core.ResourceList{
					core.ResourceStorage: resource.MustParse("128Mi"),
				},
			},
		},
	}
}

func (f *Invocation) MySQLDeployment(cred *core.Secret, pvc *core.PersistentVolumeClaim) *apps.Deployment {
	label := map[string]string{
		"app": f.app,
	}
	return &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.app,
			Namespace: f.namespace,
		},
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: label,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: label,
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  MySQLContainerName,
							Image: "mysql:8",
							Env: []core.EnvVar{
								{
									Name: KeyMySQLRootPassword,
									ValueFrom: &core.EnvVarSource{
										SecretKeyRef: &core.SecretKeySelector{
											LocalObjectReference: core.LocalObjectReference{
												Name: cred.Name,
											},
											Key: KeyPassword,
										},
									},
								},
							},
							Ports: []core.ContainerPort{
								{
									Name:          MySQLServingPortName,
									ContainerPort: MySQLServingPortNumber,
								},
							},
							VolumeMounts: []core.VolumeMount{
								{
									Name:      pvc.Name,
									MountPath: "/var/lib/mysql",
								},
							},
						},
					},
					Volumes: []core.Volume{
						{
							Name: pvc.Name,
							VolumeSource: core.VolumeSource{
								PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
									ClaimName: pvc.Name,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (f *Invocation) MySQLAppBinding(cred *core.Secret, svc *core.Service) *appCatalog.AppBinding {
	return &appCatalog.AppBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.app,
			Namespace: f.namespace,
		},
		Spec: appCatalog.AppBindingSpec{
			Type:    "mysql",
			Version: "8.0.14",
			ClientConfig: appCatalog.ClientConfig{
				Service: &appCatalog.ServiceReference{
					Scheme: "mysql",
					Name:   svc.Name,
					Port:   MySQLServingPortNumber,
				},
			},
			Secret: &core.LocalObjectReference{
				Name: cred.Name,
			},
		},
	}
}

func (f *Invocation) DeployMySQLDatabase() (*apps.Deployment, *appCatalog.AppBinding, error) {
	By("Creating Secret for MySQL")
	cred := f.MySQLCredentials()
	_, err := f.CreateSecret(*cred)
	Expect(err).NotTo(HaveOccurred())

	By("Creating PVC for MySQL")
	pvc := f.MySQLPVC()
	_, err = f.CreatePersistentVolumeClaim(pvc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating Service for MySQL")
	svc := f.MySQLService()
	_, err = f.CreateService(*svc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating MySQL")
	dpl := f.MySQLDeployment(cred, pvc)
	dpl, err = f.CreateDeployment(*dpl)
	Expect(err).NotTo(HaveOccurred())

	By("Waiting for MySQL Deployment to be ready")
	err = apps_util.WaitUntilDeploymentReady(f.KubeClient, dpl.ObjectMeta)
	Expect(err).NotTo(HaveOccurred())

	By("Creating AppBinding for the MySQL")
	appBinding := f.MySQLAppBinding(cred, svc)
	appBinding, err = f.createAppBinding(appBinding)
	Expect(err).NotTo(HaveOccurred())

	f.AppendToCleanupList(appBinding, dpl, svc, pvc, cred)
	return dpl, appBinding, nil
}

func (f *Invocation) EventuallyConnectWithMySQLServer(db *sql.DB) error {
	return wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
		if err := db.Ping(); err != nil {
			return false, nil // don't return error. we need to retry.
		}
		return true, nil
	})
}

func (f *Invocation) createAppBinding(appBinding *appCatalog.AppBinding) (*appCatalog.AppBinding, error) {
	return f.catalogClient.AppcatalogV1alpha1().AppBindings(appBinding.Namespace).Create(appBinding)
}

func (f *Invocation) CreateTable(db *sql.DB, tableName string) error {
	stmnt, err := db.Prepare(fmt.Sprintf("CREATE TABLE %s ( ID int );", tableName))
	if err != nil {
		return err
	}
	defer stmnt.Close()

	_, err = stmnt.Exec()
	if err != nil {
		return err
	}
	return nil
}

func (f *Invocation) ListTables(db *sql.DB) (sets.String, error) {
	res, err := db.Query("SHOW TABLES IN mysql")
	if err != nil {
		return nil, err
	}
	defer res.Close()
	tables := sets.String{}
	var tableName string
	for res.Next() {
		err = res.Scan(&tableName)
		if err != nil {
			return nil, err
		}
		tables.Insert(tableName)
	}
	return tables, nil
}

func (f *Invocation) SetupDatabaseBackup(appBinding *appCatalog.AppBinding, repo *v1alpha1.Repository, transformFuncs ...func(bc *v1beta1.BackupConfiguration)) (*v1beta1.BackupConfiguration, error) {
	// Generate desired BackupConfiguration definition
	backupConfig := f.GetBackupConfiguration(repo.Name, func(bc *v1beta1.BackupConfiguration) {
		bc.Spec.Target = &v1beta1.BackupTarget{
			Ref: GetTargetRef(appBinding.Name, apis.KindAppBinding),
		}
		bc.Spec.Task.Name = MySQLBackupTask
	})

	// transformFuncs provides a array of functions that made test specific change on the BackupConfiguration
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(backupConfig)
	}

	By("Creating BackupConfiguration: " + backupConfig.Name)
	createdBC, err := f.StashClient.StashV1beta1().BackupConfigurations(backupConfig.Namespace).Create(backupConfig)
	f.AppendToCleanupList(createdBC)

	By("Verifying that backup triggering CronJob has been created")
	f.EventuallyCronJobCreated(backupConfig.ObjectMeta).Should(BeTrue())

	return createdBC, err
}

func (f *Invocation) MySQLAddonInstalled() bool {
	_, err := f.StashClient.StashV1beta1().Functions().Get(MySQLBackupFunction, metav1.GetOptions{})
	if err != nil {
		return false
	}

	_, err = f.StashClient.StashV1beta1().Functions().Get(MySQLRestoreFunction, metav1.GetOptions{})
	if err != nil {
		return false
	}

	_, err = f.StashClient.StashV1beta1().Tasks().Get(MySQLBackupTask, metav1.GetOptions{})
	if err != nil {
		return false
	}

	_, err = f.StashClient.StashV1beta1().Tasks().Get(MySQLRestoreTask, metav1.GetOptions{})

	return err == nil
}