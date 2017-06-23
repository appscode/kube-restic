package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

type RetentionStrategy string

const (
	KeepLast    RetentionStrategy = "keep-last"
	KeepHourly  RetentionStrategy = "keep-hourly"
	KeepDaily   RetentionStrategy = "keep-daily"
	KeepWeekly  RetentionStrategy = "keep-weekly"
	KeepMonthly RetentionStrategy = "keep-monthly"
	KeepYearly  RetentionStrategy = "keep-yearly"
)

type Restic struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResticSpec   `json:"spec,omitempty"`
	Status            ResticStatus `json:"status,omitempty"`
}

type ResticSpec struct {
	Selector   metav1.LabelSelector `json:"selector,omitempty"`
	FileGroups []FileGroup          `json:"fileGroups,omitempty"`
	Backend    Backend              `json:"backend,omitempty"`
	Schedule   string               `json:"schedule,omitempty"`
}

type ResticStatus struct {
	FirstBackupTime          *metav1.Time `json:"firstBackupTime,omitempty"`
	LastBackupTime           *metav1.Time `json:"lastBackupTime,omitempty"`
	LastSuccessfulBackupTime *metav1.Time `json:"lastSuccessfulBackupTime,omitempty"`
	LastBackupDuration       string       `json:"lastBackupDuration,omitempty"`
	BackupCount              int64        `json:"backupCount,omitempty"`
}

type ResticList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Restic `json:"items,omitempty"`
}

type FileGroup struct {
	// Source of the backup volumeName:path
	Path string `json:"path,omitempty"`
	// Tags of a snapshots
	Tags []string `json:"tags,omitempty"`
	// retention policy of snapshots
	RetentionPolicy RetentionPolicy `json:"retentionPolicy,omitempty"`
}

type Source struct {
	VolumeName string `json:"volumeName,omitempty"`
	Path       string `json:"path,omitempty"`
}

type Backend struct {
	Local                *LocalSpec `json:"local"`
	S3                   *S3Spec    `json:"s3,omitempty"`
	GCS                  *GCSSpec   `json:"gcs,omitempty"`
	Azure                *AzureSpec `json:"azure,omitempty"`
	RepositorySecretName string     `json:"repositorySecretName,omitempty"`
}

type LocalSpec struct {
	Volume apiv1.Volume `json:"volume,omitempty"`
	Path   string       `json:"path,omitempty"`
}

type S3Spec struct {
	Endpoint string `json:"endpoint,omitempty"`
	Bucket   string `json:"bucket,omiempty"`
	Prefix   string `json:"prefix,omitempty"`
}

type GCSSpec struct {
	Location string `json:"location,omitempty"`
	Bucket   string `json:"bucket,omiempty"`
	Prefix   string `json:"prefix,omitempty"`
}

type AzureSpec struct {
	Container string `json:"container,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
}

type RetentionPolicy struct {
	KeepLastSnapshots    int      `json:"keepLastSnapshots,omitempty"`
	KeepHourlySnapshots  int      `json:"keepHourlySnapshots,omitempty"`
	KeepDailySnapshots   int      `json:"keepDailySnapshots,omitempty"`
	KeepWeeklySnapshots  int      `json:"keepWeeklySnapshots,omitempty"`
	KeepMonthlySnapshots int      `json:"keepMonthlySnapshots,omitempty"`
	KeepYearlySnapshots  int      `json:"keepYearlySnapshots,omitempty"`
	KeepTags             []string `json:"keepTags,omitempty"`
}
