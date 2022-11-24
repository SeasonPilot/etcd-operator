package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/SeasonPilot/etcd-operator/api/v1alpha1"
	"github.com/SeasonPilot/etcd-operator/pkg/file"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/snapshot"
	"go.uber.org/zap"
)

func main() {
	var (
		backupTempDir      string
		etcdURL            string
		backupURL          string
		dialTimeoutSeconds int64
		timeoutSeconds     int64
	)

	flag.StringVar(&backupTempDir, "backup-path", os.TempDir(), "path for backup")
	flag.StringVar(&etcdURL, "etcd-url", "", "url for etcd")
	flag.StringVar(&backupURL, "backup-url", "", "URL for backup etcd object storage.")
	flag.Int64Var(&dialTimeoutSeconds, "dial-timeout-seconds", 5, "Timeout for dialing the Etcd.")
	flag.Int64Var(&timeoutSeconds, "timeout-seconds", 60, "Timeout for Backup the Etcd.")
	flag.Parse() // fixme:

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeoutSeconds))
	defer cancel()

	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	l := zapLogger.Sugar()
	localPath := filepath.Join(backupTempDir, "snapshot.db")

	l.Infof("backupURL: %s\n", backupURL) // backupURL: s3://season/snapshot.db
	// backup etcd
	err = snapshot.Save(ctx, zapLogger, clientv3.Config{
		Endpoints:   []string{etcdURL},
		DialTimeout: time.Second * time.Duration(dialTimeoutSeconds),
	}, localPath)
	if err != nil {
		panic(err)
	}

	storageType, backetName, objectName, err := file.ParsePath(backupURL)
	if err != nil {
		return
	}

	endpoint := os.Getenv("ENDPOINT")
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY")
	secretAccessKey := os.Getenv("MINIO_SECRET_KEY")

	switch storageType {
	case string(v1alpha1.EtcdBackupStorageTypeS3):
		size, err := handleS3(ctx, endpoint, accessKeyID, secretAccessKey, backetName, objectName, localPath)
		if err != nil {
			l.Errorf("s3 upload err: %s", err)
			return
		}
		l.Infof("upload size %d", size)
	case string(v1alpha1.EtcdBackupStorageTypeOSS):
		handlerOSS()
	}

}

func handleS3(ctx context.Context, endpoint, accessKeyID, secretAccessKey, backetName, objectName, localPath string) (int64, error) {
	// upload
	uploader := file.NewS3Uploader(endpoint, accessKeyID, secretAccessKey)
	return uploader.Upload(ctx, backetName, objectName, localPath)
}

func handlerOSS() {

}
