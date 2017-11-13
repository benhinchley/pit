package pit

import (
	"os"
	"testing"

	"github.com/sanity-io/litter"
)

func TestFindPackages(t *testing.T) {
	wd, _ := os.Getwd()
	pkgs, err := FindPackages(wd)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(litter.Sdump(pkgs))
}

func TestPackageRepository(t *testing.T) {
	wd, _ := os.Getwd()
	pkgs, err := FindPackages(wd)
	if err != nil {
		t.Error(err)
		return
	}

	for _, pkg := range pkgs {
		r, err := pkg.Repository()
		if err != nil {
			t.Error(err)
			return
		}
		c, _ := r.Config()
		t.Logf("pkg: %s repo config: %+v", pkg.Name, litter.Sdump(c.Remotes))
	}
}
