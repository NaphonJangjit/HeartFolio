package webhttp

import (
	"fmt"
	"net/http"
)

const ContentTypeJSONAPI = "application/vnd.api+json"

// Resource represents a single JSON:API resource object.
type Resource struct {
	Type          string                 `json:"type"`
	ID            string                 `json:"id,omitempty"`
	Attributes    interface{}            `json:"attributes"`
	Relationships map[string]Relationship `json:"relationships,omitempty"`
}

// Relationship represents a JSON:API relationship object.
type Relationship struct {
	Data interface{} `json:"data"`
}

// RelationshipRef is a resource linkage (type + id only).
type RelationshipRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Document is a JSON:API top-level document with a single resource.
type Document struct {
	Data interface{} `json:"data"`
}

// ErrorObject is a JSON:API error object.
type ErrorObject struct {
	Status string `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// ErrorDocument is a JSON:API top-level document with errors.
type ErrorDocument struct {
	Errors []ErrorObject `json:"errors"`
}

// RespondOne writes a JSON:API response with a single resource.
func RespondOne(w http.ResponseWriter, status int, resource Resource) {
	w.Header().Set("Content-Type", ContentTypeJSONAPI)
	w.WriteHeader(status)
	encode(w, Document{Data: resource})
}

// RespondMany writes a JSON:API response with a collection of resources.
func RespondMany(w http.ResponseWriter, status int, resources []Resource) {
	if resources == nil {
		resources = []Resource{}
	}
	w.Header().Set("Content-Type", ContentTypeJSONAPI)
	w.WriteHeader(status)
	encode(w, Document{Data: resources})
}

// RespondError writes a JSON:API error response.
func RespondError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", ContentTypeJSONAPI)
	w.WriteHeader(status)
	encode(w, ErrorDocument{
		Errors: []ErrorObject{
			{
				Status: fmt.Sprintf("%d", status),
				Title:  http.StatusText(status),
				Detail: detail,
			},
		},
	})
}

func encode(w http.ResponseWriter, v interface{}) {
	if err := encodeJSON(w, v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
