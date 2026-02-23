package sync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Destination uploads files to an S3 bucket using the specified storage class.
//
// Recommended storage classes for infrequent access (cheapest first):
//
//	GLACIER_IR   – Glacier Instant Retrieval ($0.004/GB, millisecond access)
//	STANDARD_IA  – Standard Infrequent Access ($0.0125/GB, millisecond access)
//	STANDARD     – Standard ($0.023/GB, always available)
type S3Destination struct {
	client       *s3.Client
	uploader     *manager.Uploader
	bucket       string
	prefix       string
	storageClass types.StorageClass
}

// NewS3Destination creates a new S3Destination.
func NewS3Destination(client *s3.Client, bucket, prefix string, storageClass types.StorageClass) *S3Destination {
	return &S3Destination{
		client:       client,
		uploader:     manager.NewUploader(client),
		bucket:       bucket,
		prefix:       prefix,
		storageClass: storageClass,
	}
}

func (d *S3Destination) fullKey(rel string) string {
	rel = strings.TrimPrefix(rel, "/")
	if d.prefix == "" {
		return rel
	}
	return strings.TrimSuffix(d.prefix, "/") + "/" + rel
}

func (d *S3Destination) relKey(full string) string {
	if d.prefix == "" {
		return full
	}
	return strings.TrimPrefix(full, strings.TrimSuffix(d.prefix, "/")+"/")
}

func (d *S3Destination) Put(ctx context.Context, rel string, r io.Reader, size int64, modTime time.Time) error {
	_, err := d.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(d.bucket),
		Key:          aws.String(d.fullKey(rel)),
		Body:         r,
		StorageClass: d.storageClass,
		Metadata: map[string]string{
			"mtime": strconv.FormatInt(modTime.Unix(), 10),
			"size":  strconv.FormatInt(size, 10),
		},
	})
	return err
}

func (d *S3Destination) Stat(ctx context.Context, rel string) (*ObjectMeta, error) {
	out, err := d.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(d.fullKey(rel)),
	})
	if err != nil {
		var re *awshttp.ResponseError
		if errors.As(err, &re) && re.HTTPStatusCode() == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}

	meta := &ObjectMeta{Size: aws.ToInt64(out.ContentLength)}
	if v, ok := out.Metadata["mtime"]; ok {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			meta.ModTime = time.Unix(ts, 0)
		}
	}
	return meta, nil
}

func (d *S3Destination) List(ctx context.Context) ([]string, error) {
	prefix := d.prefix
	if prefix != "" {
		prefix = strings.TrimSuffix(prefix, "/") + "/"
	}

	paginator := s3.NewListObjectsV2Paginator(d.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(d.bucket),
		Prefix: aws.String(prefix),
	})

	var keys []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}
		for _, obj := range page.Contents {
			keys = append(keys, d.relKey(aws.ToString(obj.Key)))
		}
	}
	return keys, nil
}

func (d *S3Destination) Delete(ctx context.Context, rel string) error {
	_, err := d.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(d.fullKey(rel)),
	})
	return err
}
