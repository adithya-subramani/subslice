package graph

import (
	"fmt"
	"subslice/pkg/config"
	"subslice/pkg/model"
)

type Engine struct {
	graph *model.StorageGraph
}

func NewEngine(discoveredGraph *model.StorageGraph, implicitFKs []config.ImplicitForeignKey) *Engine {
	for _, ifk := range implicitFKs {
		rel := model.Relationship{
			ConstraintName: fmt.Sprintf("implicit_%s_%s", ifk.Table, ifk.Column),
			ChildEntity:    ifk.Table,
			ChildField:     ifk.Column,
			ParentEntity:   ifk.ForeignTable,
			ParentField:    ifk.ForeignColumn,
		}
		discoveredGraph.Relationships = append(discoveredGraph.Relationships, rel)
		discoveredGraph.Entities[ifk.Table] = true
		discoveredGraph.Entities[ifk.ForeignTable] = true
	}

	return &Engine{graph: discoveredGraph}
}

func (e *Engine) BuildPlan(rootEntity string, maxDepth int) (*TraversalPlan, error) {
	plan := &TraversalPlan{
		UpstreamEntities:   []string{},
		DownstreamEntities: []string{},
		Relationships:      e.graph.Relationships,
	}

	visited := make(map[string]bool)
	visited[rootEntity] = true

	e.resolveParents(rootEntity, visited, &plan.UpstreamEntities, 0, maxDepth)

	visitedDownstream := make(map[string]bool)
	visitedDownstream[rootEntity] = true

	e.resolveChildren(rootEntity, visitedDownstream, &plan.DownstreamEntities, 0, maxDepth)

	return plan, nil
}

func (e *Engine) resolveParents(entity string, visited map[string]bool, upstream *[]string, depth, maxDepth int) {
	if depth >= maxDepth {
		return
	}

	for _, rel := range e.graph.Relationships {
		if rel.ChildEntity == entity && !visited[rel.ParentEntity] {
			visited[rel.ParentEntity] = true
			*upstream = append(*upstream, rel.ParentEntity)
			e.resolveParents(rel.ParentEntity, visited, upstream, depth+1, maxDepth)
		}
	}
}

func (e *Engine) resolveChildren(entity string, visited map[string]bool, downstream *[]string, depth, maxDepth int) {
	if depth >= maxDepth {
		return
	}

	for _, rel := range e.graph.Relationships {
		if rel.ParentEntity == entity && !visited[rel.ChildEntity] {
			visited[rel.ChildEntity] = true
			*downstream = append(*downstream, rel.ChildEntity)
			e.resolveChildren(rel.ChildEntity, visited, downstream, depth+1, maxDepth)
		}
	}
}
