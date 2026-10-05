package workflow

import (
	"testing"
)

func TestConfigurationValidation(t *testing.T) {
	c := fixture(t)
	t.Setenv("VSC_DATA_DIR", c.DataDir)
	t.Setenv("VSC_DEV_MODE", "")
	t.Setenv("VSC_DIRECTORIES", `["~/Code","/tmp/a,b"]`)
	t.Setenv("VSC_LIMIT", "0")
	if _, e := LoadConfig(); e == nil {
		t.Fatal("invalid limit accepted")
	}
	t.Setenv("VSC_LIMIT", "50")
	got, e := LoadConfig()
	if e != nil || len(got.Roots) != 2 || got.Roots[1] != "/tmp/a,b" {
		t.Fatalf("roots: %+v %v", got, e)
	}
}
