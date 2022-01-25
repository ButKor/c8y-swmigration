package main

import (
	"bytes"
	"os"

	"github.com/dimiro1/banner"
	_ "github.com/dimiro1/banner/autoload"
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
)

func main() {
	isEnabled := true
	isColorEnabled := true
	banner.Init(colorable.NewColorableStdout(), isEnabled, isColorEnabled, bytes.NewBufferString(createBannerContent()))

	Logger.Info("Migration server started", zap.Any("Args", os.Args))
	RunServer()
	Logger.Info("Migration finished")
}

func createBannerContent() string {
	s := `{{ .Title "SW Migration" "slant" 2}}
	GoVersion:   {{ .GoVersion }}
	GOOS:        {{ .GOOS }}
	GOARCH:      {{ .GOARCH }}
	NumCPU:      {{ .NumCPU }}
	GOPATH:      {{ .GOPATH }}
	GOROOT:      {{ .GOROOT }}
	Compiler:    {{ .Compiler }}
	ENV:         {{ .Env "GOPATH" }}
	Now:         {{ .Now "2006-01-02 15:04:05" }}
	Author:      Korbinian Butz
	Description: Tool to migrate software repository entry entries towards the new Software Versions introduced with c8y v10.7

`
	return s
}
