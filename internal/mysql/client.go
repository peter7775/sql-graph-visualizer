package mysql

import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "github.com/spf13/viper"
)

type Client struct {
    *sql.DB
}

func NewClient() (*Client, error) {
    dsn := viper.GetString("mysql.dsn")
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }

    return &Client{db}, nil
}