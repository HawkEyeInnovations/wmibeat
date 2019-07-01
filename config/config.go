// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Period    time.Duration `config:"period"`
	Namespace string        `config:"namespace"`
	Classes   []ClassConfig
}

type ClassConfig struct {
	Class       string   `config:"class"`
	Fields      []string `config:"fields"`
	WhereClause string   `config:"whereclause"`
}

var DefaultConfig = Config{
	Period: 1 * time.Second,
}
