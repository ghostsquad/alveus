package github

import (
	"encoding/json"
	"fmt"

	"github.com/cakehappens/gocto"
	"github.com/samber/lo"

	"github.com/ghostsquad/alveus/api/v1alpha1"
	"github.com/ghostsquad/alveus/internal/constants"
	"github.com/ghostsquad/alveus/internal/integrations/argocd"
	"github.com/ghostsquad/alveus/internal/util"
)

func SetWorkflowFilenameWithAlveusPrefix(w gocto.Workflow) gocto.Workflow {
	filename := gocto.FilenameFor(w)
	filename = constants.Alveus + "-" + filename
	(&w).SetFilename(filename)

	return w
}

type DestinationChoice struct {
	Group     string `json:"group"`
	Cluster   string `json:"cluster"`
	Namespace string `json:"ns"`
}

func NewWorkflow(service v1alpha1.Service, apps argocd.ApplicationRepository) (gocto.Workflow, error) {
	top := gocto.Workflow{
		Name: "deploy-" + service.Name,
		On:   service.Github.On,
		Defaults: gocto.Defaults{
			Run: gocto.DefaultsRun{
				Shell: gocto.ShellBash,
			},
		},
		Jobs: make(map[string]gocto.Job),
		Env: map[string]string{
			"ARGOCD_SERVER":     "argocd.example.com",
			"ARGOCD_AUTH_TOKEN": "${{ secrets.ARGOCD_AUTH_TOKEN }}",
			"ARGOCD_OPTS":       "--grpc-web",
		},
	}

	var destinationChoices []string
	for _, group := range service.DestinationGroups {
		for _, dest := range group.Destinations {
			choice := DestinationChoice{
				Group:     group.Name,
				Cluster:   v1alpha1.CoalesceSanitizeDestination(dest),
				Namespace: dest.Namespace,
			}

			choiceJSON, err := json.Marshal(choice)
			if err != nil {
				return gocto.Workflow{}, fmt.Errorf("failed to marshal dispatch choice: %w", err)
			}

			destinationChoices = append(destinationChoices, string(choiceJSON))
		}
	}

	top.On.Dispatch = &gocto.OnDispatch{
		Inputs: map[string]gocto.OnDispatchInput{
			"destination-group": {
				Description: `the destination group to deploy (mutually exclusive with destination, default "-" for all groups)`,
				Required:    false,
				Type:        gocto.OnDispatchInputTypeChoice,
				Options: append([]string{"-"}, lo.Map(
					service.DestinationGroups, func(group v1alpha1.DestinationGroup, _ int) string {
						return group.Name
					})...,
				),
			},
			"destination": {
				Description: `the destination to deploy (mutually exclusive with destination-group, default "-" for all destinations)`,
				Required:    false,
				Type:        gocto.OnDispatchInputTypeChoice,
				Options:     append([]string{"-"}, destinationChoices...),
			},
			"revision-to-deploy": {
				Description: `the revision to deploy, default is github.sha`,
				Required:    false,
				Type:        gocto.OnDispatchInputTypeChoice,
				// TODO add more choices
				Options: []string{
					"-",
				},
			},
		},
	}

	top.Jobs["single"] = gocto.Job{
		If:          `${{ inputs.destination != '-' && inputs.destination-group == '-' }}`,
		RunsOn:      []string{"ubuntu-latest"},
		Environment: gocto.Environment{},
		Concurrency: gocto.Concurrency{},
		Outputs:     nil,
		Env:         nil,
		Defaults:    gocto.Defaults{},
		Steps: []gocto.Step{
			{
				Uses: "actions/checkout@v4",
				With: map[string]any{
					"fetch-depth":         0,
					"persist-credentials": false,
				},
			},
			// insert custom pre-deploy steps here
			{
				Name: "input-jq",
				ID:   "input-jq",
				Run: util.SprintfDedent(`
					echo "group=${{ fromJSON(inputs.destination).group }}" >> $GITHUB_OUTPUT
					echo "cluster=${{ fromJSON(inputs.destination).cluster }}" >> $GITHUB_OUTPUT
					echo "ns=${{ fromJSON(inputs.destination).ns }}" >> $GITHUB_OUTPUT
					echo "target-revision=${{ inputs.revision-to-deploy == '-' && github.sha || inputs.revision-to-deploy }}" >> $GITHUB_OUTPUT
				`),
			},
			{
				Uses: "./.github/actions/alveus-deploy",
				With: map[string]any{
					"target-revision":  `${{ steps.input-jq.outputs.target-revision }}`,
					"application-file": `./.alveus/applications/podinfo/${{ steps.input-jq.outputs.group }}/${{ steps.input-jq.outputs.cluster }}.yaml`,
					"namespace":        `${{ steps.input-jq.outputs.ns }}`,
					"git-commit-msg":   `deploy(${{ steps.input-jq.outputs.group }}): 🚀 to ${{ steps.input-jq.outputs.cluster }}`,
				},
			},
		},
	}

	for _, dg := range service.DestinationGroups {
		top.Jobs[dg.Name] = gocto.Job{
			Name:        "",
			Permissions: gocto.Permissions{},
			Needs:       dg.Needs,
			If: fmt.Sprintf(
				`${{ inputs.destination == '-' && ( inputs.destination-group == '-' || inputs.destination-group == '%s' ) }}`,
				dg.Name,
			),
			RunsOn: []string{"ubuntu-latest"},
			Strategy: gocto.Strategy{
				Matrix: &gocto.Matrix{
					Map: map[string][]gocto.StringOrInt{
						"dest": lo.Map(dg.Destinations, func(dest v1alpha1.Destination, _ int) gocto.StringOrInt {
							return gocto.StringOrInt{
								StringValue: util.Ptr(v1alpha1.CoalesceSanitizeDestination(dest)),
							}
						}),
					},
				},
			},
			Steps: []gocto.Step{
				{
					Uses: "actions/checkout@v4",
					With: map[string]any{
						"fetch-depth":         0,
						"persist-credentials": false,
					},
				},
				// insert custom pre-deploy steps here
				{
					Name: "input-jq",
					ID:   "input-jq",
					Run: util.SprintfDedent(`
						echo "target-revision=${{ inputs.revision-to-deploy == '-' && github.sha || inputs.revision-to-deploy }}" >> $GITHUB_OUTPUT
				`),
				},
				{
					Uses: "./.github/actions/alveus-deploy",
					With: map[string]any{
						"target-revision": `${{ steps.input-jq.outputs.target-revision }}`,
						"application-file": fmt.Sprintf(
							`./.alveus/applications/%s/%s/${{ matrix.dest }}.yaml`,
							service.Name,
							dg.Name,
						),
						"git-commit-msg": fmt.Sprintf(
							`deploy(%s): 🚀 to ${{ matrix.dest }}`,
							dg.Name,
						),
					},
				},
			},
		}
	}

	top = SetWorkflowFilenameWithAlveusPrefix(top)
	return top, nil
}
