package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"github.com/SeasonPilot/etcd-operator/pkg/file"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/snapshot"
	"go.uber.org/zap"
)

var timeOut int

func main() {
	var (
		backupTempDir string
		etcdURL       string
	)

	flag.StringVar(&backupTempDir, "backup-path", "", "path for backup")
	flag.StringVar(&etcdURL, "etcd-url", "", "url for etcd")
	flag.Parse() // fixme:

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeOut))
	defer cancel()

	l := &zap.Logger{}

	localPath := filepath.Join(backupTempDir, "snapshot.db")

	// backup etcd
	err := snapshot.Save(ctx, l, clientv3.Config{
		Endpoints:   []string{etcdURL},
		DialTimeout: time.Second * time.Duration(timeOut),
	}, localPath)
	if err != nil {
		panic(err)
	}

	// upload
	endpoint := "play.min.io"
	accessKeyID := "Q3AM3UQ867SPQQA43P2F"
	secretAccessKey := "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG"
	backetName := ""

	uploader := file.NewS3Uploader(endpoint, accessKeyID, secretAccessKey)
	size, err := uploader.Upload(ctx, backetName, "snapshot.db", localPath)
	if err != nil {
		return
	}

	fmt.Printf("upload size %d", size)

}
