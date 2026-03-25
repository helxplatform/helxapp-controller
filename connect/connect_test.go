package connect

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchData_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"alice","age":30,"active":true,"tags":["admin","user"]}`))
	}))
	defer server.Close()

	data, err := FetchData(server.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if data["name"] != "alice" {
		t.Errorf("expected name=alice, got %v", data["name"])
	}
	// JSON numbers decode as float64
	if data["age"] != float64(30) {
		t.Errorf("expected age=30, got %v", data["age"])
	}
	if data["active"] != true {
		t.Errorf("expected active=true, got %v", data["active"])
	}
	tags, ok := data["tags"].([]interface{})
	if !ok || len(tags) != 2 {
		t.Errorf("expected tags with 2 elements, got %v", data["tags"])
	}
}

func TestFetchData_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not valid json`))
	}))
	defer server.Close()

	_, err := FetchData(server.URL)
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "error unmarshaling JSON") {
		t.Errorf("expected unmarshal error, got %v", err)
	}
}

func TestFetchData_InvalidURL(t *testing.T) {
	_, err := FetchData("http://127.0.0.1:0/nonexistent")
	if err == nil {
		t.Fatal("expected an error for invalid URL, got nil")
	}
	if !strings.Contains(err.Error(), "error fetching data") {
		t.Errorf("expected fetch error, got %v", err)
	}
}

func TestFetchData_EmptyJSONObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	data, err := FetchData(server.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty map, got %v", data)
	}
}
