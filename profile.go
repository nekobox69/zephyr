// Package zephyr Create at 2021-01-19 13:48
package zephyr

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// Profile 项目信息
type Profile struct {
	Name      string // Application name
	Version   string // Application version
	BuildTime string // Compilation date
	GoVersion string // Golang version
	Mode      string // Deployment mode
	URL       string // URL
	Desc      string // Description.
}

// NewProfile profile
func NewProfile(url, version, goVersion, buildTime, mode string) *Profile {
	return &Profile{
		URL:       url,
		Name:      filepath.Base(os.Args[0]),
		Version:   version,
		BuildTime: buildTime,
		GoVersion: goVersion,
		Mode:      mode,
		Desc:      fmt.Sprintf("%s application.\n", filepath.Base(os.Args[0])),
	}
}

// Description profile desc
func (p *Profile) Description() string {
	desc := p.Desc
	mode := fmt.Sprintf("\tdeployment mode: %s\n", p.Mode)
	bt := fmt.Sprintf("\tbuild time: %v\n", p.BuildTime)
	version := fmt.Sprintf("\tversion: %v\n", p.Version)
	goVersion := fmt.Sprintf("\tgo version: %v", p.GoVersion)
	return desc + mode + bt + version + goVersion
}

// Handler url handler
func (p *Profile) Handler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	b, err := json.Marshal(p)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(b)
}
