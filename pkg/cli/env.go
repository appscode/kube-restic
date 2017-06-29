package cli

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/appscode/log"
	sapi "github.com/appscode/stash/api"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	RESTIC_REPOSITORY = "RESTIC_REPOSITORY"
	RESTIC_PASSWORD   = "RESTIC_PASSWORD"
	TMPDIR            = "TMPDIR"

	AWS_ACCESS_KEY_ID     = "AWS_ACCESS_KEY_ID"
	AWS_SECRET_ACCESS_KEY = "AWS_SECRET_ACCESS_KEY"

	GOOGLE_PROJECT_ID               = "GOOGLE_PROJECT_ID"
	GOOGLE_SERVICE_ACCOUNT_JSON_KEY = "GOOGLE_SERVICE_ACCOUNT_JSON_KEY"
	GOOGLE_APPLICATION_CREDENTIALS  = "GOOGLE_APPLICATION_CREDENTIALS"

	AZURE_ACCOUNT_NAME = "AZURE_ACCOUNT_NAME"
	AZURE_ACCOUNT_KEY  = "AZURE_ACCOUNT_KEY"

	REST_SERVER_USERNAME = "REST_SERVER_USERNAME"
	REST_SERVER_PASSWORD = "REST_SERVER_PASSWORD"

	B2_ACCOUNT_ID  = "B2_ACCOUNT_ID"
	B2_ACCOUNT_KEY = "B2_ACCOUNT_KEY"

	// For keystone v1 authentication
	ST_AUTH = "ST_AUTH"
	ST_USER = "ST_USER"
	ST_KEY  = "ST_KEY"
	// For keystone v2 authentication (some variables are optional)
	OS_AUTH_URL    = "OS_AUTH_URL"
	OS_REGION_NAME = "OS_REGION_NAME"
	OS_USERNAME    = "OS_USERNAME"
	OS_PASSWORD    = "OS_PASSWORD"
	OS_TENANT_ID   = "OS_TENANT_ID"
	OS_TENANT_NAME = "OS_TENANT_NAME"
	// For keystone v3 authentication (some variables are optional)
	OS_USER_DOMAIN_NAME    = "OS_USER_DOMAIN_NAME"
	OS_PROJECT_NAME        = "OS_PROJECT_NAME"
	OS_PROJECT_DOMAIN_NAME = "OS_PROJECT_DOMAIN_NAME"
	// For authentication based on tokens
	OS_STORAGE_URL = "OS_STORAGE_URL"
	OS_AUTH_TOKEN  = "OS_AUTH_TOKEN"
)

