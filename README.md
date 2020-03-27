# AWS Credential Cachier

The goal of this project is to cache credentials locally by way of `credential_process`.

When running processes on large instances with high concurrency across many accounts, (e.g. Cloud Custodian), I found the AWS Instance Metadata Service to be incapable of servicing the barrage of requests.  Rather than set static credentials as environment variables prior to execution, I chose to write this utility, which is a functional `credential_helper` process.  It behaves as a sort of caching proxy for the AWS credential retrieval process, intended to rate limit requests to IMDS endpoints.

### Note
You may wish to explicity disable shared config, especially if you are leveraging custom profiles via `AWS_PROFILE`.  You can otherwise create a circular dependency (fork bomb)!

### Example Usage
```
[default]
region = us-east-1

[profile cached]
credential_process = /usr/local/bin/aws-cred-cachier -disable-shared-config

[profile my_cool_role]
role_arn = arn:aws:iam::123456789876:role/MyCoolRole
source_profile = cached
```
