/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package graphql


import (
	"context"
	graphql "mysql-graph-visualizer/internal/application/services/graphql/generated"
	graphqlModels "mysql-graph-visualizer/internal/domain/models/graphql"
)

func (r *queryResolver) Nodes(ctx context.Context) ([]*graphqlModels.Node, error) {
	graphAgg := graph.NewGraphAggregate("")
	nodes := graphAgg.GetNodes()

	var gqlNodes []*graphqlModels.Node
	for _, node := range nodes {
		var props *graphqlModels.Properties
		if node.Properties != nil {
			props = &graphqlModels.Properties{
				Key:   node.Properties["key"].(string),
				Value: node.Properties["value"].(string),
			}
		}

		gqlNode := &graphqlModels.Node{
			ID:         node.ID,
			Label:      node.Type,
			Properties: props,
		}
		gqlNodes = append(gqlNodes, gqlNode)
	}

	return gqlNodes, nil
}

func (r *Resolver) Query() graphql.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
