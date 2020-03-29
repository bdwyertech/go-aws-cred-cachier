# AWS Credential Cachier

The goal of this project is to cache credentials locally by way of `credential_process`.

When running processes on large instances with high concurrency across many accounts, (e.g. Cloud Custodian), I found the AWS Instance Metadata Service to be incapable of servicing the barrage of requests.  Rather than set static credentials as environment variables prior to execution, I chose to write this utility, which is a functional `credential_helper` process.  It behaves as a sort of caching proxy for the AWS credential retrieval process, intended to rate limit requests to IMDS endpoints.

Credentials are cached based on request checksum.  The checksum is a hash of AWS environment variables (`AWS_*`) and supplied arguments.

A simple loop prevention mechanism is implemented whereby each process sets an environment variable to the invocations Request Checksum.  If this variable is set and matches the current invocations checksum, it assumes it is a loop and errors.

### Example Usage
```ini
[default]
region = us-east-1

[profile cached]
credential_process = /usr/local/bin/aws-cred-cachier -disable-shared-config

[profile my_cool_role]
role_arn = arn:aws:iam::123456789876:role/MyCoolRole
source_profile = cached
```
