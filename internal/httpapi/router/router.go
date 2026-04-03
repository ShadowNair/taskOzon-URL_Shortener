package router

import (
	"net/http"
	linkhandler "url-shortener/internal/httpapi/link"
	swaggerhandler "url-shortener/internal/httpapi/swagger"
	"url-shortener/internal/httpapi/middleware"

	"github.com/gorilla/mux"
)

func Setup(linkH *linkhandler.Handler) *mux.Router {
	r := mux.NewRouter()
	r.Use(middleware.CORS(nil))

	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/link", linkH.CreateShortLink).Methods(http.MethodPost)
	api.HandleFunc("/link/{shortCode}", linkH.GetOriginalURLByShort).Methods(http.MethodGet)

	r.HandleFunc("/healthz", linkH.Health).Methods(http.MethodGet)
	r.Handle("/swagger/", swaggerhandler.UI()).Methods(http.MethodGet)
	r.Handle("/swagger/doc.json", swaggerhandler.Doc()).Methods(http.MethodGet)

	return r
}