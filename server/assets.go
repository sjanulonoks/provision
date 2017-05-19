package server

// Yes - Twice - once to get the basic pieces in place to let swagger run, then the final parts
//
//go:generate ../tools/download-assets.sh ../embedded
//go:generate go-bindata -prefix ../embedded/assets -pkg embedded -o ../embedded/embed.go ../embedded/assets/...
//go:generate swagger generate spec -i ./swagger.base.yml -o ../embedded/assets/swagger.json
//go:generate ../tools/build-all-license.sh .. embedded/assets/ALL-LICENSE
//go:generate ../tools/build-all-license.sh .. ALL-LICENSE
//go:generate swagger generate client  -f ../embedded/assets/swagger.json -A DigitalRebarProvision --principal User -t .. --template-dir ../override
//go:generate env GOOS=linux GOARCH=amd64 go build -o ../embedded/assets/drpcli.amd64.linux ../cmds/drpcli/drpcli.go
//go:generate go-bindata -prefix ../embedded/assets -pkg embedded -o ../embedded/embed.go ../embedded/assets/...

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/digitalrebar/provision/embedded"
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
		"drpcli.amd64.linux": "files",

		// General ISO things
		"explode_iso.sh": "",

		// Sledgehammer things
		"jq": "files",

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
		"libcom32.c32": "",
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
