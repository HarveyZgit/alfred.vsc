package workflow

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCLIErrorContract(t *testing.T) {
	c := fixture(t)
	t.Setenv("VSC_DATA_DIR", c.DataDir)
	t.Setenv("VSC_CACHE_DIR", c.CacheDir)
	t.Setenv("VSC_DEV_MODE", "")
	t.Setenv("VSC_DIRECTORIES", "")
	t.Setenv("VSC_IDE_PATH", c.VSCodeDir)
	t.Setenv("VSC_NO_BACKGROUND", "1")
	var out, errout bytes.Buffer
	if code := Run([]string{"open", "--target", "https://invalid"}, &out, &errout); code == 0 {
		t.Fatal("invalid open succeeded")
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"query", "nothing"}, &out, &errout); code != 0 || !json.Valid(out.Bytes()) {
		t.Fatalf("query must produce JSON: %d %s", code, out.String())
	}
}
