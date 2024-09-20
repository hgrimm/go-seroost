package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

func startServer(address string, model *Model, modelMutex *sync.Mutex) {
	fmt.Printf("INFO: listening at http://%s/\n", address)

	http.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		serveAPISearch(w, r, model, modelMutex)
	})

	http.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		serveAPIStats(w, r, model, modelMutex)
	})

	http.HandleFunc("/index.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Write(indexJs)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(indexHtml)
		} else {
			http.NotFound(w, r)
		}
	})

	if err := http.ListenAndServe(address, nil); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not start HTTP server at %s: %v\n", address, err)
	}
}

func serveAPISearch(w http.ResponseWriter, r *http.Request, model *Model, modelMutex *sync.Mutex) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Could not read request body", http.StatusInternalServerError)
		return
	}
	query := []rune(string(body))
	// print query string
	fmt.Println("INFO: query string: ", string(body))
	modelMutex.Lock()
	results := model.SearchQuery(query)
	modelMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results[:min(len(results), 20)])
}

func serveAPIStats(w http.ResponseWriter, r *http.Request, model *Model, modelMutex *sync.Mutex) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	modelMutex.Lock()
	stats := struct {
		DocsCount  int `json:"docs_count"`
		TermsCount int `json:"terms_count"`
	}{
		DocsCount:  len(model.Docs),
		TermsCount: len(model.DF),
	}
	modelMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

var indexJs = []byte(`// TODO: live update results as you type
async function search(prompt) {
    const results = document.getElementById("results")
    results.innerHTML = "";
    const response = await fetch("/api/search", {
        method: 'POST',
        headers: {'Content-Type': 'text/plain'},
        body: prompt,
    });
    const json = await response.json();
    results.innerHTML = "";
    for (const [index, json_item] of Object.entries(json)) {
        let item = document.createElement("span");
        item.appendChild(document.createTextNode(json_item.path+" "+json_item.rank));
        item.appendChild(document.createElement("br"));
        results.appendChild(item);
    }
}

let query = document.getElementById("query");
let currentSearch = Promise.resolve()

query.addEventListener("keypress", (e) => {
    if (e.key == "Enter") {
        currentSearch.then(() => search(query.value));
    }
})`)

var indexHtml = []byte(`<html>
    <head>
        <title>Seroost</title>
    </head>
    <body>
        <h1>Provide Your Query:</h1>
        <input id="query" type="text" />
        <div id="results"></div>
        <script src="index.js"></script>
    </body>
</html>`)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
