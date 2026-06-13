//go:build tools
// +build tools

// Package main records build-time tool dependencies so go mod tidy retains them.
package main

import (
	_ "github.com/signintech/gopdf"
	_ "golang.org/x/image/font/opentype"
)
