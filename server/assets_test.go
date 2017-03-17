package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestExtractAssets(t *testing.T) {
	tgt, err := ioutil.TempDir("", "assetTest")
	if err != nil {
		t.Errorf("Failed to make temporary directory for asset tests: %v", err)
		return
	}
	defer os.RemoveAll(tgt)
	if err := ExtractAssets(tgt); err != nil {
		t.Errorf("Could not extract assets: %v", err)
		return
	}

	files := []string{
		"ALL-LICENSE",
		"explode_iso.sh",
		"install-sledgehammer.sh",
		"machines/start-up.sh",
		"files/jq",
		"files/rscli",
		"bootia32.efi",
		"bootia64.efi",
		"esxi.0",
		"ipxe.efi",
		"ipxe.pxe",
		"ldlinux.c32",
		"libutil.c32",
		"lpxelinux.0",
		"pxechn.c32",
		"wimboot",
	}

	for _, f := range files {
		if _, err := os.Stat(path.Join(tgt, f)); os.IsNotExist(err) {
			t.Errorf("File %s does NOT exist, but should.", f)
		}
	}

}
