package main

import (
	"github.com/apex/gateway"
	"log"
	"net/http"
	"os"
)

var RedirectUrl = os.Getenv("REDIRECT_URL")

func Redirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, RedirectUrl, 302)
}

func main() {
	http.HandleFunc("/", Redirect)
	if len(RedirectUrl) == 0 {
		log.Fatal("Redirect not defined")
	}
	log.Printf("Redirect URL: %s\n", RedirectUrl)
	log.Fatal(gateway.ListenAndServe(":3000", nil))
}
