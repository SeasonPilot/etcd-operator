package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

	localPath := filepath.Join(backupTempDir, "snapshot.db")

	// backup etcd
	err = snapshot.Save(ctx, zapLogger, clientv3.Config{
		Endpoints:   []string{etcdURL},
		DialTimeout: time.Second * time.Duration(dialTimeoutSeconds),
	}, localPath)
	if err != nil {
		panic(err)
	}

	// upload
	endpoint := "play.min.io"
	accessKeyID := "Q3AM3UQ867SPQQA43P2F"
	secretAccessKey := "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG"
	backetName := "season"

	uploader := file.NewS3Uploader(endpoint, accessKeyID, secretAccessKey)
	size, err := uploader.Upload(ctx, backetName, "snapshot.db", localPath)
	if err != nil {
		return
	}

	fmt.Printf("upload size %d", size)

}
