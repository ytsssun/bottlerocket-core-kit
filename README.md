# Bottlerocket OS

Welcome to Bottlerocket!

Bottlerocket is a free and open-source Linux-based operating system meant for hosting containers.

If you’re ready to jump right in, read our [QUICKSTART](QUICKSTART.md) to try Bottlerocket in an Amazon EKS cluster.

Bottlerocket focuses on security and maintainability, providing a reliable, consistent, and safe platform for container-based workloads.
This is a reflection of what we've learned building operating systems and services at Amazon.
You can read more about what drives us in [our charter](CHARTER.md).

The base operating system has just what you need to run containers reliably, and is built with standard open-source components.
Bottlerocket-specific additions focus on reliable updates and on the API.
Instead of making configuration changes manually, you can change settings with an API call, and these changes are automatically migrated through updates.

Some notable features include:
* [API access](#api) for configuring your system, with secure out-of-band [access methods](#exploration) when you need them.
* [Updates](#updates) based on partition flips, for fast and reliable system updates.
* [Modeled configuration](#settings) that's automatically migrated through updates.
* [Security](#security) as a top priority.

## Contact us

If you find a security issue, please [contact our security team](https://github.com/bottlerocket-os/bottlerocket/security/policy) rather than opening an issue.

If you're interested in contributing, thank you!
Please see our [contributor's guide](CONTRIBUTING.md).

We use GitHub issues to track other bug reports and feature requests.
You can select from a few templates and get some guidance on the type of information that would be most helpful.

[Contact us with a new issue here.](https://github.com/bottlerocket-os/bottlerocket/issues/new/choose)

We don't have other communication channels set up quite yet, but don't worry about making an issue!
You can let us know about things that seem difficult, or even ways you might like to help.

## Variants

To start, we're focusing on use of Bottlerocket as a host OS in AWS EKS Kubernetes clusters.
We’re excited to get early feedback and to continue working on more use cases!

Bottlerocket is architected such that different cloud environments and container orchestrators can be supported in the future.
A build of Bottlerocket that supports different features or integration characteristics is known as a 'variant'.
The artifacts of a build will include the architecture and variant name.
For example, an `x86_64` build of the `aws-k8s-1.15` variant will produce an image named `bottlerocket-aws-k8s-1.15-x86_64-<version>-<commit>.img`.

Our first supported variant, `aws-k8s-1.15`, supports EKS as described above.

## Setup

:walking: :running:

To build your own Bottlerocket images, please see [BUILDING](BUILDING.md).
It describes:
* how to build an image
* how to register an EC2 AMI from an image

To get started using Bottlerocket, please see [QUICKSTART](QUICKSTART.md).
It describes:
* how to set up a Kubernetes cluster, so your Bottlerocket instance can run pods
* how to launch a Bottlerocket instance in EC2

## Exploration

To improve security, there's no SSH server in a Bottlerocket image, and not even a shell.

Don't panic!

There are a couple out-of-band access methods you can use to explore Bottlerocket like you would a typical Linux system.
Either option will give you a shell within Bottlerocket.
From there, you can [change settings](#settings), manually [update Bottlerocket](#updates), debug problems, and generally explore.

### Control container

Bottlerocket has a ["control" container](https://github.com/bottlerocket-os/bottlerocket-control-container), enabled by default, that runs outside of the orchestrator in a separate instance of containerd.
This container runs the [AWS SSM agent](https://github.com/aws/amazon-ssm-agent) that lets you run commands, or start shell sessions, on Bottlerocket instances in EC2.
(You can easily replace this control container with your own just by changing the URI; see [Settings](#settings).)

You need to give your instance the SSM role for this to work; see the [setup guide](QUICKSTART.md#enabling-ssm).

Once the instance is started, you can start a session:

* Go to AWS SSM's [Session Manager](https://console.aws.amazon.com/systems-manager/session-manager/sessions)
* Select “Start session” and choose your Bottlerocket instance
* Select “Start session” again to get a shell

If you prefer a command-line tool, you can start a session with a recent [AWS CLI](https://aws.amazon.com/cli/) and the [session-manager-plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html).
Then you'd be able to start a session using only your instance ID, like this:

```
aws ssm start-session --target INSTANCE_ID
```

With the [default control container](https://github.com/bottlerocket-os/bottlerocket-control-container), you can make API calls to change settings in your Bottlerocket host.
To do even more, read the next section about the [admin container](#admin-container).

### Admin container

Bottlerocket has an [administrative container](https://github.com/bottlerocket-os/bottlerocket-admin-container), disabled by default, that runs outside of the orchestrator in a separate instance of containerd.
This container has an SSH server that lets you log in as `ec2-user` using your EC2-registered SSH key.
(You can easily replace this admin container with your own just by changing the URI; see [Settings](#settings).

To enable the container, you can change the setting in user data when starting Bottlerocket, for example EC2 instance user data:

```
[settings.host-containers.admin]
enabled = true
```

If Bottlerocket is already running, you can enable the admin container from the default [control container](#control-container) like this:

```
enable-admin-container
```

If you're using a custom control container, or want to make the API calls directly, you can enable the admin container like this instead:

```
apiclient -u /settings -m PATCH -d '{"host-containers": {"admin": {"enabled": true}}}'
apiclient -u /tx/commit_and_apply -m POST
```

Once you're in the admin container, you can run `sheltie` to get a full root shell in the Bottlerocket host.
Be careful; while you can inspect and change even more as root, Bottlerocket's filesystem and dm-verity setup will prevent most changes from persisting over a restart - see [Security](#security).

## Updates

Rather than a package manager that updates individual pieces of software, Bottlerocket downloads a full filesystem image and reboots into it.
It can automatically roll back if boot failures occur, and workload failures can trigger manual rollbacks.

Currently, you can update using a CLI tool, updog.
Here's how you can see whether there's an update:

```
updog check-update
```

Here's how you initiate an update:

```
updog update
reboot
```

(If you know what you're doing and want to update *now*, you can run `updog update --reboot --now`)

The system will automatically roll back if it's unable to boot.
If the update is not functional for a given container workload, you can do a manual rollback:

```
signpost rollback-to-inactive
reboot
```

We're working on more automated update methods.

The update process uses images secured by [TUF](https://theupdateframework.github.io/).
For more details, see the [update system documentation](sources/updater/).

## Settings

Here we'll describe the settings you can configure on your Bottlerocket instance, and how to do it.

(API endpoints are defined in our [OpenAPI spec](sources/api/openapi.yaml) if you want more detail.)

### Interacting with settings

#### Using the API client

You can see the current settings with an API request:
```
apiclient -u /settings
```

This will return all of the current settings in JSON format.
For example, here's an abbreviated response:
```
{"motd":"...", {"kubernetes": ...}}
```

You can change settings by sending back the same type of JSON data in a PATCH request.
This can include any number of settings changes.
```
apiclient -m PATCH -u /settings -d '{"motd": "my own value!"}'
```

This will *stage* the setting in a "pending" area - a transaction.
You can see all your pending settings like this:
```
apiclient -u /tx
```

To *commit* the settings, and let the system apply them to any relevant configuration files or services, do this:
```
apiclient -m POST -u /tx/commit_and_apply
```

Behind the scenes, these commands are working with the "default" transaction.
This keeps the interface simple.
System services use their own transactions, so you don't have to worry about conflicts.
For example, there's a "bottlerocket-launch" transaction used to coordinate changes at startup.

If you want to group sets of changes yourself, pick a transaction name and append a `tx` parameter to the URLs above.
For example, if you want the name "FOO", you can `PATCH` to `/settings?tx=FOO` and `POST` to `/tx/commit_and_apply?tx=FOO`.
(Transactions are created automatically when used, and are cleaned up on reboot.)

For more details on using the client, see the [apiclient documentation](sources/api/apiclient/).

#### Using user data

If you know what settings you want to change when you start your Bottlerocket instance, you can send them in the user data.

In user data, we structure the settings in TOML form to make things a bit simpler.
Here's the user data to change the message of the day setting, as we did in the last section:

```
[settings]
motd = "my own value!"
```

### Description of settings

Here we'll describe each setting you can change.

**Note:** You can see the default values (for any settings that are not generated at runtime) by looking at [defaults.toml](sources/models/defaults.toml).

When you're sending settings to the API, or receiving settings from the API, they're in a structured JSON format.
This allows allow modification of any number of keys at once.
It also lets us ensure that they fit the definition of the Bottlerocket data model - requests with invalid settings won't even parse correctly, helping ensure safety.

Here, however, we'll use the shortcut "dotted key" syntax for referring to keys.
This is used in some API endpoints with less-structured requests or responses.
It's also more compact for our needs here.

In this format, "settings.kubernetes.cluster-name" refers to the same key as in the JSON `{"settings": {"kubernetes": {"cluster-name": "value"}}}`.

#### Top-level settings

* `settings.motd`: This setting is just written out to /etc/motd. It's useful as a way to get familiar with the API!  Try changing it.

#### Kubernetes settings

The following settings must be specified in order to join a Kubernetes cluster.
You should [specify them in user data](#using-user-data).
See the [setup guide](QUICKSTART.md) for *much* more detail on setting up Bottlerocket and Kubernetes.
* `settings.kubernetes.cluster-name`: The cluster name you chose during setup; the [setup guide](QUICKSTART.md) uses "bottlerocket".
* `settings.kubernetes.cluster-certificate`: This is the base64-encoded certificate authority of the cluster.
* `settings.kubernetes.api-server`: This is the cluster's Kubernetes API endpoint.

The following settings can be optionally set to customize the node labels and taints. 
* `settings.kubernetes.node-labels`: [Labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) in the form of key, value pairs added when registering the node in the cluster.
* `settings.kubernetes.node-taints`: [Taints](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/) in the form of key, value and effect entries added when registering the node in the cluster.
  * Example user data for setting up labels and taints:
    ```
    [settings.kubernetes.node-labels]
    label1 = "foo"
    label2 = "bar"
    [settings.kubernetes.node-taints]
    dedicated = "experimental:PreferNoSchedule"
    special = "true:NoSchedule"
    ```

The following settings are set for you automatically by [pluto](sources/api/) based on runtime instance information, but you can override them if you know what you're doing!
* `settings.kubernetes.max-pods`: The maximum number of pods that can be scheduled on this node (limited by number of available IPv4 addresses)
* `settings.kubernetes.cluster-dns-ip`: The CIDR block of the primary network interface.
* `settings.kubernetes.node-ip`: The IPv4 address of this node.
* `settings.kubernetes.pod-infra-container-image`: The URI of the "pause" container.

#### Updates settings

* `settings.updates.metadata-base-url`: The common portion of all URIs used to download update metadata.
* `settings.updates.targets-base-url`: The common portion of all URIs used to download update files.
* `settings.updates.seed`: A `u32` value that determines how far into in the update schedule this machine will accept an update.  We recommending leaving this at its default generated value so that updates can be somewhat randomized in your cluster.

#### Time settings

* `settings.ntp.time-servers`: A list of NTP servers used to set and verify the system time.

#### Host containers settings
* `settings.host-containers.admin.source`: The URI of the [admin container](#admin-container).
* `settings.host-containers.admin.enabled`: Whether the admin container is enabled.
* `settings.host-containers.admin.superpowered`: Whether the admin container has high levels of access to the Bottlerocket host.
* `settings.host-containers.control.source`: The URI of the [control container](#control-container).
* `settings.host-containers.control.enabled`: Whether the control container is enabled.
* `settings.host-containers.control.superpowered`: Whether the control container has high levels of access to the Bottlerocket host.

##### Custom host containers

[`admin`](https://github.com/bottlerocket-os/bottlerocket-admin-container) and [`control`](https://github.com/bottlerocket-os/bottlerocket-control-container) are our default host containers, but you're free to change this.
Beyond just changing the settings above to affect the `admin` and `control` containers, you can add and remove host containers entirely.
As long as you define the three fields above -- `source` with a URI, and `enabled` and `superpowered` with true/false -- you can add host containers with an API call or user data.

Here's an example of adding a custom host container with API calls:
```
apiclient -u /settings -X PATCH -d '{"host-containers": {"custom": {"source": "MY-CONTAINER-URI", "enabled": true, "superpowered": false}}}'
apiclient -u /tx/commit_and_apply -X POST
```

Here's the same example, but with the settings you'd add to user data:
```
[settings.host-containers.custom]
enabled = true
source = "MY-CONTAINER-URI"
superpowered = false
```

If the `enabled` flag is `true`, it will be started automatically.

All host containers will have the `apiclient` binary available at `/usr/local/bin/apiclient` so they're able to [interact with the API](#using-the-api-client).

In addition, all host containers come with persistent storage at `/.bottlerocket/host-containers/$HOST_CONTAINER_NAME` that is persisted across reboots and container start/stop cycles.
The default `admin` host-container, for example, store its SSH host keys under `/.bottlerocket/host-containers/admin/etc/ssh/`.

There are a few important caveats to understand about host containers:
* They're not orchestrated.  They only start or stop according to that `enabled` flag.
* They run in a separate instance of containerd than the one used for orchestrated containers like Kubernetes pods.
* They're not updated automatically.  You need to update the `source`, disable the container, commit those changes, then re-enable it.
* If you set `superpowered` to true, they'll essentially have root access to the host.

Because of these caveats, host containers are only intended for special use cases.
We use it for the control container because it needs to be available early to give you access to the OS, and we use it for the admin container because it needs high levels of privilege and because you need it to debug when orchestration isn't working.

Be careful, and make sure you have a similar low-level use case before reaching for host containers.

## Details

### Security

We use [dm-verity](https://gitlab.com/cryptsetup/cryptsetup/wikis/DMVerity) to load a verified read-only root filesystem, preventing some classes of persistent security threats.
Only a few locations are made writable:
* some through [tmpfs mounts](sources/preinit/laika), used for configuration, that don't persist over a restart.
* one [persistent location](packages/release/var-lib-bottlerocket.mount) for the data store.

We enable [SELinux](https://selinuxproject.org/) in enforcing mode.
This protects the data store from tampering, and blocks modification of sensitive files such as container archives.

Almost all first-party components are written in [Rust](https://www.rust-lang.org/).
Rust eliminates some classes of memory safety issues, and encourages design patterns that help security.

### Packaging

Bottlerocket is built from source using a container toolchain.
We use RPM package definitions to build and install individual packages into an image.
RPM itself is not in the image - it's just a common and convenient package definition format.

We currently package the following major third-party components:
* Linux kernel ([background](https://en.wikipedia.org/wiki/Linux), [packaging](packages/kernel/))
* glibc ([background](https://www.gnu.org/software/libc/), [packaging](packages/glibc/))
* Buildroot as build toolchain ([background](https://buildroot.org/), via the [SDK](https://github.com/bottlerocket-os/bottlerocket-sdk))
* GRUB, with patches for partition flip updates ([background](https://www.gnu.org/software/grub/), [packaging](packages/grub/))
* systemd as init ([background](https://en.wikipedia.org/wiki/Systemd), [packaging](packages/systemd/))
* wicked for networking ([background](https://github.com/openSUSE/wicked), [packaging](packages/wicked/))
* containerd ([background](https://containerd.io/), [packaging](packages/containerd/))
* Kubernetes ([background](https://kubernetes.io/), [packaging](packages/kubernetes/))
* aws-iam-authenticator ([background](https://github.com/kubernetes-sigs/aws-iam-authenticator), [packaging](packages/aws-iam-authenticator/))

For further documentation or to see the rest of the packages, see the [packaging directory](packages/).

### Updates

The Bottlerocket image has two identical sets of partitions, A and B.
When updating Bottlerocket, the partition table is updated to point from set A to set B, or vice versa.

We also track successful boots, and if there are failures it will automatically revert back to the prior working partition set.

The update process uses images secured by [TUF](https://theupdateframework.github.io/).
For more details, see the [update system documentation](sources/updater/).

### API

There are two main ways you'd interact with a production Bottlerocket instance.
(There are a couple more [exploration](#exploration) methods above for test instances.)

The first method is through a container orchestrator, for when you want to run or manage containers.
This uses the standard channel for your orchestrator, for example a tool like `kubectl` for Kubernetes.

The second method is through the Bottlerocket API, for example when you want to configure the system.

There's an HTTP API server that listens on a local Unix-domain socket.
Remote access to the API requires an authenticated transport such as SSM's RunCommand or Session Manager, as described above.
For more details, see the [apiserver documentation](sources/api/apiserver/).

The [apiclient](sources/api/apiclient/) can be used to make requests.
They're just HTTP requests, but the API client simplifies making requests with the Unix-domain socket.

To make configuration easier, we have [early-boot-config](sources/api/early-boot-config/), which can send an API request for you based on instance user data.
If you start a virtual machine, like an EC2 instance, it will read TOML-formatted Bottlerocket configuration from user data and send it to the API server.
This way, you can configure your Bottlerocket instance without having to make API calls after launch.

See [Settings](#settings) above for examples and to understand what you can configure.

The server and client are the user-facing components of the API system, but there are a number of other components that work together to make sure your settings are applied, and that they survive upgrades of Bottlerocket.

For more details, see the [API system documentation](sources/api/).
