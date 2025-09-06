# GraphQL Examples

This document contains example GraphQL queries and mutations that can be used to test the GraphQL endpoint.

## Access Points

- **GraphQL Endpoint**: `http://localhost:8081/graphql`
- **GraphQL Playground**: `http://localhost:8081/playground`

## Query Examples

### 1. Get Full Graph Data

Query to retrieve all nodes and relationships in the graph:

```graphql
query GetFullGraph {
  graph {
    nodes {
      id
      label
      properties
    }
    relationships {
      from
      to
      type
      properties
    }
  }
}
```

### 2. Get Nodes by Type/Label

Query to retrieve all nodes of a specific type:

```graphql
query GetNodesByType($nodeType: String!) {
  nodesByType(type: $nodeType) {
    id
    label
    properties
  }
}
```

**Variables:**
```json
{
  "nodeType": "User"
}
```

### 3. Get Specific Node by ID

Query to retrieve a specific node by its ID:

```graphql
query GetNodeById($nodeId: ID!) {
  node(id: $nodeId) {
    id
    label
    properties
  }
}
```

**Variables:**
```json
{
  "nodeId": "User_1"
}
```

### 4. Search Nodes by Property

Query to search for nodes containing specific text in their properties:

```graphql
query SearchNodes($searchTerm: String!) {
  searchNodes(query: $searchTerm) {
    id
    label
    properties
  }
}
```

**Variables:**
```json
{
  "searchTerm": "john"
}
```

### 5. Get Relationships by Type

Query to retrieve all relationships of a specific type:

```graphql
query GetRelationshipsByType($relType: String!) {
  relationshipsByType(type: $relType) {
    from
    to
    type
    properties
  }
}
```

**Variables:**
```json
{
  "relType": "CREATED"
}
```

### 6. Get Configuration

Query to retrieve the current application configuration:

```graphql
query GetConfig {
  config {
    neo4j {
      uri
      username
      password
    }
  }
}
```

## Mutation Examples

### 1. Trigger Data Transformation

Mutation to trigger the MySQL to Neo4j data transformation process:

```graphql
mutation TransformData {
  transformData
}
```

## Complex Query Examples

### 1. Get Graph with Filtered Data

Combined query to get specific nodes and their relationships:

```graphql
query GetGraphWithFilter($nodeType: String!, $relType: String!) {
  nodesByType(type: $nodeType) {
    id
    label
    properties
  }
  relationshipsByType(type: $relType) {
    from
    to
    type
    properties
  }
}
```

**Variables:**
```json
{
  "nodeType": "User",
  "relType": "FOLLOWS"
}
```

### 2. Search and Configure

Query to search nodes and get configuration in a single request:

```graphql
query SearchAndConfig($searchTerm: String!) {
  searchNodes(query: $searchTerm) {
    id
    label
    properties
  }
  config {
    neo4j {
      uri
      username
    }
  }
}
```

**Variables:**
```json
{
  "searchTerm": "test"
}
```

## Using with curl

You can also test the GraphQL endpoint using curl:

```bash
# Example query using curl
curl -X POST http://localhost:8081/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query { graph { nodes { id label properties } } }"
  }'

# Example query with variables using curl
curl -X POST http://localhost:8081/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query GetNodesByType($type: String!) { nodesByType(type: $type) { id label properties } }",
    "variables": { "type": "User" }
  }'
```

## Error Handling

GraphQL returns errors in a specific format. Here's an example of error response:

```json
{
  "data": null,
  "errors": [
    {
      "message": "node with ID NonExistent_999 not found",
      "path": ["node"]
    }
  ]
}
```

## Notes

1. **Property Format**: Node and relationship properties are returned as JSON strings
2. **ID Format**: Node IDs follow the pattern `{Type}_{Key}` (e.g., `User_1`, `Post_2`)
3. **Real-time Updates**: Subscription support is available for future real-time graph updates
4. **Authentication**: Currently no authentication is required for GraphQL queries
5. **Rate Limiting**: No rate limiting is currently implemented

## Playground Usage

The GraphQL Playground provides an interactive environment to test queries:

1. Open `http://localhost:8081/playground` in your browser
2. Use the left panel to write queries
3. Use the bottom-left panel to add variables
4. Click the play button to execute queries
5. View results in the right panel
6. Use the "Docs" tab to explore the schema
