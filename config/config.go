// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"time"
)

type Config struct {
	Queries []QueryConfig
}

type Field struct {
	Name  string `config:"name"`
	IsInt bool   `config:"int"`
}

type QueryConfig struct {
	Period      time.Duration `config:"period"`
	Class       string        `config:"class"`
	Fields      []Field       `config:"fields"`
	WhereClause string        `config:"whereclause"`
	Namespace   string        `config:"namespace"`
}

var DefaultConfig = QueryConfig{
	Period: 1 * time.Second,
}
