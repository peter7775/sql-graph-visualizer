package models

type TransformationConfig struct {
	Name      string           `yaml:"name"`
	Source    SourceConfig     `yaml:"source"`
	Nodes     []NodeConfig     `yaml:"nodes"`
	Relations []RelationConfig `yaml:"relations"`
}

type SourceConfig struct {
	Type  string `yaml:"type"` // "table" nebo "query"
	Value string `yaml:"value"`
}

type NodeConfig struct {
	Label      string            `yaml:"label"`
	Properties []PropertyMapping `yaml:"properties"`
}

type PropertyMapping struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

type RelationConfig struct {
	Type string       `yaml:"type"`
	From RelationNode `yaml:"from"`
	To   RelationNode `yaml:"to"`
}

type RelationNode struct {
	Label string `yaml:"label"`
	Match string `yaml:"match"`
}

type Config struct {
	Neo4j struct {
		URI      string
		User     string
		Password string
	}
	MySQL struct {
		Host     string
		Port     int
		User     string
		Password string
		Database string
	}
	Server struct {
		Port int
	}
}
