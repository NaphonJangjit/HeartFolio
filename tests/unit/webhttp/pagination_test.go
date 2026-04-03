package webhttp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NaphonJangjit/HeartFolio/internal/webhttp"
	"github.com/stretchr/testify/assert"
)

func TestParsePage_Defaults(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	p := webhttp.ParsePage(r)

	assert.Equal(t, 0, p.Offset)
	assert.Equal(t, webhttp.DefaultPageLimit, p.Limit)
}

func TestParsePage_Custom(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test?page[offset]=10&page[limit]=5", nil)
	p := webhttp.ParsePage(r)

	assert.Equal(t, 10, p.Offset)
	assert.Equal(t, 5, p.Limit)
}

func TestParsePage_MaxLimit(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test?page[limit]=999", nil)
	p := webhttp.ParsePage(r)

	assert.Equal(t, webhttp.MaxPageLimit, p.Limit)
}

func TestParsePage_InvalidValues(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test?page[offset]=abc&page[limit]=-1", nil)
	p := webhttp.ParsePage(r)

	assert.Equal(t, 0, p.Offset, "invalid offset should default to 0")
	assert.Equal(t, webhttp.DefaultPageLimit, p.Limit, "invalid limit should default to DefaultPageLimit")
}

func TestPage_Apply(t *testing.T) {
	resources := make([]webhttp.Resource, 25)
	for i := range resources {
		resources[i] = webhttp.Resource{Type: "test", ID: "id"}
	}

	tests := []struct {
		name      string
		page      webhttp.Page
		wantLen   int
		wantTotal int
	}{
		{"first page", webhttp.Page{Offset: 0, Limit: 10}, 10, 25},
		{"second page", webhttp.Page{Offset: 10, Limit: 10}, 10, 25},
		{"last page partial", webhttp.Page{Offset: 20, Limit: 10}, 5, 25},
		{"beyond end", webhttp.Page{Offset: 30, Limit: 10}, 0, 25},
		{"all at once", webhttp.Page{Offset: 0, Limit: 100}, 25, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total := tt.page.Apply(resources)
			assert.Len(t, got, tt.wantLen)
			assert.Equal(t, tt.wantTotal, total)
		})
	}
}
