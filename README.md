# foldersync

A CLI tool that syncs a local directory to an AWS S3 bucket. Defaults to **S3 Glacier Instant Retrieval** — the cheapest storage class with millisecond access, making it ideal for backups and infrequent access workloads.

## Features

- Incremental sync — skips files already up to date (matched by size and modification time)
- Dry-run mode — preview what would change without touching anything
- Mirror mode — optionally delete S3 objects that no longer exist locally
- Configurable storage class
- Supports key prefixes for organizing objects within a bucket

## Installation

```sh
git clone https://github.com/r3dstang/foldersync.git
cd foldersync
go mod tidy
go build -o foldersync .
```

## Usage

```sh
foldersync -src <directory> -bucket <bucket-name> [options]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `-src` | _(required)_ | Local source directory |
| `-bucket` | _(required)_ | S3 destination bucket |
| `-prefix` | `""` | Key prefix within the bucket |
| `-region` | `us-east-1` | AWS region |
| `-storage-class` | `GLACIER_IR` | S3 storage class (see below) |
| `-dry-run` | `false` | Print actions without making changes |
| `-delete` | `false` | Delete S3 objects absent from source |

### Storage Classes

| Class | Cost (storage) | Access time | Best for |
|---|---|---|---|
| `GLACIER_IR` | $0.004/GB/mo | Milliseconds | Backups, infrequent access |
| `STANDARD_IA` | $0.0125/GB/mo | Milliseconds | Slightly more frequent access |
| `STANDARD` | $0.023/GB/mo | Milliseconds | Frequent access |

## Examples

Dry-run to preview what would be uploaded:
```sh
foldersync -src ./photos -bucket my-backup-bucket -dry-run
```

Sync with a key prefix:
```sh
foldersync -src ./photos -bucket my-backup-bucket -prefix backups/photos
```

Mirror mode — keep S3 in sync with local (deletes removed files):
```sh
foldersync -src ./photos -bucket my-backup-bucket -delete
```

Use a different storage class:
```sh
foldersync -src ./docs -bucket my-backup-bucket -storage-class STANDARD_IA
```

## AWS Authentication

`foldersync` uses the standard AWS credential chain. Any of the following will work:

- **Environment variables:** `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
- **AWS credentials file:** `~/.aws/credentials`
- **IAM role** (EC2 instance profile, ECS task role, etc.)

The IAM principal needs the following S3 permissions on the target bucket:

```json
{
  "Effect": "Allow",
  "Action": [
    "s3:PutObject",
    "s3:HeadObject",
    "s3:ListBucket",
    "s3:DeleteObject"
  ],
  "Resource": [
    "arn:aws:s3:::my-backup-bucket",
    "arn:aws:s3:::my-backup-bucket/*"
  ]
}
```
