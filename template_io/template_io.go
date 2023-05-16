package template_io

import (
	"context"
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

func (loader *InMemoryLoader) Open(name string) (io.Reader, error) {
	if content, ok := loader.templates[name]; ok {
		return strings.NewReader(content), nil
	}
	return nil, fmt.Errorf("template %s not found", name)
}

func (loader *InMemoryLoader) Exists(name string) bool {
	_, exists := loader.templates[name]
	return exists
}

func BuildConnectionString(host string, user string, password string, dbname string) string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", host, user, password, dbname)
}

func (loader *InMemoryLoader) LoadTemplatesFromDB(connStr string, tableName string) error {
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), fmt.Sprintf("SELECT set_name, name, content FROM %s", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var setName, name, content string
		err = rows.Scan(&setName, &name, &content)
		if err != nil {
			return err
		}
		fullName := setName + ":" + name
		loader.templates[fullName] = content
	}
	return nil
}
