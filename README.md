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
