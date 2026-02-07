# s3lock

[![CI](https://github.com/winebarrel/s3lock/actions/workflows/ci.yml/badge.svg)](https://github.com/winebarrel/s3lock/actions/workflows/ci.yml)

s3lock is a locking command using S3.

## Usage

```
Usage: s3lock <command> [flags]

Flags:
  -h, --help       Show context-sensitive help.
      --version

Commands:
  lock <s3-url> [flags]

  unlock <lock-file> [flags]

Run "s3lock <command> --help" for more information on a command.
```

```sh
$ s3lock lock s3://my-bucket/lock-object > object.lock

# A locked object cannot be double-locked
$ s3lock lock s3://my-bucket/lock-object
s3lock: error: lock already held

$ s3lock unlock object.lock

$ s3lock unlock object.lock
s3lock: error: already unlocked
```
