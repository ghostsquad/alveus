<h1 align="center">
  <br>
  <a href="http://github.com/ghostsquad/alveus"><img src="./docs/assets/river.png" alt="github.com/ghostsquad/alveus" width="200px" /></a>
  <br>
  Alveus
  <br>
</h1>

<p align="center">
  <a href="#introduction">Introduction</a> •
  <a href="#getting-started">Getting Started</a> •
  <a href="#contributing">Contributing</a> •
</p>

## Introduction

> alveus (_plural_ alvei)
>
>    1. (_rare, now usually law_) The bed or channel of a river, especially when the river flowing in its natural or ordinary course; (also) the trough of the sea.
>    2. (_neuroanatomy_) A thin layer of medullary nerve fibers on the ventricular surface of the hippocampus.

Generates GitHub workflows to allow for progressive delivery of Kubernetes resources across environments.

This initial version of Alveus uses GitHub actions as it's "execution platform", and ArgoCD to manage state.

## Getting Started

Scenario: Staging -> Prod
Facts:
- The name of the service you wish to deploy is called `podinfo`
- You have 2 distinct clusters to deploy to (staging & prod)
- You have 1 ArgoCD instance (overseeing both staging & prod clusters)
- You have 1 Service to manage

```yaml
name: podinfo
destinationGroups:
- name: mock-staging
  destinations:
  - name: in-cluster
- name: mock-prod
  needs: [mock-staging]
  destinations:
  - name: in-cluster
destinationNamespace: podinfo
argoCD:
  source:
    path: .alveus/demo/manifests
  loginCommandArgs:
    - e02a16d5f0fa.ngrok-free.app
    - --username admin
    - --password ${{ secrets.ARGOCD_ADMIN_PASSWORD }}
    - --insecure
  extraArgs:
    - --insecure
github:
  "on":
    push:
      paths:
      - .alveus/demo/manifests
      branches:
      - main
  preDeploySteps:
    - uses: jdx/mise-action@v3
      with:
        version: 2025.10.17
    - name: create-kube-config
      run: |
        mkdir -p ~/.kube
        echo "${{ secrets.KUBE_CONFIG_B64 }}" | base64 -d > ~/.kube/config

```

## Contributing

TBD

## Attribution

<a href="https://www.flaticon.com/free-icons/river" title="river icons">River icons created by Freepik - Flaticon</a>
