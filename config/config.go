// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"strconv"
	"time"

	"github.com/elastic/go-ucfg"
	"github.com/go-ole/go-ole"
)

type Config struct {
	Queries []QueryConfig
}

// alias of Field which allows for the unpack without causing a stackoverflow
type cfgfield struct {
	Field
}

type Field struct {
	Name  string `config:"name"`
	IsInt bool   `config:"int"`
}

type QueryConfig struct {
	Period      time.Duration `config:"period"`
	Class       string        `config:"class"`
	Fields      []cfgfield    `config:"fields"`
	WhereClause string        `config:"whereclause"`
	Namespace   string        `config:"namespace"`
}

var DefaultConfig = QueryConfig{
	Period: 1 * time.Second,
}

func (f *Field) Convert(v *ole.VARIANT) (interface{}, error) {
	if f.IsInt {
		return strconv.Atoi(v.ToString())
	}

	return v.Value(), nil
}

func (f *cfgfield) Unpack(v interface{}) error {
	switch tv := v.(type) {
	case string:
		*f = cfgfield{
			Field: Field{
				Name: tv,
			},
		}
	default:
		cfg, err := ucfg.NewFrom(v)
		if err != nil {
			return err
		}
		field := Field{}
		if err := cfg.Unpack(&field); err != nil {
			return err
		}
		*f = cfgfield{Field: field}
	}
	return nil
}
