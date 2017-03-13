package main

// Yes - Twice - once to get the basic pieces in place to let swagger run, then the final parts
//
//go:generate ../tools/download-assets.sh ../embedded
//go:generate go-bindata -prefix ../embedded/assets -pkg embedded -o ../embedded/embed.go ../embedded/assets/...
//go:generate swagger generate spec -o ../embedded/assets/swagger.json
//go:generate ../tools/build-all-license.sh .. embedded/assets/ALL-LICENSE
//go:generate ../tools/build-all-license.sh .. ALL-LICENSE
//go:generate swagger generate client  -f ../embedded/assets/swagger.json -A RocketSkates --principal User -t ..
//go:generate go build -o ../embedded/assets/rscli ../cli/...
//go:generate go-bindata -prefix ../embedded/assets -pkg embedded -o ../embedded/embed.go ../embedded/assets/...

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/rackn/rocket-skates/embedded"
)

func ExtractAssets(fileRoot string) error {
	dirs := []string{"isos", "files", "machines", "pxelinux.cfg"}
	for _, dest := range dirs {
		destDir := path.Join(fileRoot, dest)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}
	}

	assets := map[string]string{
		// General LICENSE thing
		"ALL-LICENSE": "",

		// CLI things
		"rscli": "files",

		// General ISO things
		"explode_iso.sh": "",

		// Sledgehammer things
		"install-sledgehammer.sh": "",
		"start-up.sh":             "machines",
		"jq":                      "files",
		"default.ipxe.tmpl":       "",
		"elilo.conf.tmpl":         "",
		"default.tmpl":            "pxelinux.cfg",

		// General Boot things
		"bootia32.efi": "",
		"bootia64.efi": "",
		"bootx64.efi":  "",
		"esxi.0":       "",
		"ipxe.efi":     "",
		"ipxe.pxe":     "",
		"ldlinux.c32":  "",
		"libutil.c32":  "",
		"lpxelinux.0":  "",
		"pxechn.c32":   "",
		"wimboot":      "",
	}

	for src, dest := range assets {
		buf, err := embedded.Asset(src)
		if err != nil {
			return fmt.Errorf("No such embedded asset %s: %v", src, err)
		}
		info, err := embedded.AssetInfo(src)
		if err != nil {
			return fmt.Errorf("No mode info for embedded asset %s: %v", src, err)
		}

		if strings.HasSuffix(src, ".tmpl") {
			var doc bytes.Buffer

			t, err := template.New("test").Parse(string(buf))
			if err != nil {
				return err
			}

			params := struct {
				ProvIp      string
				ProvFileURL string
				ProvApiURL  string
			}{
				ProvIp:      c_opts.OurAddress,
				ProvFileURL: fmt.Sprintf("http://%s:%d", c_opts.OurAddress, c_opts.StaticPort),
				ProvApiURL:  fmt.Sprintf("https://%s:%d", c_opts.OurAddress, c_opts.ApiPort),
			}
			err = t.Execute(&doc, params)
			if err != nil {
				return err
			}
			buf = doc.Bytes()
			src = strings.TrimSuffix(src, ".tmpl")
		}

		destFile := path.Join(fileRoot, dest, src)
		destDir := path.Dir(destFile)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}

		if err := ioutil.WriteFile(destFile, buf, info.Mode()); err != nil {
			return err
		}
		if err := os.Chtimes(destFile, info.ModTime(), info.ModTime()); err != nil {
			return err
		}
	}

	return nil
}
