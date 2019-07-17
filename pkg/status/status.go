package status

import (
	"fmt"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/apis/core"
	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	stash_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
	stash_util_v1beta1 "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/restic"
)

type UpdateStatusOptions struct {
	KubeClient  kubernetes.Interface
	StashClient *cs.Clientset

	Namespace      string
	Repository     string
	BackupSession  string
	RestoreSession string
	OutputDir      string
	OutputFileName string
}

func (o UpdateStatusOptions) UpdateBackupStatusFromFile() error {
	// read backup output from file
	backupOutput, err := restic.ReadBackupOutput(filepath.Join(o.OutputDir, o.OutputFileName))
	if err != nil {
		return err
	}

	var backupErr []error
	for _, hostStats := range backupOutput.HostBackupStats {
		if hostStats.Error != "" {
			backupErr = append(backupErr, fmt.Errorf(hostStats.Error))
		}
	}

	updateStatusErr := o.UpdatePostBackupStatus(backupOutput)
	if updateStatusErr != nil {
		backupErr = append(backupErr, updateStatusErr)
	}

	if backupErr != nil {
		return errors.NewAggregate(backupErr)
	}
	return nil
}

func (o UpdateStatusOptions) UpdateRestoreStatusFromFile() error {
	// read restore output from file
	restoreOutput, err := restic.ReadRestoreOutput(filepath.Join(o.OutputDir, o.OutputFileName))
	if err != nil {
		return err
	}

	var restoreErr []error
	for _, hostStats := range restoreOutput.HostRestoreStats {
		if hostStats.Error != "" {
			restoreErr = append(restoreErr, fmt.Errorf(hostStats.Error))
		}
	}

	updateStatusErr := o.UpdatePostRestoreStatus(restoreOutput)
	if updateStatusErr != nil {
		restoreErr = append(restoreErr, updateStatusErr)
	}

	if restoreErr != nil {
		return errors.NewAggregate(restoreErr)
	}
	return nil
}

func (o UpdateStatusOptions) UpdatePostBackupStatus(backupOutput *restic.BackupOutput) error {
	// get backup session, update status and create event
	backupSession, err := o.StashClient.StashV1beta1().BackupSessions(o.Namespace).Get(o.BackupSession, metav1.GetOptions{})
	if err != nil {
		return err
	}

	overallBackupSucceeded := true

	// add or update entry for each host in BackupSession status
	for _, hostStats := range backupOutput.HostBackupStats {
		_, err = stash_util_v1beta1.UpdateBackupSessionStatusForHost(o.StashClient.StashV1beta1(), backupSession, hostStats)
		if err != nil {
			return err
		}
		// create event for backup session
		var eventType, eventReason, eventMessage string
		if hostStats.Error != "" {
			overallBackupSucceeded = false
			eventType = core.EventTypeWarning
			eventReason = eventer.EventReasonHostBackupFailed
			eventMessage = fmt.Sprintf("backup failed for host %q. Reason: %s", hostStats.Hostname, hostStats.Error)
		} else {
			eventType = core.EventTypeNormal
			eventReason = eventer.EventReasonHostBackupSucceded
			eventMessage = fmt.Sprintf("backup succeeded for host %s", hostStats.Hostname)
		}
		_, err = eventer.CreateEvent(
			o.KubeClient,
			eventer.EventSourcePostBackupStatusUpdater,
			backupSession,
			eventType,
			eventReason,
			eventMessage,
		)
		if err != nil {
			return err
		}
	}

	// no need to update repository status for failed backup
	if !overallBackupSucceeded {
		return nil
	}

	bs, err := o.StashClient.StashV1beta1().BackupConfigurations(o.Namespace).Get(backupSession.Spec.BackupConfiguration.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if bs.Spec.Driver != api_v1beta1.VolumeSnapshotter {
		// get repository and update status
		repository, err := o.StashClient.StashV1alpha1().Repositories(o.Namespace).Get(o.Repository, metav1.GetOptions{})
		if err != nil {
			return err
		}
		_, err = stash_util.UpdateRepositoryStatus(
			o.StashClient.StashV1alpha1(),
			repository,
			func(in *api.RepositoryStatus) *api.RepositoryStatus {
				// TODO: fix Restic Wrapper
				in.Integrity = backupOutput.RepositoryStats.Integrity
				in.Size = backupOutput.RepositoryStats.Size
				in.SnapshotCount = backupOutput.RepositoryStats.SnapshotCount
				in.SnapshotsRemovedOnLastCleanup = backupOutput.RepositoryStats.SnapshotsRemovedOnLastCleanup

				currentTime := metav1.Now()
				in.LastBackupTime = &currentTime

				if in.FirstBackupTime == nil {
					in.FirstBackupTime = &currentTime
				}
				return in
			},
			apis.EnableStatusSubresource,
		)
	}
	return err
}

func (o UpdateStatusOptions) UpdatePostRestoreStatus(restoreOutput *restic.RestoreOutput) error {
	// get restore session, update status and create event
	restoreSession, err := o.StashClient.StashV1beta1().RestoreSessions(o.Namespace).Get(o.RestoreSession, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// add or update entry for this host in RestoreSession status
	for _, hostStats := range restoreOutput.HostRestoreStats {

		_, err = stash_util_v1beta1.UpdateRestoreSessionStatusForHost(o.StashClient.StashV1beta1(), restoreSession, hostStats)
		if err != nil {
			return err
		}

		// create event for restore session
		var eventType, eventReason, eventMessage string
		if hostStats.Error != "" {
			eventType = core.EventTypeWarning
			eventReason = eventer.EventReasonHostRestoreFailed
			eventMessage = fmt.Sprintf("restore failed for host %q. Reason: %s", hostStats.Hostname, hostStats.Error)
		} else {
			eventType = core.EventTypeNormal
			eventReason = eventer.EventReasonHostRestoreSucceeded
			eventMessage = fmt.Sprintf("restore succeeded for host %q", hostStats.Hostname)
		}
		_, err = eventer.CreateEvent(
			o.KubeClient,
			eventer.EventSourcePostRestoreStatusUpdater,
			restoreSession,
			eventType,
			eventReason,
			eventMessage,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
