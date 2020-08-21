[![Go Report Card](https://goreportcard.com/badge/stash.appscode.dev/stash)](https://goreportcard.com/report/stash.appscode.dev/stash)
[![Build Status](https://github.com/stashed/stash/workflows/CI/badge.svg)](https://github.com/stashed/stash/actions?workflow=CI)
[![codecov](https://codecov.io/gh/stashed/stash/branch/master/graph/badge.svg)](https://codecov.io/gh/stashed/stash)
[![Docker Pulls](https://img.shields.io/docker/pulls/appscode/stash.svg)](https://hub.docker.com/r/appscode/stash/)
[![Slack](https://slack.appscode.com/badge.svg)](https://slack.appscode.com)
[![Twitter](https://img.shields.io/twitter/follow/kubestash.svg?style=social&logo=twitter&label=Follow)](https://twitter.com/intent/follow?screen_name=KubeStash)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fstashed%2Fstash.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fstashed%2Fstash?ref=badge_shield)

# Stash

[Stash](https://stash.run) by AppsCode is a cloud-native data backup and recovery solution for Kubernetes workloads. If you are running production workloads in Kubernetes, you might want to take backup of your disks, databases, etc. Traditional tools are too complex to setup and maintain in a dynamic compute environment like Kubernetes. Stash is a Kubernetes operator that uses [restic](https://github.com/restic/restic) or Kubernetes CSI Driver VolumeSnapshotter functionality to address these issues. Using Stash, you can backup Kubernetes volumes mounted in workloads, stand-alone volumes, and databases. User may even extend Stash via [addons](https://stash.run/docs/latest/guides/latest/addons/overview/) for any custom workload.

## Features

| Features                                                                                | Community Edition | Enterprise Edition | Scope                                                                                                                                                               |
| --------------------------------------------------------------------------------------- | :---------------: | :----------------: | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Backup & Restore Workload Data                                                          |     &#10003;      |      &#10003;      | Deployment, DaemonSet, StatefulSet, ReplicaSet, ReplicationController, OpenShift DeploymentConfig                                                                   |
| Backup & Restore Stand-alone Volume (PVC)                                               |     &#10003;      |      &#10003;      | PersistentVolumeClaim, PersistentVolume                                                                                                                             |
| Backup & Restore databases                                                              |     &#10003;      |      &#10003;      | PostgreSQL, MySQL, MongoDB, Elasticsearch, Percona-XtraDB                                                                                                           |
| Schedule Backup, Instant Backup                                                         |     &#10003;      |      &#10003;      | Schedule through [cron expression](https://en.wikipedia.org/wiki/Cron) or trigger instant backup using Stash Kubernetes plugin                                      |
| Pause Backup                                                                            |     &#10003;      |      &#10003;      | No new backup when paused.                                                                                                                                          |
| Backup & Restore subset of files                                                        |     &#10003;      |      &#10003;      | Only backup/restore the files that matches the provided patterns                                                                                                    |
| Cleanup old snapshots automatically                                                     |     &#10003;      |      &#10003;      | Cleanup old snapshots according to different [retention policies](https://restic.readthedocs.io/en/stable/060_forget.html#removing-snapshots-according-to-a-policy) |
| Encryption, Deduplication (send only diff)                                              |     &#10003;      |      &#10003;      | Encrypt backed up data with AES-256. Stash only sends the changes since last backup.                                                                                |
| [CSI Driver Integration](https://kubernetes.io/docs/concepts/storage/volume-snapshots/) |     &#10003;      |      &#10003;      | VolumeSnapshot for Kubernetes workloads. Supported for Kubernetes v1.17.0+.                                                                                         |
| Prometheus Metrics                                                                      |     &#10003;      |      &#10003;      | Rich backup metrics, restore metrics and Stash operator metrics.                                                                                                    |
| Security                                                                                |     &#10003;      |      &#10003;      | Built-in support for RBAC, PSP and Network Policy                                                                                                                   |
| CLI                                                                                     |     &#10003;      |      &#10003;      | `kubectl` plugin (for Kubernetes 1.12+)                                                                                                                             |
| Extensibility and Customizability                                                       |     &#10003;      |      &#10003;      | Write addons for bespoke applications and customize currently supported workloads                                                                                   |
| Hooks                                                                                   |     &#10003;      |      &#10003;      | Execute `httpGet`, `httpPost`, `tcpSocket` and `exec` hooks before and after of backup or restore process.                                                          |
| Cloud Storage as Backend                                                                |     &#10003;      |      &#10003;      | Stores backup data in AWS S3, Minio, Rook, GCS, Azure, OpenStack Swift, Backblaze B2 and Rest Server                                                                |
| On-prem Storage as Backend                                                              |     &#10007;      |      &#10003;      | Stores backup data in any locally mounted Kubernetes Volumes such as NFS, etc.                                                                                      |
| Auto Backup                                                                             |     &#10007;      |      &#10003;      | Share backup configuration across workloads using templates. Enable backup for a target application via annotation.                                                 |
| Batch Backup & Batch Restore                                                            |     &#10007;      |      &#10003;      | Backup and restore co-related applications (eg, WordPress server and its database) together                                                                         |
| Point-In-Time Recovery (PITR)                                                           |     &#10007;      |      Planned       | Restore a set of files from a time in the past.                                                                                                                     |

## Supported Versions

Please pick a version of Stash that matches your Kubernetes installation.

| Stash Version                                                                       | Docs                                                          | Kubernetes Version |
| ----------------------------------------------------------------------------------- | ------------------------------------------------------------- | ------------------ |
| [v0.9.0-rc.6](https://github.com/stashed/stash/releases/tag/v0.9.0-rc.6) (uses CRD) | [User Guide](https://stash.run/docs/v0.9.0-rc.6)              | 1.11.x+            |
| [0.8.3](https://github.com/stashed/stash/releases/tag/0.8.3) (uses CRD)             | [User Guide](https://stash.run/docs/0.8.3)                    | 1.9.x+             |
| [0.7.0](https://github.com/stashed/stash/releases/tag/0.7.0) (uses CRD)             | [User Guide](https://stash.run/docs/0.7.0)                    | 1.8.x              |
| [0.6.4](https://github.com/stashed/stash/releases/tag/0.6.4) (uses CRD)             | [User Guide](https://stash.run/docs/0.6.4)                    | 1.7.x              |
| [0.4.2](https://github.com/stashed/stash/releases/tag/0.4.2) (uses TPR)             | [User Guide](https://github.com/stashed/docs/tree/0.4.2/docs) | 1.5.x - 1.6.x      |

## Installation

To install Stash, please follow the guide [here](https://stash.run/docs/latest/setup/install).

## Using Stash

Want to learn how to use Stash? Please start [here](https://stash.run/docs/latest/).

## Stash API Clients

You can use Stash API clients to programmatically access its objects. Here are the supported clients:

- Go: [https://github.com/stashed/stash](/client/clientset/versioned)
- Java: https://github.com/stashed/java

## Contribution guidelines

Want to help improve Stash? Please start [here](https://stash.run/docs/latest/welcome/contributing).

---

**Stash binaries collect anonymous usage statistics to help us learn how the software is being used and how we can improve it. To disable stats collection, run the operator with the flag** `--enable-analytics=false`.

---

## Acknowledgement

- Many thanks to [Alexander Neumann](https://github.com/fd0) for [Restic](https://restic.net) project.

## Support

We use Slack for public discussions. To chit chat with us or the rest of the community, join us in the [AppsCode Slack team](https://appscode.slack.com/messages/C8NCX6N23/details/) channel `#stash`. To sign up, use our [Slack inviter](https://slack.appscode.com/).

If you have found a bug with Stash or want to request new features, please [file an issue](https://github.com/stashed/stash/issues/new).

## License

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fstashed%2Fstash.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fstashed%2Fstash?ref=badge_large)
