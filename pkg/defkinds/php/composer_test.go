package php_test

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/NiR-/zbuild/pkg/builddef"
	"github.com/NiR-/zbuild/pkg/defkinds/php"
	"github.com/NiR-/zbuild/pkg/mocks"
	"github.com/NiR-/zbuild/pkg/statesolver"
	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"golang.org/x/xerrors"
)

type loadComposerLockTC struct {
	initial     php.StageDefinition
	context     *builddef.Context
	solver      statesolver.StateSolver
	expected    php.StageDefinition
	expectedErr error
}

func initSuccessfullyLoadAndParseComposerLockTC(
	t *testing.T,
	mockCtrl *gomock.Controller,
) loadComposerLockTC {
	solver := mocks.NewMockStateSolver(mockCtrl)
	solver.EXPECT().FromContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	raw := loadRawTestdata(t, "testdata/composer/valid/composer.lock")
	solver.EXPECT().ReadFile(
		gomock.Any(), "composer.lock", gomock.Any(),
	).Return(raw, nil)

	return loadComposerLockTC{
		initial: php.StageDefinition{
			Dev: true,
		},
		context: &builddef.Context{
			Type:   builddef.ContextTypeLocal,
			Source: "context",
		},
		solver: solver,
		expected: php.StageDefinition{
			Dev: true,
			LockedPackages: map[string]string{
				"clue/stream-filter":    "v1.4.0",
				"webmozart/assert":      "1.4.0",
				"sebastian/environment": "4.2.2",
				"sebastian/exporter":    "3.1.0",
			},
			PlatformReqs: map[string]string{
				"mbstring": "*",
			},
		},
	}
}

func initLoadComposerLockFromGitSubdirTC(t *testing.T, mockCtrl *gomock.Controller) loadComposerLockTC {
	solver := mocks.NewMockStateSolver(mockCtrl)
	solver.EXPECT().FromContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	raw := loadRawTestdata(t, "testdata/composer/valid/composer.lock")
	solver.EXPECT().ReadFile(
		gomock.Any(), "/sub/dir/composer.lock", gomock.Any(),
	).Return(raw, nil)

	return loadComposerLockTC{
		initial: php.StageDefinition{
			Dev: true,
		},
		context: &builddef.Context{
			Type:   builddef.ContextTypeGit,
			Source: "git://github.com/some/repo",
			GitContext: builddef.GitContext{
				Path: "sub/dir",
			},
		},
		solver: solver,
		expected: php.StageDefinition{
			Dev: true,
			LockedPackages: map[string]string{
				"clue/stream-filter":    "v1.4.0",
				"webmozart/assert":      "1.4.0",
				"sebastian/environment": "4.2.2",
				"sebastian/exporter":    "3.1.0",
			},
			PlatformReqs: map[string]string{
				"mbstring": "*",
			},
		},
	}
}

func initSilentlyFailWhenComposerLockFileDoesNotExistTC(
	t *testing.T,
	mockCtrl *gomock.Controller,
) loadComposerLockTC {
	solver := mocks.NewMockStateSolver(mockCtrl)
	solver.EXPECT().FromContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	solver.EXPECT().ReadFile(
		gomock.Any(), "composer.lock", gomock.Any(),
	).Return([]byte{}, statesolver.FileNotFound)

	return loadComposerLockTC{
		initial: php.StageDefinition{
			LockedPackages: map[string]string{},
			PlatformReqs:   map[string]string{},
		},
		context: &builddef.Context{
			Type:   builddef.ContextTypeLocal,
			Source: "context",
		},
		solver: solver,
		expected: php.StageDefinition{
			LockedPackages: map[string]string{},
			PlatformReqs:   map[string]string{},
		},
	}
}

func initFailToLoadBrokenComposerLockFileTC(
	t *testing.T,
	mockCtrl *gomock.Controller,
) loadComposerLockTC {
	solver := mocks.NewMockStateSolver(mockCtrl)
	solver.EXPECT().FromContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	raw := loadRawTestdata(t, "testdata/composer/broken/composer.lock")
	solver.EXPECT().ReadFile(
		gomock.Any(), "composer.lock", gomock.Any(),
	).Return(raw, nil)

	return loadComposerLockTC{
		initial: php.StageDefinition{},
		context: &builddef.Context{
			Type:   builddef.ContextTypeLocal,
			Source: "context",
		},
		solver:      solver,
		expectedErr: xerrors.New("could not unmarshal composer.lock: unexpected end of JSON input"),
	}
}

func TestLoadComposerLock(t *testing.T) {
	if *flagTestdata {
		return
	}

	testcases := map[string]func(*testing.T, *gomock.Controller) loadComposerLockTC{
		"successfully load and parse composer.lock file":       initSuccessfullyLoadAndParseComposerLockTC,
		"load composer.lock from git subdir":                   initLoadComposerLockFromGitSubdirTC,
		"silently fail when composer.lock file does not exist": initSilentlyFailWhenComposerLockFileDoesNotExistTC,
		"fail to load broken composer.lock file":               initFailToLoadBrokenComposerLockFileTC,
	}

	for tcname := range testcases {
		tcinit := testcases[tcname]

		t.Run(tcname, func(t *testing.T) {
			t.Parallel()

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			tc := tcinit(t, mockCtrl)
			stage := tc.initial

			ctx := context.Background()

			err := php.LoadComposerLock(ctx, tc.solver, &stage, tc.context)
			if tc.expectedErr != nil {
				if err == nil || tc.expectedErr.Error() != err.Error() {
					t.Fatalf("Expected error: %v\nGot: %v", tc.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if diff := deep.Equal(stage, tc.expected); diff != nil {
				t.Fatal(diff)
			}
		})
	}
}

func loadRawTestdata(t *testing.T, filepath string) []byte {
	raw, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
