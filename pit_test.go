package pit

import (
	"os"
	"testing"
)

func TestFindPackages(t *testing.T) {
	wd, _ := os.Getwd()
	_, err := FindPackages(wd)
	if err != nil {
		t.Error(err)
		return
	}
	// t.Log(litter.Sdump(pkgs))
}

func TestPackageRepository(t *testing.T) {
	wd, _ := os.Getwd()
	pkgs, err := FindPackages(wd)
	if err != nil {
		t.Error(err)
		return
	}

	for _, pkg := range pkgs {
		_, err := pkg.Repository()
		if err != nil {
			t.Error(err)
			return
		}
		// c, _ := r.Config()
		// t.Logf("pkg: %s repo config: %+v", pkg.Name, litter.Sdump(c.Remotes))
	}
}
