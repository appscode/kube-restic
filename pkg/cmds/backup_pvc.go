package cmds

import (
	"path/filepath"

	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"
)

func NewCmdBackupPVC() *cobra.Command {
	var (
		outputDir string
		backupOpt = restic.BackupOptions{
			Host: restic.DefaultHost,
		}
		setupOpt = restic.SetupOptions{
			ScratchDir:  restic.DefaultScratchDir,
			EnableCache: false,
		}
	)

	cmd := &cobra.Command{
		Use:               "backup-pvc",
		Short:             "Takes a backup of Persistent Volume Claim",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "backup-dirs", "provider", "secret-dir")

			// apply nice, ionice settings from env
			var err error
			setupOpt.Nice, err = util.NiceSettingsFromEnv()
			if err != nil {
				return util.HandleResticError(outputDir, restic.DefaultOutputFileName, err)
			}
			setupOpt.IONice, err = util.IONiceSettingsFromEnv()
			if err != nil {
				return util.HandleResticError(outputDir, restic.DefaultOutputFileName, err)
			}

			// init restic wrapper
			resticWrapper, err := restic.NewResticWrapper(setupOpt)
			if err != nil {
				return util.HandleResticError(outputDir, restic.DefaultOutputFileName, err)
			}
			// Run backup
			backupOutput, backupErr := resticWrapper.RunBackup(backupOpt)
			if backupErr != nil {
				return util.HandleResticError(outputDir, restic.DefaultOutputFileName, backupErr)
			}
			// If output directory specified, then write the output in "output.json" file in the specified directory
			if outputDir != "" {
				return backupOutput.WriteOutput(filepath.Join(outputDir, restic.DefaultOutputFileName))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&setupOpt.Provider, "provider", setupOpt.Provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&setupOpt.Bucket, "bucket", setupOpt.Bucket, "Name of the cloud bucket/container (keep empty for local backend)")
	cmd.Flags().StringVar(&setupOpt.Endpoint, "endpoint", setupOpt.Endpoint, "Endpoint for s3/s3 compatible backend or REST server URL")
	cmd.Flags().StringVar(&setupOpt.Path, "path", setupOpt.Path, "Directory inside the bucket where backed up data will be stored")
	cmd.Flags().StringVar(&setupOpt.SecretDir, "secret-dir", setupOpt.SecretDir, "Directory where storage secret has been mounted")
	cmd.Flags().StringVar(&setupOpt.ScratchDir, "scratch-dir", setupOpt.ScratchDir, "Temporary directory")
	cmd.Flags().BoolVar(&setupOpt.EnableCache, "enable-cache", setupOpt.EnableCache, "Specify whether to enable caching for restic")
	cmd.Flags().IntVar(&setupOpt.MaxConnections, "max-connections", setupOpt.MaxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")

	cmd.Flags().StringVar(&backupOpt.Host, "hostname", backupOpt.Host, "Name of the host machine")
	cmd.Flags().StringSliceVar(&backupOpt.BackupPaths, "backup-paths", backupOpt.BackupPaths, "List of paths to backup")

	cmd.Flags().IntVar(&backupOpt.RetentionPolicy.KeepLast, "retention-keep-last", backupOpt.RetentionPolicy.KeepLast, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupOpt.RetentionPolicy.KeepHourly, "retention-keep-hourly", backupOpt.RetentionPolicy.KeepHourly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupOpt.RetentionPolicy.KeepDaily, "retention-keep-daily", backupOpt.RetentionPolicy.KeepDaily, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupOpt.RetentionPolicy.KeepWeekly, "retention-keep-weekly", backupOpt.RetentionPolicy.KeepWeekly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupOpt.RetentionPolicy.KeepMonthly, "retention-keep-monthly", backupOpt.RetentionPolicy.KeepMonthly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupOpt.RetentionPolicy.KeepYearly, "retention-keep-yearly", backupOpt.RetentionPolicy.KeepYearly, "Specify value for retention strategy")
	cmd.Flags().StringSliceVar(&backupOpt.RetentionPolicy.KeepTags, "retention-keep-tags", backupOpt.RetentionPolicy.KeepTags, "Specify value for retention strategy")
	cmd.Flags().BoolVar(&backupOpt.RetentionPolicy.Prune, "retention-prune", backupOpt.RetentionPolicy.Prune, "Specify whether to prune old snapshot data")
	cmd.Flags().BoolVar(&backupOpt.RetentionPolicy.DryRun, "retention-dry-run", backupOpt.RetentionPolicy.DryRun, "Specify whether to test retention policy without deleting actual data")

	cmd.Flags().StringVar(&outputDir, "output-dir", outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	return cmd
}
