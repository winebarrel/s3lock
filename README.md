# s3lock

[![CI](https://github.com/winebarrel/s3lock/actions/workflows/ci.yml/badge.svg)](https://github.com/winebarrel/s3lock/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/winebarrel/s3lock.svg)](https://pkg.go.dev/github.com/winebarrel/s3lock)

s3lock is a locking command using S3.

Exclusive control is implemented using [conditional writes](https://docs.aws.amazon.com/AmazonS3/latest/userguide/conditional-writes.html).

```sh
$ s3lock lock s3://my-bucket/lock-object
s3://my-bucket/lock-object has been locked
create lock-object.lock

# A locked object cannot be double-locked
$ s3lock lock s3://my-bucket/lock-object
s3lock: error: lock already held

$ s3lock unlock lock-object.lock
s3://my-bucket/lock-object has been unlocked
delete lock-object.lock
```

## Installation

```
brew install winebarrel/s3lock/s3lock
```

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

<details>

<summary>s3lock lock</summary>

```
Usage: s3lock lock <s3-url> [flags]

Arguments:
  <s3-url>    S3 URL of the object to lock, e.g., s3://bucket/lock-obj-key

Flags:
  -h, --help             Show context-sensitive help.
      --version

  -w, --wait=UINT        Wait for the specified number of seconds until it locks.
  -o, --output=STRING    Lock file output path (default: <lock-obj-key>.lock)
  -f, --force            Force lock by overwriting any existing lock.
```

</details>
<details>

<summary>s3lock unlock</summary>

```
Usage: s3lock unlock <lock-file> [flags]

Arguments:
  <lock-file>    Lock file path.

Flags:
  -h, --help       Show context-sensitive help.
      --version

  -f, --force      Ignore unlocked errors and lock mismatch errors.
```

</details>

### Use as a library

```go
package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/winebarrel/s3lock"
)

func main() {
	ctx := context.Background()

	cfg, _ := config.LoadDefaultConfig(ctx)
	s3cli := s3.NewFromConfig(cfg)

	obj := s3lock.New(s3cli, "my-bucket", "lock-object")
	lock, err := obj.Lock(ctx)

	if err != nil {
		log.Fatal(err)
	}

	defer lock.Unlock()

	// ...
}
```

## Lock Sequence

```mermaid
sequenceDiagram
    autonumber
    actor U as User
    participant C as s3lock CLI
    participant O as s3lock.Object
    participant S as Amazon S3

    U->>C: s3lock lock s3://bucket/key [--wait N] [--force]
    C->>O: New(s3Client, bucket, key)

    alt No --wait, no --force
        C->>O: Lock(ctx)
        O->>S: PutObject(body=uuid, If-None-Match="*")
        alt Object does not exist
            S-->>O: 200 OK (ETag)
            O-->>C: Lock{id, etag}
            C->>C: Save lock JSON to output file
            C-->>U: locked + lock file created
        else Lock already exists
            S-->>O: 412 Precondition Failed
            O-->>C: ErrLockAlreadyHeld
            C-->>U: error: lock already held
        end
    else --force only
        C->>O: ForceLock(ctx)
        O->>S: PutObject(body=uuid)
        S-->>O: 200 OK (ETag)
        O-->>C: Lock{id, etag}
        C->>C: Save lock JSON to output file
        C-->>U: locked + lock file created
    else --wait N only
        C->>C: context.WithTimeout(N seconds)
        C->>O: LockWait(ctx)
        O->>S: PutObject(body=uuid, If-None-Match="*")
        alt First attempt succeeds
            S-->>O: 200 OK (ETag)
            O-->>C: Lock{id, etag}
            C->>C: Save lock JSON to output file
            C-->>U: locked + lock file created
        else First attempt returns ErrLockAlreadyHeld
            S-->>O: 412 Precondition Failed
            loop Every LockWaitInterval (default: 1s)
                O->>S: PutObject(body=uuid, If-None-Match="*")
                alt Succeeds
                    S-->>O: 200 OK (ETag)
                    O-->>C: Lock{id, etag}
                    C->>C: Save lock JSON to output file
                    C-->>U: locked + lock file created
                else Still locked
                    S-->>O: 412 Precondition Failed
                    O-->>O: Continue retrying
                end
            end
            alt Timeout / context done
                O-->>C: ErrLockAlreadyHeld
                C-->>U: error: lock already held
            end
        end
    else --wait N --force
        C->>C: context.WithTimeout(N seconds)
        C->>O: ForceLockWait(ctx)
        O->>S: PutObject(body=uuid, If-None-Match="*")
        alt First attempt succeeds
            S-->>O: 200 OK (ETag)
            O-->>C: Lock{id, etag}
            C->>C: Save lock JSON to output file
            C-->>U: locked + lock file created
        else First attempt returns ErrLockAlreadyHeld
            S-->>O: 412 Precondition Failed
            loop Every LockWaitInterval (default: 1s)
                O->>S: PutObject(body=uuid, If-None-Match="*")
                alt Succeeds
                    S-->>O: 200 OK (ETag)
                    O-->>C: Lock{id, etag}
                    C->>C: Save lock JSON to output file
                    C-->>U: locked + lock file created
                else Still locked
                    S-->>O: 412 Precondition Failed
                    O-->>O: Continue retrying
                end
            end
            alt Timeout / context done
                O->>S: PutObject(body=uuid) [force, no If-None-Match]
                S-->>O: 200 OK (ETag)
                O-->>C: Lock{id, etag}
                C->>C: Save lock JSON to output file
                C-->>U: locked + lock file created
            end
        end
    end
```

## Unlock Sequence

```mermaid
sequenceDiagram
    autonumber
    actor U as User
    participant C as s3lock CLI
    participant L as s3lock.Lock
    participant S as Amazon S3

    U->>C: s3lock unlock lock-file [--force]
    C->>C: Read lock file JSON
    C->>L: NewLockFromJSON(s3Client, data)
    C->>L: Unlock()

    L->>L: validate(ctx)
    L->>S: GetObject(If-Match=etag)

    alt 200 OK and body matches id
        S-->>L: 200 OK (body=id)
        L->>S: DeleteObject(If-Match=etag)
        S-->>L: Success
        L-->>C: nil
        C->>C: Delete lock file
        C-->>U: unlocked + lock file deleted
    else 404 Not Found
        S-->>L: 404 Not Found
        L-->>C: ErrAlreadyUnlocked
        alt --force
            C->>C: Ignore error, delete lock file
            C-->>U: unlocked + lock file deleted
        else No --force
            C-->>U: error: already unlocked
        end
    else 412 Precondition Failed (ETag mismatch)
        S-->>L: 412 Precondition Failed
        L-->>C: ErrLockMismatch
        alt --force
            C->>C: Ignore error, delete lock file
            C-->>U: unlocked + lock file deleted
        else No --force
            C-->>U: error: lock mismatch
        end
    else 200 OK but body does not match id
        S-->>L: 200 OK (body≠id)
        L-->>C: ErrLockMismatch
        alt --force
            C->>C: Ignore error, delete lock file
            C-->>U: unlocked + lock file deleted
        else No --force
            C-->>U: error: lock mismatch
        end
    end
```
