package argocd

import (
	"errors"
	"fmt"
	"strings"

	argoapisapplication "github.com/argoproj/argo-cd/v3/pkg/apis/application"
	argov1alpha1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ghostsquad/alveus/api/v1alpha1"
	"github.com/ghostsquad/alveus/internal/util"
)

type Options struct {
	ApplicationNamespace string
	Labels               map[string]string
	Annotations          map[string]string
	IgnoreDifferences    argov1alpha1.IgnoreDifferences
	SyncPolicy           *argov1alpha1.SyncPolicy
	Project              string
	Source               v1alpha1.Source
}

type Option func(*Options)

type Input struct {
	Name           string
	RepoURL        string
	TargetRevision string
	Destination    v1alpha1.Destination
}

func (in *Input) Validate() error {
	var errs []error

	if in.Name == "" {
		errs = append(errs, fmt.Errorf("name is required"))
	}

	if in.RepoURL == "" {
		errs = append(errs, fmt.Errorf("repoURL is required"))
	}

	if in.TargetRevision == "" {
		errs = append(errs, fmt.Errorf("targetRevision is required"))
	}

	if in.Destination.Namespace == "" {
		errs = append(errs, fmt.Errorf("destinationNamespace is required"))
	}

	if in.Destination.Name == "" && in.Destination.Server == "" {
		errs = append(errs, fmt.Errorf("destination.Name or destination.Server is required"))
	}

	if in.Destination.Name != "" && in.Destination.Server != "" {
		errs = append(errs, fmt.Errorf("destination.Name and destination.Server are mutually exclusive"))
	}

	return errors.Join(errs...)
}

func WithSource(src v1alpha1.Source) Option {
	return func(o *Options) {
		o.Source = src
	}
}

func FromServiceAPI(service v1alpha1.Service) Option {
	return func(o *Options) {
		o.SyncPolicy = &service.ArgoCD.SyncPolicy
	}
}

func NewApplication(input Input, options ...Option) (argov1alpha1.Application, error) {
	opts := &Options{
		ApplicationNamespace: "argocd",
		Labels:               map[string]string{},
		Annotations:          map[string]string{},
		Project:              "default",
	}
	for _, o := range options {
		o(opts)
	}

	labels := opts.Labels
	if len(labels) == 0 {
		labels = nil
	}

	annotations := opts.Annotations
	if len(annotations) == 0 {
		annotations = nil
	}

	err := input.Validate()
	if err != nil {
		return argov1alpha1.Application{}, err
	}

	source := argov1alpha1.ApplicationSource{
		RepoURL:        input.RepoURL,
		Path:           opts.Source.Path,
		TargetRevision: input.TargetRevision,
		Directory: &argov1alpha1.ApplicationSourceDirectory{
			Recurse: true,
			Jsonnet: opts.Source.Jsonnet,
			Exclude: opts.Source.Exclude,
			Include: opts.Source.Include,
		},
	}

	if source.Directory.Include == "" {
		source.Directory.Include = "{*.yml,*.yaml}"
	}

	finalizedName, err := util.SanitizeNameForKubernetes(input.Name)
	if err != nil {
		return argov1alpha1.Application{}, fmt.Errorf("sanitized application name is still problematic: %w", err)
	}

	app := argov1alpha1.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       argoapisapplication.ApplicationKind,
			APIVersion: argov1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        finalizedName,
			Namespace:   opts.ApplicationNamespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: argov1alpha1.ApplicationSpec{
			Destination: argov1alpha1.ApplicationDestination{
				Server:    input.Destination.Server,
				Namespace: input.Destination.Namespace,
				Name:      input.Destination.Name,
			},
			Project:           opts.Project,
			SyncPolicy:        opts.SyncPolicy,
			IgnoreDifferences: opts.IgnoreDifferences,
			Source:            &source,
		},
	}

	return app, nil
}

func FilenameFor(application argov1alpha1.Application) string {
	return strings.ToLower(
		util.Join("-",
			application.Name,
		),
	) + ".yaml"
}

type ApplicationRepository map[string]argov1alpha1.Application

func (r ApplicationRepository) GetByDestination(dest v1alpha1.Destination) (string, argov1alpha1.Application, bool) {
	for filePath, app := range r {
		if app.Spec.Destination.Namespace != dest.Namespace {
			continue
		}

		if app.Spec.Destination.Server != "" && app.Spec.Destination.Server == dest.Server {
			return filePath, app, true
		}

		if app.Spec.Destination.Name != "" && app.Spec.Destination.Name == dest.Name {
			return filePath, app, true
		}
	}

	return "", argov1alpha1.Application{}, false
}
