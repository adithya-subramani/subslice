package graph

import (
	"subslice/pkg/model"
)

type TraversalDirection int

const (
	Upstream TraversalDirection = iota
	Downstream
)

type TraversalPlan struct {
	UpstreamEntities   []string
	DownstreamEntities []string
	Relationships      []model.Relationship
}
