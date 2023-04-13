# Bifröst

Bifröst is a portal for managing Frontend Platform resources. In Norse mythology, Bifröst is a bridge that connects Midgard, the realm of humans, with Asgard, the realm of the gods. At NAV Bifröst is a bridge that connects developers with the resources they need to build and run their frontend applications.

## Features

* [x] Manage Unleash Instances

## Pre-requisites

### Google Clooud Service Account

Bifröst needs a Google Cloud service account with the following roles:

* Cloud SQL Admin

### Google Cloud Resources

Bifröst needs the following Google Cloud resources:

* A Google Cloud PostgreSQL instance for Unleash databases

## Configuration

Bifröst is configured using environment variables. The following variables are required:

| Variable | Description |
| -------- |  ------- |
| `BIFROST_GOOGLE_PROJECT_ID` | The Google Cloud project ID |
| `BIFROST_UNLEASH_SQL_INSTANCE_ID` | The SQL instance ID for Unleash databases |

## Local development

### Prerequisite

* Google Cloud Service Account
* Local Kubernets Cluster
  * Unleasherator CRD
  * FQDNNetworkPolicy CRD

Apply the required custom resource definitions:

```bash
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/gke-fqdnnetworkpolicies-golang/main/config/crd/bases/networking.gke.io_fqdnnetworkpolicies.yaml
kubectl apply -f https://raw.githubusercontent.com/nais/unleasherator/main/config/crd/bases/unleash.nais.io_unleashes.yaml
```

### Enviornment variables

The following environment variables needs to be set:

| Variable | Value | Description |
| -------- |  ---- | ----------- |
| `BIFROST_HOST` | `127.0.0.1` | |
| `BIFROST_GOOGLE_PROJECT_ID` | The Google Cloud project ID |
| `BIFROST_UNLEASH_SQL_INSTANCE_ID` | The SQL instance ID for Unleash databases |
| `BIFROST_UNLEASH_INSTANCE_NAMESPACE` | `default` | |
| `GOOGLE_APPLICATION_CREDENTIALS` | <path-to-file> | |
| `KUBECONFIG` | <path-to-file> | |

### Start the server

```
make start
```
