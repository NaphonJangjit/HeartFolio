package webhttp

import (
	"net/http"
	"strconv"
)

const (
	DefaultPageLimit = 20
	MaxPageLimit     = 100
)

// Page holds parsed pagination parameters.
type Page struct {
	Offset int
	Limit  int
}

// ParsePage extracts pagination from query params: ?page[offset]=0&page[limit]=20
func ParsePage(r *http.Request) Page {
	p := Page{Offset: 0, Limit: DefaultPageLimit}

	if v := r.URL.Query().Get("page[offset]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			p.Offset = n
		}
	}
	if v := r.URL.Query().Get("page[limit]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.Limit = n
		}
	}
	if p.Limit > MaxPageLimit {
		p.Limit = MaxPageLimit
	}
	return p
}

// Apply slices a resource list according to offset/limit.
// Returns the sliced portion and the total count.
func (p Page) Apply(resources []Resource) ([]Resource, int) {
	total := len(resources)
	if p.Offset >= total {
		return []Resource{}, total
	}
	end := p.Offset + p.Limit
	if end > total {
		end = total
	}
	return resources[p.Offset:end], total
}

// RespondManyPaginated writes a JSON:API collection with pagination meta.
func RespondManyPaginated(w http.ResponseWriter, status int, resources []Resource, page Page) {
	sliced, total := page.Apply(resources)
	resp := map[string]interface{}{
		"data": sliced,
		"meta": map[string]interface{}{
			"total":  total,
			"offset": page.Offset,
			"limit":  page.Limit,
		},
	}
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	encodeJSON(w, resp)
}
