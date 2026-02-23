package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sandeepkandula/foldersync/sync"
)

func main() {
	src := flag.String("src", "", "source directory (required)")
	bucket := flag.String("bucket", "", "S3 destination bucket (required)")
	prefix := flag.String("prefix", "", "key prefix within the bucket")
	region := flag.String("region", "us-east-1", "AWS region")
	storageClass := flag.String("storage-class", "GLACIER_IR",
		"S3 storage class: GLACIER_IR (cheapest, instant access), STANDARD_IA, STANDARD")
	dryRun := flag.Bool("dry-run", false, "print actions without making changes")
	delete := flag.Bool("delete", false, "delete S3 objects absent from src")
	flag.Parse()

	if *src == "" || *bucket == "" {
		fmt.Fprintln(os.Stderr, "usage: foldersync -src <dir> -bucket <bucket> [options]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(*region))
	if err != nil {
		log.Fatalf("load AWS config: %v", err)
	}

	dst := sync.NewS3Destination(
		s3.NewFromConfig(cfg),
		*bucket,
		*prefix,
		types.StorageClass(*storageClass),
	)

	if err := sync.Sync(ctx, sync.Options{
		Src:    *src,
		Dst:    dst,
		DryRun: *dryRun,
		Delete: *delete,
	}); err != nil {
		log.Fatalf("sync failed: %v", err)
	}
}
