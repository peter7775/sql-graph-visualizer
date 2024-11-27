package main

import (
    "fmt"
    "log"
    "neo4j-mysql-bridge/internal/mysql"
    "neo4j-mysql-bridge/internal/neo4j"
    "neo4j-mysql-bridge/internal/visualization"
)

func main() {
    mysqlClient, err := mysql.NewClient()
    if err != nil {
        log.Fatalf("Failed to connect to MySQL: %v", err)
    }
    defer mysqlClient.Close()

    neo4jClient, err := neo4j.NewClient()
    if err != nil {
        log.Fatalf("Failed to connect to Neo4j: %v", err)
    }
    defer neo4jClient.Close()

    // Transform data
    transformer := analysis.NewTransformer()
    mysqlData := []mysql.DataType{} // Replace with a function to fetch data from MySQL
    transformedData := transformer.TransformData(mysqlData)

    // Import transformed data to Neo4j
    // Example: neo4jClient.InsertData(transformedData)

    // Start visualization
    visualizer := visualization.NewVisualizer()
    visualizer.ServeVisualization()

    fmt.Println("Data transfer and visualization complete!")
}