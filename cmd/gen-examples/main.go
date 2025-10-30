package main

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/cakehappens/gocto"
	"github.com/goccy/go-yaml"
	"github.com/oklog/run"

	"github.com/ghostsquad/alveus/api/v1alpha1"
)

var version string

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	var g run.Group
	var err error

	g.Add(func() error {
		return createServiceFile()
	}, func(error) {
		cancel()
	})

	g.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	err = g.Run()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func createServiceFile() error {
	const manifestsPath = ".alveus/demo/manifests"

	service := &v1alpha1.Service{
		Name: "example-service",
		ArgoCD: v1alpha1.ArgoCD{
			Source: v1alpha1.Source{
				Path: manifestsPath,
			},
		},
		Github: v1alpha1.Github{
			On: gocto.WorkflowOn{
				Push: &gocto.OnPush{
					OnPaths: &gocto.OnPaths{
						Paths: []string{manifestsPath},
					},
					OnBranches: &gocto.OnBranches{
						Branches: []string{"main"},
					},
				},
			},
		},
		DestinationGroups: []v1alpha1.DestinationGroup{
			{
				Name: "staging",
				Destinations: []v1alpha1.Destination{
					{
						Server: "http://kube.local",
					},
				},
			},
		},
		DestinationNamespace: "podinfo",
	}

	serviceBytes, err := yaml.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service object: %w", err)
	}

	err = os.WriteFile("examples/example-service.yaml", serviceBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	return nil
}
