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

Workflows and documentation for a GitOps approach to deploying directly from Github to ArgoCD.

## Getting Started

https://argo-cd.readthedocs.io/en/stable/operator-manual/user-management/github-actions/#using-dex

Edit the argocd-cm and configure the dex.config section:

```yaml
dex.config: |
  connectors:
    - type: oidc
      id: github-actions
      name: GitHub Actions
      config:
        issuer: https://token.actions.githubusercontent.com/
        # If using GitHub Enterprise Server, then use this issuer:
        #issuer: https://github.example.com/_services/token
        scopes: [openid]
        userNameKey: sub
        insecureSkipEmailVerified: true

```

When using ArgoCD v3.0.0 or later, then you define your policy.csv like so:

```yaml
configs:
  rbac:
    policy.csv: |
      p, repo:my-org/my-repo:pull_request, projects, get, my-project, allow
      p, repo:my-org/my-repo:pull_request, applications, get, my-project/*, allow
      p, repo:my-org/my-repo:pull_request, applicationsets, get, my-project/*, allow
```

### Example GitHub Subject Claims

#### Environment

The subject claim includes the environment name when the job references an environment.

Syntax: `repo:ORG-NAME/REPO-NAME:environment:ENVIRONMENT-NAME`

Example: `repo:octo-org/octo-repo:environment:Production`

#### Pull Request

The subject claim includes the pull_request string when the workflow is triggered by a pull request event,
but only if the job doesn't reference an environment.

Syntax: `repo:ORG-NAME/REPO-NAME:pull_request`

Example: `repo:octo-org/octo-repo:pull_request`

#### Branch

The subject claim includes the branch name of the workflow,
but only if the job doesn't reference an environment, 
and if the workflow is not triggered by a pull request event.

Syntax: `repo:ORG-NAME/REPO-NAME:ref:refs/heads/BRANCH-NAME`

Example: `repo:octo-org/octo-repo:ref:refs/heads/demo-branch`

#### Tag

The subject claim includes the tag name of the workflow, 
but only if the job doesn't reference an environment, 
and if the workflow is not triggered by a pull request event.

Syntax: `repo:ORG-NAME/REPO-NAME:ref:refs/tags/TAG-NAME`

Example: `repo:octo-org/octo-repo:ref:refs/tags/demo-tag`

#### Filtering for metadata containing `:`

Any `:` within the metadata values will be replaced with `%3A` in the subject claim.

You can configure a subject that includes metadata containing colons. 

In this example, the workflow run must have originated from a job that has an environment named `Production:V1`,
in a repository named octo-repo that is owned by the octo-org organization:

Syntax: `repo:ORG-NAME/REPO-NAME:environment:ENVIRONMENT-NAME`

Example: `repo:octo-org/octo-repo:environment:Production%3AV1`


More info: 
- [ArgoCD RBAC Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/rbac/)
- [How GitHub configures the `sub` field](https://docs.github.com/en/actions/reference/security/oidc#example-subject-claims)

## Contributing

TBD

## Attribution

<a href="https://www.flaticon.com/free-icons/river" title="river icons">River icons created by Freepik - Flaticon</a>
