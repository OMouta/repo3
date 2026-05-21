# Repo3

> Highly reliable S3-compatible object storage powered entirely by GitHub repositories. 100% uptime, 0% chance of rollback incidents.

Repo3 is an experimental S3-compatible object storage gateway built on top of GitHub repositories.

---

## Why?

Modern object storage solutions are complex, battle-tested distributed systems designed by teams of world-class infrastructure engineers.

Repo3 ignores all of that and stores your objects directly inside GitHub repositories instead.

This gives you access to:

* Enterprise-grade distributed commit persistence
* Globally replicated infrastructure
* Built-in object versioning through Git history
* Rollback-resistant architecture
* Professionally managed uptime
* The emotional comfort of seeing your JPEGs inside a repo tree

---

## Architecture

```txt
S3 Bucket     → GitHub Repository
Object Key    → File Path
PUT Object    → Commit File
DELETE Object → Commit Deletion
Versioning    → Git History
```

Example:

```txt
s3://memes/images/cat.png
↓
github.com/your-org/memes/blob/main/images/cat.png
```

---

## Features

* S3-compatible API
* GitHub-backed storage
* Object versioning via commits
* Works with AWS SDKs
* Works with MinIO clients
* Zero custom storage engine
* 100% transparent storage backend
* Enterprise-grade rollback opportunities

---

## Example

Create a bucket:

```bash
aws --endpoint-url http://localhost:9000 s3 mb s3://memes
```

Upload an object:

```bash
aws --endpoint-url http://localhost:9000 s3 cp cat.png s3://memes/images/cat.png
```

List objects:

```bash
aws --endpoint-url http://localhost:9000 s3 ls s3://memes/images/
```

Internally this becomes:

```txt
github.com/your-org/memes/blob/main/images/cat.png
```

---

## Reliability

Repo3 is built on top of GitHub, one of the most trusted platforms in software engineering.

This means your objects benefit from:

* Distributed infrastructure
* Redundant storage
* Commit history
* Enterprise operations
* Absolutely no concerning rollback incidents whatsoever

Your data is safe.*

* Safe is defined emotionally, not legally.

---

## Performance

Repo3 delivers acceptable performance for workloads including:

* Memes
* Side projects
* JPEG archival
* Chaos engineering
* Extremely questionable architecture experiments
* “It would be funny if this actually worked”

Not recommended for:

* Production databases
* Large media workloads
* Compliance-sensitive environments
* Any system described as “mission critical”
* Human civilization infrastructure

---

## Running

```bash
repo3 serve \
  --github-token $GITHUB_TOKEN \
  --owner your-org \
  --addr :9000
```

---

## Compatibility

Repo3 aims to support:

* AWS SDKs
* AWS CLI
* MinIO client
* Basic S3 tooling
* Theoretical enterprise adoption

---

## Roadmap

* Multipart uploads
* Presigned URLs
* GitLab backend
* Gitea backend
* Local git backend
* Object lifecycle policies
* Glacier-equivalent cold storage (archived repositories)
* Multi-region replication (multiple GitHub organizations)
* SOC2-looking diagrams

---

## FAQ

### Is this production ready?

No.

### Should I store critical infrastructure assets in this?

Also no.

### Is it technically hilarious?

Yes.

### Could this accidentally become useful?

Unfortunately, yes.

---

## License

MIT

Use responsibly, irresponsibly, or academically.
