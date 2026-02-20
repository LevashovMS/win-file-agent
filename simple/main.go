package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/{name}", func(w http.ResponseWriter, r *http.Request) {
		var fileName = r.PathValue("name")
		if len(fileName) == 0 {
			return
		}

		var buffer, err = os.ReadFile(fileName)
		if err != nil {
			fmt.Println("Error file read:", err)
			return
		}

		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusOK)
		w.Write(buffer)
	})

	fmt.Println("Starting server at port 8088")
	err := http.ListenAndServe(":8088", nil)
	if err != nil {
		fmt.Println("Error starting the server:", err)
	}
}
