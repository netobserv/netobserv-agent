// Code generated by "libovsdb.modelgen"
// DO NOT EDIT.

package serverdb

import (
	"encoding/json"

	"github.com/ovn-org/libovsdb/model"
	"github.com/ovn-org/libovsdb/ovsdb"
)

// FullDatabaseModel returns the DatabaseModel object to be used in libovsdb
func FullDatabaseModel() (model.ClientDBModel, error) {
	return model.NewClientDBModel("_Server", map[string]model.Model{
		"Database": &Database{},
	})
}

var schema = `{
  "name": "_Server",
  "version": "1.2.0",
  "tables": {
    "Database": {
      "columns": {
        "cid": {
          "type": {
            "key": {
              "type": "uuid"
            },
            "min": 0,
            "max": 1
          }
        },
        "connected": {
          "type": "boolean"
        },
        "index": {
          "type": {
            "key": {
              "type": "integer"
            },
            "min": 0,
            "max": 1
          }
        },
        "leader": {
          "type": "boolean"
        },
        "model": {
          "type": {
            "key": {
              "type": "string",
              "enum": [
                "set",
                [
                  "standalone",
                  "clustered",
                  "relay"
                ]
              ]
            }
          }
        },
        "name": {
          "type": "string"
        },
        "schema": {
          "type": {
            "key": {
              "type": "string"
            },
            "min": 0,
            "max": 1
          }
        },
        "sid": {
          "type": {
            "key": {
              "type": "uuid"
            },
            "min": 0,
            "max": 1
          }
        }
      },
      "isRoot": true
    }
  }
}`

func Schema() ovsdb.DatabaseSchema {
	var s ovsdb.DatabaseSchema
	err := json.Unmarshal([]byte(schema), &s)
	if err != nil {
		panic(err)
	}
	return s
}
