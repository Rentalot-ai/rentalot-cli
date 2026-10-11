package rentalotcli_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
)

type response6igContact struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Email *string `json:"email"`
}

type response6igBody struct {
	io.Reader
	closed bool
}

func (b *response6igBody) Close() error {
	b.closed = true
	return nil
}

func TestResponse6igDecodeHTTP(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer offline-6ig" {
			t.Error("missing authentication")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, `{"data":{"id":"c1","name":"Alice","email":null,"role":"tenant"}}`)
		case http.MethodPost, http.MethodPatch:
			var body response6igContact
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name != "Alice" {
				t.Errorf("request body = %+v, error = %v", body, err)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"data":{"id":"c1","name":"Alice","email":"alice@example.test"}}`)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()
	client := rentalotcli.NewClient(rentalotcli.Config{BaseURL: srv.URL, APIKey: "offline-6ig"})
	requests := map[string]func() (*http.Response, error){
		"GET": func() (*http.Response, error) { return client.Get(context.Background(), "/api/v1/contacts/c1", nil) },
		"POST": func() (*http.Response, error) {
			return client.Post(context.Background(), "/api/v1/contacts", response6igContact{Name: "Alice"})
		},
		"PATCH": func() (*http.Response, error) {
			return client.Patch(context.Background(), "/api/v1/contacts/c1", response6igContact{Name: "Alice"})
		},
		"DELETE": func() (*http.Response, error) { return client.Delete(context.Background(), "/api/v1/contacts/c1") },
	}
	for method, request := range requests {
		t.Run(method, func(t *testing.T) {
			resp, err := request()
			if err != nil {
				t.Fatal(err)
			}
			got, err := rentalotcli.DecodeResponse[rentalotcli.DataResponse[response6igContact]](resp)
			if err != nil {
				t.Fatal(err)
			}
			if method == "DELETE" {
				if got.Data != (response6igContact{}) {
					t.Fatalf("204 data = %+v, want zero value", got.Data)
				}
				return
			}
			if got.Data.ID != "c1" || got.Data.Name != "Alice" {
				t.Fatalf("decoded data = %+v", got.Data)
			}
			if method == "GET" && got.Data.Email != nil {
				t.Fatalf("null email = %v", got.Data.Email)
			}
			if method != "GET" && (got.Data.Email == nil || *got.Data.Email != "alice@example.test") {
				t.Fatalf("email = %v", got.Data.Email)
			}
		})
	}
}

func TestResponse6igDecodeFailuresAndOwnership(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status int
		body   string
		apiErr bool
		ok     bool
	}{
		{"plain typed resource", 200, `{"id":"c1","name":"Alice"}`, false, true},
		{"trailing whitespace", 200, "{\"id\":\"c1\"}\n\t ", false, true},
		{"malformed", 200, `{"id":`, false, false},
		{"wrong field type", 200, `{"id":42}`, false, false},
		{"empty success", 200, "", false, false},
		{"extra value", 200, `{"id":"c1"} {"id":"c2"}`, false, false},
		{"trailing garbage", 200, `{"id":"c1"} junk`, false, false},
		{"no content", 204, "", false, true},
		{"reset content", 205, "", false, true},
		{"redirect", 302, `{"id":"c1"}`, true, false},
		{"envelope error", 404, `{"error":{"code":"not_found","message":"Contact not found"}}`, true, false},
		{"problem error", 429, `{"type":"urn:problem:rate-limited","detail":"Write limit exceeded"}`, true, false},
		{"non JSON error", 502, "upstream unavailable", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := &response6igBody{Reader: strings.NewReader(tc.body)}
			resp := &http.Response{StatusCode: tc.status, Body: body, Header: http.Header{"Retry-After": {"3"}}}
			got, err := rentalotcli.DecodeResponse[response6igContact](resp)
			if (err == nil) != tc.ok {
				t.Fatalf("error = %v, want success %v", err, tc.ok)
			}
			if !body.closed {
				t.Error("response body was not closed")
			}
			if err != nil && got != (response6igContact{}) {
				t.Errorf("failed decoding returned partial data: %+v", got)
			}
			var apiErr *rentalotcli.APIError
			if errors.As(err, &apiErr) != tc.apiErr {
				t.Fatalf("API error = %v, want %v", err, tc.apiErr)
			}
			if apiErr != nil && (apiErr.StatusCode != tc.status || apiErr.RetryAfter != 3) {
				t.Errorf("lost API metadata: %+v", apiErr)
			}
		})
	}
}

func TestResponse6igMissingResponseAndBody(t *testing.T) {
	t.Parallel()
	if _, err := rentalotcli.DecodeResponse[response6igContact](nil); err == nil {
		t.Fatal("nil response succeeded")
	}
	if _, err := rentalotcli.DecodeResponse[response6igContact](&http.Response{StatusCode: 200}); !errors.Is(err, io.EOF) {
		t.Fatalf("nil success body error = %v, want EOF", err)
	}
	_, err := rentalotcli.DecodeResponse[response6igContact](&http.Response{StatusCode: 500})
	var apiErr *rentalotcli.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 500 {
		t.Fatalf("nil error body = %v", err)
	}
}

func TestResponse6igCursorTraversalHTTP(t *testing.T) {
	t.Parallel()
	const cursor = "opaque+/=& cursor"
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/api/v1/contacts" || q.Get("limit") != "2" || q.Get("status") != "prospect" || r.Header.Get("Authorization") != "Bearer offline-6ig" {
			t.Errorf("unexpected request: %s, auth %q", r.URL, r.Header.Get("Authorization"))
		}
		seen = append(seen, q.Get("cursor"))
		switch q.Get("cursor") {
		case "":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []response6igContact{{ID: "c1", Name: "Alice"}, {ID: "c2", Name: "Bob"}}, "pagination": map[string]any{"nextCursor": cursor, "hasMore": true}})
		case cursor:
			_, _ = io.WriteString(w, `{"data":[{"id":"c3","name":"Carol"}],"pagination":{"nextCursor":null,"hasMore":false}}`)
		default:
			t.Errorf("unexpected cursor %q", q.Get("cursor"))
			http.Error(w, "bad cursor", 400)
		}
	}))
	defer srv.Close()
	client := rentalotcli.NewClient(rentalotcli.Config{BaseURL: srv.URL, APIKey: "offline-6ig"})
	current := rentalotcli.CursorParams{Limit: 2}
	var ids []string
	for requests := 0; ; requests++ {
		if requests >= 3 {
			t.Fatal("cursor traversal did not terminate")
		}
		params, err := current.QueryParams()
		if err != nil {
			t.Fatal(err)
		}
		params["status"] = "prospect"
		resp, err := client.Get(context.Background(), "/api/v1/contacts", params)
		if err != nil {
			t.Fatal(err)
		}
		page, err := rentalotcli.DecodeResponse[rentalotcli.CursorPage[response6igContact]](resp)
		if err != nil {
			t.Fatal(err)
		}
		for _, contact := range page.Data {
			ids = append(ids, contact.ID)
		}
		next, more, err := page.Next(current)
		if err != nil {
			t.Fatal(err)
		}
		if !more {
			break
		}
		current = next
	}
	if !reflect.DeepEqual(ids, []string{"c1", "c2", "c3"}) || !reflect.DeepEqual(seen, []string{"", cursor}) {
		t.Fatalf("ids = %v, requested cursors = %v", ids, seen)
	}
}

func TestResponse6igOffsetTraversalHTTP(t *testing.T) {
	t.Parallel()
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("status") != "active" {
			t.Error("filter lost")
		}
		seen = append(seen, q.Get("offset"))
		switch q.Get("offset") {
		case "0":
			if q.Get("limit") != "" {
				t.Errorf("default limit = %q", q.Get("limit"))
			}
			_, _ = io.WriteString(w, `{"data":[{"id":"p1"},{"id":"p2"}],"pagination":{"offset":0,"limit":2,"total":3}}`)
		case "2":
			if q.Get("limit") != "2" {
				t.Errorf("effective limit = %q", q.Get("limit"))
			}
			_, _ = io.WriteString(w, `{"data":[{"id":"p3"}],"pagination":{"offset":2,"limit":2,"total":3}}`)
		default:
			t.Errorf("unexpected offset %q", q.Get("offset"))
			http.Error(w, "bad offset", 400)
		}
	}))
	defer srv.Close()
	client := rentalotcli.NewClient(rentalotcli.Config{BaseURL: srv.URL})
	current := rentalotcli.OffsetParams{}
	var ids []string
	for requests := 0; ; requests++ {
		if requests >= 3 {
			t.Fatal("offset traversal did not terminate")
		}
		params, err := current.QueryParams()
		if err != nil {
			t.Fatal(err)
		}
		params["status"] = "active"
		resp, err := client.Get(context.Background(), "/api/v1/properties", params)
		if err != nil {
			t.Fatal(err)
		}
		page, err := rentalotcli.DecodeResponse[rentalotcli.OffsetPage[response6igContact]](resp)
		if err != nil {
			t.Fatal(err)
		}
		for _, property := range page.Data {
			ids = append(ids, property.ID)
		}
		next, more, err := page.Next()
		if err != nil {
			t.Fatal(err)
		}
		if !more {
			break
		}
		current = next
	}
	if !reflect.DeepEqual(ids, []string{"p1", "p2", "p3"}) || !reflect.DeepEqual(seen, []string{"0", "2"}) {
		t.Fatalf("ids = %v, requested offsets = %v", ids, seen)
	}
}

func TestResponse6igCursorNext(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		more bool
		fail bool
	}{
		{"missing metadata", `{"data":[]}`, false, false},
		{"null cursor", `{"pagination":{"nextCursor":null}}`, false, false},
		{"empty cursor", `{"pagination":{"nextCursor":""}}`, false, false},
		{"explicit last", `{"pagination":{"nextCursor":"next","hasMore":false}}`, false, false},
		{"cursor without hasMore", `{"pagination":{"nextCursor":"next"}}`, true, false},
		{"empty data with continuation", `{"data":[],"pagination":{"nextCursor":"next","hasMore":true}}`, true, false},
		{"missing continuation", `{"pagination":{"hasMore":true}}`, false, true},
		{"repeated cursor", `{"pagination":{"nextCursor":"current","hasMore":true}}`, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tc.body))}
			page, err := rentalotcli.DecodeResponse[rentalotcli.CursorPage[response6igContact]](resp)
			if err != nil {
				t.Fatal(err)
			}
			next, more, err := page.Next(rentalotcli.CursorParams{Limit: 25, Cursor: "current"})
			if more != tc.more || (err != nil) != tc.fail {
				t.Fatalf("Next = %+v, %v, %v; want more %v, error %v", next, more, err, tc.more, tc.fail)
			}
			if more && next != (rentalotcli.CursorParams{Limit: 25, Cursor: "next"}) {
				t.Errorf("next params = %+v", next)
			}
		})
	}
}

func TestResponse6igOffsetNext(t *testing.T) {
	t.Parallel()
	maxInt := int(^uint(0) >> 1)
	type response6igOffsetCase struct {
		name   string
		count  int
		limit  int
		offset int
		total  *int
		more   bool
		fail   bool
	}
	cases := []response6igOffsetCase{
		{name: "unknown total full page", count: 2, limit: 2, offset: 4, more: true},
		{name: "unknown total short page", count: 1, limit: 2, offset: 4},
		{name: "unknown total empty page", limit: 2, offset: 4},
		{name: "empty without metadata"},
		{name: "zero limit with items", count: 1, fail: true},
		{name: "too many items", count: 3, limit: 2, fail: true},
		{name: "negative limit", limit: -1, fail: true},
		{name: "negative offset", offset: -1, fail: true},
		{name: "overflow", count: 2, limit: 2, offset: maxInt - 1, fail: true},
	}
	for _, total := range []int{-1, 0, 5, 6, 7} {
		cases = append(cases, response6igOffsetCase{name: "known total " + strconv.Itoa(total), count: 2, limit: 2, offset: 4, total: &total, more: total > 6, fail: total < 0})
	}
	staleTotal := 10
	cases = append(cases,
		response6igOffsetCase{name: "empty with stale total", limit: 2, offset: 4, total: &staleTotal},
		response6igOffsetCase{name: "short page with known remaining items", count: 1, limit: 2, offset: 4, total: &staleTotal, more: true},
	)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			page := rentalotcli.OffsetPage[response6igContact]{Data: make([]response6igContact, tc.count), Pagination: rentalotcli.OffsetPagination{Limit: tc.limit, Offset: tc.offset, Total: tc.total}}
			next, more, err := page.Next()
			if more != tc.more || (err != nil) != tc.fail {
				t.Fatalf("Next = %+v, %v, %v; want more %v, error %v", next, more, err, tc.more, tc.fail)
			}
			if more && next != (rentalotcli.OffsetParams{Limit: tc.limit, Offset: tc.offset + tc.count}) {
				t.Errorf("next params = %+v", next)
			}
		})
	}
}

func TestResponse6igQueryParams(t *testing.T) {
	t.Parallel()
	for _, limit := range []int{0, 10} {
		cursor := rentalotcli.CursorParams{Limit: limit, Cursor: "opaque"}
		got, err := cursor.QueryParams()
		if err != nil || got["cursor"] != "opaque" || got["offset"] != "" {
			t.Fatalf("cursor params = %v, %v", got, err)
		}
		if (got["limit"] == "") != (limit == 0) || (limit > 0 && got["limit"] != "10") {
			t.Errorf("limit %d encoded as %q", limit, got["limit"])
		}
		got["cursor"] = "changed"
		again, _ := cursor.QueryParams()
		if again["cursor"] != "opaque" {
			t.Error("query map mutation affected subsequent parameters")
		}
	}
	got, err := (rentalotcli.CursorParams{}).QueryParams()
	if err != nil || len(got) != 0 {
		t.Fatalf("default cursor params = %v, %v", got, err)
	}
	got, err = (rentalotcli.OffsetParams{Limit: 10, Offset: 20}).QueryParams()
	if err != nil || !reflect.DeepEqual(got, rentalotcli.QueryParams{"limit": "10", "offset": "20"}) {
		t.Fatalf("offset params = %v, %v", got, err)
	}
	if _, err := (rentalotcli.CursorParams{Limit: -1}).QueryParams(); err == nil {
		t.Error("negative cursor limit accepted")
	}
	if _, _, err := (rentalotcli.CursorPage[response6igContact]{}).Next(rentalotcli.CursorParams{Limit: -1}); err == nil {
		t.Error("negative current cursor limit accepted")
	}
	for _, p := range []rentalotcli.OffsetParams{{Limit: -1}, {Offset: -1}} {
		if _, err := p.QueryParams(); err == nil {
			t.Errorf("invalid offset params accepted: %+v", p)
		}
	}
}

type response6igFailingReader struct {
	err error
}

func (r response6igFailingReader) Read([]byte) (int, error) { return 0, r.err }

func TestResponse6igReadFailure(t *testing.T) {
	t.Parallel()
	body := &response6igBody{Reader: response6igFailingReader{err: context.Canceled}}
	_, err := rentalotcli.DecodeResponse[response6igContact](&http.Response{StatusCode: 200, Body: body})
	if !errors.Is(err, context.Canceled) || !body.closed {
		t.Fatalf("read error = %v, closed = %v", err, body.closed)
	}
}
