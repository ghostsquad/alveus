package github

import (
	"fmt"

	"github.com/cakehappens/gocto"
	"github.com/goforj/godump"

	"github.com/ghostsquad/alveus/api/v1alpha1"
	"github.com/ghostsquad/alveus/internal/util"
)

func newDeployGroupJob(name string, wf gocto.Workflow) gocto.Job {
	workflowPath := "./" + wf.GetRelativePathAndFilename()

	job := gocto.Job{
		Name: name,
		Uses: workflowPath,
		Secrets: &gocto.Secrets{
			Inherit: true,
		},
	}

	return job
}

type newDeployJobInput struct {
	name                 string
	destination          v1alpha1.Destination
	destinationGroup     string
	checkoutCommitBranch string
	argoCDSpec           v1alpha1.ArgoCD
}

func newDeployJob(input newDeployJobInput) gocto.Job {
	const (
		EnvNameArgoCDApplicationFile = "ARGOCD_APPLICATION_FILE"
		EnvNameGitCommitMessage      = "GIT_COMMIT_MESSAGE"
	)

	name := input.name
	destination := input.destination

	destinationFriendlyName := v1alpha1.CoalesceSanitizeDestination(destination)

	steps := []gocto.Step{
		{
			Uses: "actions/checkout@v4",
			With: map[string]any{
				"ref": input.checkoutCommitBranch,
				// otherwise, the token used is the GITHUB_TOKEN, instead of your personal token
				"persist-credentials": false,
				// otherwise, you will fail to push refs to dest repo
				"fetch-depth": 0,
			},
		},
	}

	steps = append(steps, input.destination.Github.PreDeploySteps...)

	steps = append(steps,
		gocto.Step{
			Name: "git-config",
			Run: util.SprintfDedent(`
					git config --global user.name '${{ github.actor }}'
					git config --global user.email '${{ github.actor }}@users.noreply.github.com'
				`),
		},
		gocto.Step{
			Uses: "frenck/action-setup-yq@v1",
		},
		gocto.Step{
			Name: "update-application-yaml",
			Run: util.SprintfDedent(`
					yq e -i '.spec.source.targetRevision = "${{ github.sha }}"' \
						"${%s}"
				`, EnvNameArgoCDApplicationFile),
		},
		gocto.Step{
			Name: "git-add-commit",
			Run: util.SprintfDedent(`
					git add "${%s}"
					if git diff-index --quiet HEAD -- 2>/dev/null; then
						echo "No changes to commit"
					else
						git commit -m "${%s}"
					fi
				`, EnvNameArgoCDApplicationFile, EnvNameGitCommitMessage),
		},
	)

	extraArgoCDArgsString := util.Join(" ", input.argoCDSpec.ExtraArgs...)

	steps = append(steps,
		gocto.Step{
			Name: "argocd-upsert",
			Run: util.SprintfDedent(`
					argocd app create \
						%s \
						--upsert \
						--file "${%s}" \
						--sync-policy=none \
						--prompts-enabled=false \
						;
				`, extraArgoCDArgsString, EnvNameArgoCDApplicationFile),
		},
		gocto.Step{
			Name: "argocd-sync",
			Run: util.SprintfDedent(`
					argocd app sync \
						%s \
						--timeout %d \
						--retry-limit %d \
						;
				`, extraArgoCDArgsString,
				*input.argoCDSpec.SyncTimeoutSeconds,
				*input.argoCDSpec.SyncRetryLimit,
			),
		})

	steps = append(steps, input.destination.Github.PostDeploySteps...)

	job := gocto.Job{
		Name:   name,
		RunsOn: []string{"ubuntu-latest"},
		Defaults: gocto.Defaults{
			Run: gocto.DefaultsRun{
				Shell: gocto.ShellBash,
			},
		},
		Env: map[string]string{
			EnvNameArgoCDApplicationFile: input.argoCDSpec.ApplicationFilePath,
			EnvNameGitCommitMessage: fmt.Sprintf(
				"feat(%s): 🚀 deploy to %s",
				input.destinationGroup,
				destinationFriendlyName,
			),
		},
		Steps: steps,
	}

	return job
}
