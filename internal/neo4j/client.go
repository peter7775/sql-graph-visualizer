package neo4j

import (
    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
    "github.com/spf13/viper"
)

type Client struct {
    driver neo4j.Driver
}

func NewClient() (*Client, error) {
    uri := viper.GetString("neo4j.uri")
    username := viper.GetString("neo4j.username")
    password := viper.GetString("neo4j.password")

    driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
    if err != nil {
        return nil, err
    }

    return &Client{driver.}, nil
}