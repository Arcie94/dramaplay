package handlers

import (
	"dramabang/services/adapter"
)

var AdapterManager *adapter.Manager

func init() {
	AdapterManager = adapter.NewManager()
}
