# SQL database conversion and visualisation as graph

## basic functionality:

- conversion of part or the whole SQL database into Neo4j graph database with its subsequent visualization
- the conversion takes place according to user-defined rules (config.yml), which allow selection from SQL not only by tables, but also by SQL queries
- it is possible to define directional logical links (sessions) between the resulting nodes, and add their direction and name

## use

- complete conversion of SQL database to Neo4j according to personaly define rules (in config.yml file)
- visualization of the analysis of SQL database elements that we need to see - rewriting new relations as logical or functional relations (as opposed to SQL relation data)
- in the next stage the possibility to turn the process around quite easily - i.e. to design Neo4j graph structure and convert it to MySQL

## architecture

Project is written in Go in Domain Drive Design software architecture style. For visualization is used GraphQL and Neovis.JS.

## config

## main functionality

## test

## todo
