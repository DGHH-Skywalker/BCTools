//go:build !windows

package kugoudb

const CanSelect = false

func discoverDatabase() string { return "" }
