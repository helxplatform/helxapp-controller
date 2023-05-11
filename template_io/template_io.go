package template_io

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v4"
)

// InMemoryLoader holds templates in memory
type InMemoryLoader struct {
	templates map[string]string
}

func NewInMemoryLoader() *InMemoryLoader {
	return &InMemoryLoader{templates: make(map[string]string)}
}

func (loader *InMemoryLoader) Open(name string) (io.ReadCloser, error) {
	if template, ok := loader.templates[name]; ok {
		return io.NopCloser(strings.NewReader(template)), nil
	}
	return nil, errors.New("template not found")
}

func (loader *InMemoryLoader) Exists(name string) bool {
	_, ok := loader.templates[name]
	return ok
}

func (loader *InMemoryLoader) LoadTemplatesFromDB(connStr string, tableName string) error {
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), fmt.Sprintf("SELECT name, content FROM %s", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, content string
		err = rows.Scan(&name, &content)
		if err != nil {
			return err
		}
		loader.templates[name] = content
	}
	return nil
}
