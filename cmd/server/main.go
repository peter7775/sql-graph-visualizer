package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Test Visualization</title>
    <script src="https://unpkg.com/vis-network/standalone/umd/vis-network.min.js"></script>
    <style>
        #mynetwork {
            width: 100%;
            height: 90vh;
            border: 1px solid lightgray;
        }
    </style>
</head>
<body>
    <div id="mynetwork"></div>
    <script>
        const nodes = [
            { id: 1, label: "Node 1" },
            { id: 2, label: "Node 2" }
        ];
        const edges = [
            { from: 1, to: 2 }
        ];
        const container = document.getElementById("mynetwork");
        const data = { nodes: nodes, edges: edges };
        new vis.Network(container, data, {});
    </script>
</body>
</html>`)
	})

	log.Println("Server běží na http://0.0.0.0:8080")
	if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
		log.Fatal(err)
	}
}
