#!/bin/bash

unleashes=$(kubectl get unleash -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')
for unleash in $unleashes; do
  echo "unleash=$unleash"

  input=$(kubectl get unleash $unleash -o yaml)

  name=$(echo "$input" | yq eval '.metadata.name')
  echo "name=$name"

  namespaces=$(echo "$input" | yq eval '.spec.federation.namespaces')
  echo "namespaces=$namespaces"

  teams_raw=$(echo "$input" | yq eval '.spec.extraEnvVars[] | select(.name == "TEAMS_ALLOWED_TEAMS") | .value')
  IFS=',' read -ra teams_array <<< "$teams_raw"
  teams_list=""
  for team in "${teams_array[@]}"; do
    teams_list+="- $team\n"
  done
  teams=$(echo -e "$teams_list")
  echo "teams=$teams"

  teams_yaml=$(echo -e "$namespaces\n$teams" | sort -u)
  echo "teams_yaml=$teams_yaml"

  teams_csv=$(echo "$teams_yaml" | sed 's/^- //g' | tr '\n' ',' | sed 's/,$//')
  echo "teams_csv=$teams_csv"

  echo "$input" | yq e '
    (.spec.extraEnvVars[] | select(.name == "TEAMS_ALLOWED_TEAMS")).value = "'"$teams_csv"'" |
    (del(.spec.extraEnvVars[] | select(.name == "TEAMS_ALLOWED_NAMESPACES" or .name == "TEAMS_ALLOWED_CLUSTERS"))) |
    .spec.customImage = "europe-north1-docker.pkg.dev/nais-io/nais/images/unleash-v4:v5.12.4-20240603-080907-61a67ee" |
    .spec.webIngress.host = .metadata.name + "-unleash-web.iap.nav.cloud.nais.io" |
    .spec.federation.namespaces = $teams_yaml
  ' > out/$name.yaml
done