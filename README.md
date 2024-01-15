# Garm External Provider For AWS

The AWS external provider allows [garm](https://github.com/cloudbase/garm) to create Linux and Windows runners on top of AWS virtual machines.

## Build

Clone the repo:

```bash
git clone https://github.com/cloudbase/garm-provider-aws
```

Build the binary:

```bash
cd garm-provider-aws
go build .
```

Copy the binary on the same system where garm is running, and [point to it in the config](https://github.com/cloudbase/garm/blob/main/doc/providers.md#the-external-provider).

## Configure

The config file for this external provider is a simple toml used to configure the AWS credentials it needs to spin up virtual machines.

```bash
region = "eu-central-1"
subnet_id = "sample_subnet_id"
availability_zone = "sample_availability_zone"

[credentials]
    access_key_id = "sample_access_key_id"
    secret_access_key = "sample_secret_access_key"
    session_token = "sample_session_token"
```

## Creating a pool

After you [add it to garm as an external provider](https://github.com/cloudbase/garm/blob/main/doc/providers.md#the-external-provider), you need to create a pool that uses it. Assuming you named your external provider as ```aws``` in the garm config, the following command should create a new pool:

```bash
garm-cli pool create \
    --os-type linux \
    --os-arch amd64 \
    --enabled=true \
    --flavor t2.small \
    --image ami-04c0bb88603bf2e3d \
    --min-idle-runners 0 \
    --repo 5b4f2fb0-3485-45d6-a6b3-545bad933df3 \
    --tags aws,ubuntu \
    --provider-name aws
```

```bash
garm-cli pool create \
    --os-type windows \
    --os-arch amd64 \
    --enabled=true \
    --flavor t2.small \
    --image ami-0d5f36b04ca291a9f \
    --min-idle-runners 0 \
    --repo 5b4f2fb0-3485-45d6-a6b3-545bad933df3 \
    --tags aws,windows \
    --provider-name aws
```