func (w *ResticWrapper) SetupEnv(resource *sapi.Restic, secret *apiv1.Secret) error {
	if v, ok := secret.Data[RESTIC_PASSWORD]; !ok {
		return errors.New("Missing repository password")
	} else {
		w.sh.SetEnv(RESTIC_PASSWORD, string(v))
	}

	tmpDir := filepath.Join(w.scratchDir, "restic-tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}
	w.sh.SetEnv(TMPDIR, tmpDir)

	backend := resource.Spec.Backend
	if backend.Local != nil {
		// if resource.Spec.SkipSmartPrefix




		r := backend.Local.Path
		w.sh.SetEnv(RESTIC_REPOSITORY, filepath.Join(r, w.hostname))
	} else if backend.S3 != nil {
		r := fmt.Sprintf("s3:%s:%s:%s", backend.S3.Endpoint, backend.S3.Bucket, backend.S3.Prefix)
		w.sh.SetEnv(RESTIC_REPOSITORY, filepath.Join(r, w.hostname))
		w.sh.SetEnv(AWS_ACCESS_KEY_ID, string(secret.Data[AWS_ACCESS_KEY_ID]))
		w.sh.SetEnv(AWS_SECRET_ACCESS_KEY, string(secret.Data[AWS_SECRET_ACCESS_KEY]))
	} else if backend.GCS != nil {
		r := fmt.Sprintf("gs:%s:%s:%s", backend.GCS.Location, backend.GCS.Bucket, backend.GCS.Prefix)
		w.sh.SetEnv(RESTIC_REPOSITORY, filepath.Join(r, w.hostname))
		w.sh.SetEnv(GOOGLE_PROJECT_ID, string(secret.Data[GOOGLE_PROJECT_ID]))
		jsonKeyPath := filepath.Join(w.scratchDir, "gcs_sa.json")
		err := ioutil.WriteFile(jsonKeyPath, secret.Data[GOOGLE_SERVICE_ACCOUNT_JSON_KEY], 600)
		if err != nil {
			return err
		}
		w.sh.SetEnv(GOOGLE_APPLICATION_CREDENTIALS, jsonKeyPath)
	} else if backend.Azure != nil {
		r := fmt.Sprintf("azure:%s:%s", backend.Azure.Container, backend.Azure.Prefix)
		w.sh.SetEnv(RESTIC_REPOSITORY, filepath.Join(r, w.hostname))
		w.sh.SetEnv(AZURE_ACCOUNT_NAME, string(secret.Data[AZURE_ACCOUNT_NAME]))
		w.sh.SetEnv(AZURE_ACCOUNT_KEY, string(secret.Data[AZURE_ACCOUNT_KEY]))
	} else if backend.Swift != nil {
		r := fmt.Sprintf("swift:%s:%s", backend.Swift.Container, backend.Swift.Prefix)
		w.sh.SetEnv(RESTIC_REPOSITORY, filepath.Join(r, w.hostname))
		// For keystone v1 authentication
		w.sh.SetEnv(ST_AUTH, string(secret.Data[ST_AUTH]))
		w.sh.SetEnv(ST_USER, string(secret.Data[ST_USER]))
		w.sh.SetEnv(ST_KEY, string(secret.Data[ST_KEY]))
		// For keystone v2 authentication (some variables are optional)
		w.sh.SetEnv(OS_AUTH_URL, string(secret.Data[OS_AUTH_URL]))
		w.sh.SetEnv(OS_REGION_NAME, string(secret.Data[OS_REGION_NAME]))
		w.sh.SetEnv(OS_USERNAME, string(secret.Data[OS_USERNAME]))
		w.sh.SetEnv(OS_PASSWORD, string(secret.Data[OS_PASSWORD]))
		w.sh.SetEnv(OS_TENANT_ID, string(secret.Data[OS_TENANT_ID]))
		w.sh.SetEnv(OS_TENANT_NAME, string(secret.Data[OS_TENANT_NAME]))
		// For keystone v3 authentication (some variables are optional)
		w.sh.SetEnv(OS_USER_DOMAIN_NAME, string(secret.Data[OS_USER_DOMAIN_NAME]))
		w.sh.SetEnv(OS_PROJECT_NAME, string(secret.Data[AZURE_ACCOUNT_NAME]))
		w.sh.SetEnv(AZURE_ACCOUNT_NAME, string(secret.Data[OS_PROJECT_NAME]))
		w.sh.SetEnv(OS_PROJECT_DOMAIN_NAME, string(secret.Data[OS_PROJECT_DOMAIN_NAME]))
		// For authentication based on tokens
		w.sh.SetEnv(OS_STORAGE_URL, string(secret.Data[OS_STORAGE_URL]))
		w.sh.SetEnv(OS_AUTH_TOKEN, string(secret.Data[OS_AUTH_TOKEN]))
	} else if backend.Rest != nil {
		u, err := url.Parse(backend.Rest.URL)
		if err != nil {
			return err
		}
		if username, ok := secret.Data[REST_SERVER_USERNAME]; ok {
			if password, ok := secret.Data[REST_SERVER_PASSWORD]; ok {
				u.User = url.UserPassword(string(username), string(password))
			} else {
				u.User = url.User(string(username))
			}
		}
		u.Path = filepath.Join(u.Path, w.hostname) // TODO: check
		r := fmt.Sprintf("rest:%s", u.String())
		w.sh.SetEnv(RESTIC_REPOSITORY, r)
	} else if backend.B2 != nil {
		r := fmt.Sprintf("b2:%s:%s", backend.B2.Bucket, backend.B2.Prefix)
		w.sh.SetEnv(RESTIC_REPOSITORY, filepath.Join(r, w.hostname))
		w.sh.SetEnv(B2_ACCOUNT_ID, string(secret.Data[B2_ACCOUNT_ID]))
		w.sh.SetEnv(B2_ACCOUNT_KEY, string(secret.Data[B2_ACCOUNT_KEY]))
	}
	return nil
}

func (w *ResticWrapper) DumpEnv() error {
	out, err := w.sh.Command("env").Output()
	if err != nil {
		return err
	}
	log.Debugf("ENV:\n%s", string(out))
	return nil
}
