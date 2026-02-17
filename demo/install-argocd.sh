#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

MYTMPDIR="$(mktemp -d)"
trap '{ rm -rf -- "$MYTMPDIR"; }' EXIT

set -x

kubectl create namespace argocd
kubectl config set-context --current --namespace=argocd
curl -sSL https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml -o $MYTMPDIR/install.yaml

export DEX_CONFIG=$(cat <<EOF
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

yq -i '(select(.metadata.name == "argocd-cm") | .data."dex.config") = strenv("DEX_CONFIG")' $MYTMPDIR/install.yaml

echo "::group::DEBUG argocd-cm"
yq 'select(.metadata.name == "argocd-cm") | .' $MYTMPDIR/install.yaml
echo "::endgroup::"

export POLICY_CSV=$(cat <<EOF
p, repo:ghostsquad/alveus:pull_request, applications, action/*, *, allow
p, repo:ghostsquad/alveus:ref:refs/heads/main, applications, action/*, *, allow
EOF
)

yq -i '(select(.metadata.name == "argocd-rbac-cm") | .data."policy.csv") = strenv("POLICY_CSV")' $MYTMPDIR/install.yaml

echo "::group::DEBUG argocd-rbac-cm"
yq 'select(.metadata.name == "argocd-rbac-cm") | .' $MYTMPDIR/install.yaml
echo "::endgroup::"

kubectl apply -n argocd --server-side --force-conflicts -f $MYTMPDIR/install.yaml

# let the CRDs finish installing
sleep 5

kubectl apply -n argocd --server-side --force-conflicts -f "${SCRIPT_DIR}/project.yml"

start=$EPOCHSECONDS
while ! [[ "$(kubectl get pods -o jsonpath='{.items[*].status.containerStatuses[0].ready}')" =~ ^(true ?)+$ ]]; do
    if (( EPOCHSECONDS-start > 60 )); then
      echo "Timed out waiting for ArgoCD to be ready"
      exit 1
    fi
    sleep 5
    echo "Waiting for ArgoCD to be ready."
done

SERVICE_NAME="argocd-server"
SERVICE_NAMESPACE="argocd"
SERVICE_PORT="443"
# The local port you want to use to access the service
LOCAL_PORT="8080"
PORT_FORWARD_LOG="$PORT_FORWARD_TEMP_DIR/k8s-port-forward.$RANDOM.log"

echo "Attempting to port-forward ${LOCAL_PORT}:${SERVICE_PORT} to service '${SERVICE_NAME}' in namespace '${SERVICE_NAMESPACE}' on cluster '${KIND_CLUSTER_NAME}'..."
# Start port-forwarding in the background
# The --kubeconfig flag might be needed if your kubeconfig is not in the default location
nohup kubectl port-forward svc/"${SERVICE_NAME}" -n "${SERVICE_NAMESPACE}" "${LOCAL_PORT}:${SERVICE_PORT}" < /dev/null &> "$PORT_FORWARD_LOG" &

# Give the port-forward process a moment to establish
sleep 5

echo "port-forward-log=${PORT_FORWARD_LOG}" >> $GITHUB_OUTPUT

echo "Port-forwarding ${LOCAL_PORT}:${SERVICE_PORT} established..."
