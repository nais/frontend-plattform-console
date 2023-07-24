#!/usr/bin/env bash

unleash_namespace="bifrost-unleash"
operator_namespace="nais-system"

function help() {
  echo "Sync unleash instance config from management to tenant"
  echo "-----------------------------------------------------"
  echo "Usage: $0 <command>"
  echo "Commands:"
  echo "  help: show this help"
  echo "  list: list all unleash instances in management"
  echo "  copy <unleash>: copy unleash instance from management"
  echo "  paste <unleash>: paste unleash instance to tenant"
}

function list() {
  echo "Listing unleash instances in management..."

  kubectl get unleash -n $unleash_namespace
}

function copy() {
  if [ -z "$2" ]; then
    echo "Error: Missing unleash instance name"
    echo ""
    help
    exit 1
  fi

  unleash_name=$2

  echo "Check if unleash instance $unleash_name exists in $unleash_namespace namespace..."
  kubectl get unleash $unleash_name -n $unleash_namespace > /dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "Error: Unleash instance $unleash_name not found in $unleash_namespace namespace"
    echo "Make sure you are on the right tenant and that the instance exists."
    exit 1
  fi

  unleash_name=$2
  secret_name="unleasherator-$unleash_namespace-$unleash_name-admin-key"
  random_string=$(openssl rand -base64 10 | tr -dc 'a-zA-Z0-9' | tr '[:upper:]' '[:lower:]')
  new_secret_name="unleasherator-$unleash_name-admin-key-$random_string"

  echo "Creating temporary directory $tmp_dir..."
  tmp_dir="./tmp-$unleash_name"
  tmp_secret_file="$tmp_dir/$new_secret_name.yaml"
  tmp_remoteunleash_file="$tmp_dir/remoteunleash-$unleash_name.yaml"
  if [ -d "$tmp_dir" ]; then
    echo "Error: Temporary directory $tmp_dir already exists"
    exit 1
  fi
  mkdir -p $tmp_dir

  echo "Creating copy of unleash instance operator secret $secret_name..."
  kubectl get secret $secret_name -n $operator_namespace -o yaml \
    | yq ".metadata.name = \"$new_secret_name\" \
      | del(.metadata.creationTimestamp) \
      | del(.metadata.resourceVersion) \
      | del(.metadata.uid) \
      | del(.data.INIT_ADMIN_API_TOKENS) \
      | del(.metadata.annotations)" > $tmp_secret_file

  echo "Creating RemoteUnleash resource for unleash instance $unleash_name..."
  unleash_api_host=$(kubectl get unleash $unleash_name -n $unleash_namespace -o yaml \
    | yq .spec.apiIngress.host)

  echo "apiVersion: unleash.nais.io/v1
kind: RemoteUnleash
metadata:
  name: $unleash_name
  namespace: $unleash_name
spec:
  unleashInstance:
    url: https://$unleash_api_host
  adminSecret:
    name: $new_secret_name
    namespace: $operator_namespace" > $tmp_remoteunleash_file

  echo "Check the following files:"
  echo "cat $tmp_secret_file"
  echo "cat $tmp_remoteunleash_file"
  echo ""
  echo "If everything looks good, run the following commands for each tenant cluster:"
  echo "kubectl apply -f $tmp_secret_file -n $operator_namespace"
  echo "kubectl apply -f $tmp_remoteunleash_file -n $unleash_name"
}

function paste() {
  echo "Pasting files to tenant..."

  if [ -z "$2" ]; then
    echo "Missing unleash instance name"
    exit 1
  fi

  unleash=$2
}

function main() {
  case "$1" in
    "help")
      help
      ;;
    "list")
      list $@
      ;;
    "copy")
      copy $@
      ;;
    "paste")
      paste $@
      ;;
    *)
      echo "Unknown command"
      help
      ;;
  esac
}

main $@