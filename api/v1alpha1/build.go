package v1alpha1

import (
	"fmt"

	"github.com/cakehappens/gocto"
	"github.com/goccy/go-yaml"

	"github.com/ghostsquad/alveus/internal/util"
)

func NewFromYaml(contents []byte) (Service, error) {
	service := &Service{}
	err := yaml.Unmarshal(contents, service)
	if err != nil {
		return *service, fmt.Errorf("unmarshalling yaml: %w", err)
	}

	service.Inflate()

	return *service, service.Validate()
}

func (s *Service) Inflate() {
	if s.Github.On.Dispatch == nil {
		s.Github.On.Dispatch = &gocto.OnDispatch{}
	}

	for gIdx, group := range s.DestinationGroups {
		for dIdx, dest := range group.Destinations {
			dest.Namespace = util.CoalesceZero(
				dest.Namespace,
				group.DestinationNamespace,
				s.DestinationNamespace,
			)

			(&dest.ArgoCD).CoalesceFields(group.ArgoCD, s.ArgoCD)
			(&dest.Github).CoalesceFields(group.Github, s.Github)

			s.DestinationGroups[gIdx].Destinations[dIdx] = dest
		}
	}
}
