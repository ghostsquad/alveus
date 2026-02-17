#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

MYTMPDIR="$(mktemp -d)"
trap '{ rm -rf -- "$MYTMPDIR"; }' EXIT

set -x

kubectl create namespace argocd
kubectl config set-context --current --namespace=argocd
curl -sSL https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml -o $MYTMPDIR/install.yaml

dex_config=$(cat <<EOF
connectors:
  - type: oidc
    id: github-actions
    name: GitHub Actions
    config:
      issuer: https://token.actions.githubusercontent.com/
      scopes: [openid]
      userNameKey: sub
      insecureSkipEmailVerified: true
EOF
)

yq -i '(select(.metadata.name == "argocd-cm") | .data."dex.config") = strenv(dex_config)' $MYTMPDIR/install.yaml

policy_csv=$(cat <<EOF
p, repo:ghostsquad/alveus:pull_request, applications, *, podinfo-demo/*, allow
EOF
)

yq -i '(select(.metadata.name == "argocd-rbac-cm") | .data."policy.csv") = strenv(policy_csv)' $MYTMPDIR/install.yaml

kubectl apply -n argocd --server-side --force-conflicts -f $MYTMPDIR/install.yaml
kubectl apply -n argocd --server-side --force-conflicts -f "${SCRIPT_DIR}/project.yml"

start=$EPOCHSECONDS
while [ "$(kubectl get pods -o jsonpath='{.items[*].status.containerStatuses[0].ready}')" != "true" ]; do
    if (( EPOCHSECONDS-start > 60 )); then
      echo "Timed out waiting for ArgoCD to be ready"
      exit 1
    fi
    sleep 5
    echo "Waiting for ArgoCD to be ready."
done

SERVICE_NAME=""
SERVICE_NAMESPACE="argocd"
# The port on the pod that your application is listening on
SERVICE_PORT="80"
# The local port you want to use to access the pod
LOCAL_PORT="8080"
echo "Attempting to port-forward to service '${SERVICE_NAME}' in namespace '${SERVICE_NAMESPACE}' on cluster '${KIND_CLUSTER_NAME}'..."

# Start port-forwarding in the background
# The --kubeconfig flag might be needed if your kubeconfig is not in the default location
kubectl port-forward svc/"${SERVICE_NAME}" -n "${SERVICE_NAMESPACE}" "${LOCAL_PORT}:${SERVICE_PORT}" &

# Give the port-forward process a moment to establish
sleep 5

echo "Port-forwarding established..."
