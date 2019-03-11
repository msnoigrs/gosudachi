// +build dev

package data

import (
	"net/http"
)

var Assets http.FileSystem = http.Dir("./root")
