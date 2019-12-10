package php_test

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/NiR-/notpecl/extindex"
	"github.com/NiR-/zbuild/pkg/builddef"
	"github.com/NiR-/zbuild/pkg/defkinds/php"
	"github.com/NiR-/zbuild/pkg/mocks"
	"github.com/NiR-/zbuild/pkg/pkgsolver"
	"github.com/NiR-/zbuild/pkg/statesolver"
	"github.com/golang/mock/gomock"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"
)

type updateLocksTC struct {
	// deffile is a path to a yaml file containing a php build definition
	deffile   string
	handler   *php.PHPHandler
	pkgSolver pkgsolver.PackageSolver
	// expected is the path to a lock file in testdata/ folder
	expected    string
	expectedErr error
}

var rawDebianOSRelease = []byte(`PRETTY_NAME="Debian GNU/Linux 10 (buster)"
NAME="Debian GNU/Linux"
VERSION_ID="10"
VERSION="10 (buster)"
VERSION_CODENAME=buster
ID=debian
HOME_URL="https://www.debian.org/"
SUPPORT_URL="https://www.debian.org/support"
BUG_REPORT_URL="https://bugs.debian.org/"`)

func initSuccessfullyUpdateLocksTC(t *testing.T, mockCtrl *gomock.Controller) updateLocksTC {
	solver := mocks.NewMockStateSolver(mockCtrl)
	solver.EXPECT().FromImage("docker.io/library/php:7.3-fpm-buster").Times(1)
	solver.EXPECT().ReadFile(
		gomock.Any(),
		"/etc/os-release",
		gomock.Any(),
	).Return(rawDebianOSRelease, nil)

	solver.EXPECT().FromBuildContext(gomock.Any()).Times(1)
	solver.EXPECT().ReadFile(
		gomock.Any(), "composer.lock", gomock.Any(),
	).Return([]byte{}, statesolver.FileNotFound)

	pkgSolver := mocks.NewMockPackageSolver(mockCtrl)
	pkgSolver.EXPECT().Configure(gomock.Any()).Times(1)
	pkgSolver.EXPECT().ResolveVersions(map[string]string{
		"git":          "*",
		"libicu-dev":   "*",
		"libpcre3-dev": "*",
		"libssl-dev":   "*",
		"libxml2-dev":  "*",
		"openssl":      "*",
		"unzip":        "*",
		"zlib1g-dev":   "*",
	}).Return(map[string]string{
		"git":          "git-version",
		"libicu-dev":   "libicu-dev-version",
		"libpcre3-dev": "libpcre3-dev-version",
		"libssl-dev":   "libssl-dev-version",
		"libxml2-dev":  "libxml2-dev-version",
		"openssl":      "openssl-version",
		"unzip":        "unzip-version",
		"zlib1g-dev":   "zlib1g-dev-version",
	}, nil)

	h := php.NewPHPHandler()
	h.NotPecl = h.NotPecl.WithExtensionIndex(extindex.ExtIndex{
		"redis": extindex.ExtVersions{
			"5.1.1": extindex.Beta,
			"5.1.0": extindex.Stable,
		},
		"yaml": extindex.ExtVersions{
			"1.0.0": extindex.Stable,
			"1.1.0": extindex.Beta,
		},
	})
	h.WithSolver(solver)

	return updateLocksTC{
		deffile:   "testdata/locks/without-stages.yml",
		handler:   h,
		pkgSolver: pkgSolver,
		expected:  "testdata/locks/without-stages.lock",
	}
}

var rawAlpineOSRelease = []byte(`NAME="Alpine Linux"
ID=alpine
VERSION_ID=3.10.3
PRETTY_NAME="Alpine Linux v3.10"
HOME_URL="https://alpinelinux.org/"
BUG_REPORT_URL="https://bugs.alpinelinux.org/"
`)

func failToUpdateLocksForAlpineBaseImageTC(t *testing.T, mockCtrl *gomock.Controller) updateLocksTC {
	solver := mocks.NewMockStateSolver(mockCtrl)
	solver.EXPECT().FromImage("docker.io/library/php:7.3-fpm-buster").Times(1)
	solver.EXPECT().ReadFile(
		gomock.Any(),
		"/etc/os-release",
		gomock.Any(),
	).Return(rawAlpineOSRelease, nil)

	kindHandler := php.NewPHPHandler()
	kindHandler.WithSolver(solver)

	return updateLocksTC{
		deffile:     "testdata/locks/without-stages.yml",
		handler:     kindHandler,
		pkgSolver:   mocks.NewMockPackageSolver(mockCtrl),
		expectedErr: xerrors.New("unsupported OS \"alpine\": only debian-based base images are supported"),
	}
}

func TestUpdateLocks(t *testing.T) {
	testcases := map[string]func(*testing.T, *gomock.Controller) updateLocksTC{
		"successfully update locks":                        initSuccessfullyUpdateLocksTC,
		"fail to update locks for alpine based base image": failToUpdateLocksForAlpineBaseImageTC,
	}

	for tcname := range testcases {
		tcinit := testcases[tcname]

		t.Run(tcname, func(t *testing.T) {
			t.Parallel()

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			tc := tcinit(t, mockCtrl)
			genericDef := loadGenericDef(t, tc.deffile, "")

			var locks builddef.Locks
			var rawLocks []byte
			var err error

			ctx := context.Background()
			locks, err = tc.handler.UpdateLocks(ctx, tc.pkgSolver, &genericDef)
			if err == nil {
				rawLocks, err = locks.RawLocks()
			}

			if *flagTestdata {
				if tc.expected != "" {
					writeTestdata(t, tc.expected, string(rawLocks))
				}
				return
			}

			if tc.expectedErr != nil {
				if err == nil || err.Error() != tc.expectedErr.Error() {
					t.Fatalf("Expected error: %v\nGot: %v", tc.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			expectedRaw := loadTestdata(t, tc.expected)
			if expectedRaw != string(rawLocks) {
				t.Fatalf("Expected: %s\nGot: %s", expectedRaw, rawLocks)
			}
		})
	}
}

func loadGenericDef(t *testing.T, filepath, lockpath string) builddef.BuildDef {
	raw := loadTestdata(t, filepath)

	var def builddef.BuildDef
	if err := yaml.Unmarshal([]byte(raw), &def); err != nil {
		t.Fatal(err)
	}

	if lockpath != "" {
		lockContent, err := ioutil.ReadFile(lockpath)
		if err != nil {
			t.Fatal(err)
		}
		def.RawLocks = lockContent
	}

	return def
}