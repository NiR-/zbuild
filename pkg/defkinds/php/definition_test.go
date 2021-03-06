package php_test

import (
	"errors"
	"testing"
	"time"

	"github.com/NiR-/zbuild/pkg/builddef"
	"github.com/NiR-/zbuild/pkg/defkinds/php"
	"github.com/NiR-/zbuild/pkg/llbutils"
	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"gopkg.in/yaml.v2"
)

type newDefinitionTC struct {
	file        string
	lockFile    string
	expected    php.Definition
	expectedErr error
}

func initParseRawDefinitionWithoutStagesTC() newDefinitionTC {
	file := "testdata/def/without-stages.yml"
	lockFile := "testdata/def/without-stages.lock"

	isFPM := true
	isDev := true
	isNotDev := false
	inferMode := false

	devStage := emptyStage()
	prodStage := emptyStage()

	return newDefinitionTC{
		file:     file,
		lockFile: lockFile,
		expected: php.Definition{
			BaseStage: php.Stage{
				ExternalFiles:  []llbutils.ExternalFile{},
				SystemPackages: &builddef.VersionMap{},
				FPM:            &isFPM,
				Extensions: &builddef.VersionMap{
					"intl":      "*",
					"pdo_mysql": "*",
					"soap":      "*",
				},
				GlobalDeps: &builddef.VersionMap{},
				ConfigFiles: builddef.PathsMap{
					"docker/app/php.ini":  "${php_ini}",
					"docker/app/fpm.conf": "${fpm_conf}",
				},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					APCU:                  false,
					ClassmapAuthoritative: true,
				},
				Sources:      []string{"./src"},
				Integrations: []string{"blackfire"},
				StatefulDirs: []string{"./public/uploads"},
				Healthcheck: &builddef.HealthcheckConfig{
					HealthcheckFCGI: &builddef.HealthcheckFCGI{
						Path:     "/ping",
						Expected: "pong",
					},
					Type:     builddef.HealthcheckTypeFCGI,
					Interval: 10 * time.Second,
					Timeout:  1 * time.Second,
					Retries:  3,
				},
				PostInstall: []string{
					"some more commands",
					"another one",
				},
			},
			Version:       "7.4.0",
			MajMinVersion: "7.4",
			BaseImage:     "docker.io/library/php:7.4.0-fpm-buster",
			Infer:         &inferMode,
			Stages: map[string]php.DerivedStage{
				"dev": {
					DeriveFrom: "base",
					Dev:        &isDev,
					Stage:      devStage,
				},
				"prod": {
					DeriveFrom: "base",
					Dev:        &isNotDev,
					Stage:      prodStage,
				},
			},
			Locks: php.DefinitionLocks{
				ExtensionDir: "/usr/local/lib/php/extensions/no-debug-non-zts-20190902/",
				Stages: map[string]php.StageLocks{
					"dev": {
						SystemPackages: map[string]string{
							"git":        "1:2.1.4-2.1+deb8u7",
							"libicu-dev": "52.1-8+deb8u7",
						},
						Extensions: map[string]string{
							"intl":      "*",
							"pdo_mysql": "*",
							"soap":      "*",
						},
					},
				},
			},
		},
	}
}

func initParseRawDefinitionWithStagesTC() newDefinitionTC {
	devStageDevMode := true
	prodStageDevMode := false
	isFPM := true
	isNotFPM := false
	workerCmd := []string{"bin/worker"}
	inferMode := true

	devStage := emptyStage()
	devStage.ConfigFiles = builddef.PathsMap{
		"docker/app/php.dev.ini": "${php_ini}",
	}

	prodStage := emptyStage()
	prodStage.ConfigFiles = builddef.PathsMap{
		"docker/app/php.prod.ini": "${php_ini}",
	}
	prodStage.Healthcheck = &builddef.HealthcheckConfig{
		HealthcheckFCGI: &builddef.HealthcheckFCGI{
			Path:     "/ping",
			Expected: "pong",
		},
		Type:     builddef.HealthcheckTypeFCGI,
		Interval: 10 * time.Second,
		Timeout:  1 * time.Second,
		Retries:  3,
	}
	prodStage.Integrations = []string{"blackfire"}

	return newDefinitionTC{
		file:     "testdata/def/merge-all.yml",
		lockFile: "",
		expected: php.Definition{
			BaseStage: php.Stage{
				ExternalFiles:  []llbutils.ExternalFile{},
				SystemPackages: &builddef.VersionMap{},
				FPM:            &isFPM,
				Extensions: &builddef.VersionMap{
					"intl":      "*",
					"pdo_mysql": "*",
					"soap":      "*",
				},
				GlobalDeps: &builddef.VersionMap{},
				ConfigFiles: builddef.PathsMap{
					"docker/app/fpm.conf": "${fpm_conf}",
				},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					APCU:                  false,
					ClassmapAuthoritative: true,
				},
				Sources:      []string{"generated/"},
				Integrations: []string{},
				StatefulDirs: []string{"public/uploads"},
				Healthcheck: &builddef.HealthcheckConfig{
					Type: builddef.HealthcheckTypeDisabled,
				},
				PostInstall: []string{"echo some command"},
			},
			Version:       "7.4.0",
			MajMinVersion: "7.4",
			BaseImage:     "docker.io/library/php:7.4.0-fpm-buster",
			Infer:         &inferMode,
			Stages: map[string]php.DerivedStage{
				"dev": {
					DeriveFrom: "",
					Dev:        &devStageDevMode,
					Stage:      devStage,
				},
				"prod": {
					DeriveFrom: "",
					Dev:        &prodStageDevMode,
					Stage:      prodStage,
				},
				"worker": {
					DeriveFrom: "prod",
					Stage: php.Stage{
						ComposerDumpFlags: &php.ComposerDumpFlags{
							APCU:                  true,
							ClassmapAuthoritative: false,
						},
						Sources:      []string{"worker/"},
						StatefulDirs: []string{"data/imports"},
						PostInstall:  []string{"echo some other command"},
						FPM:          &isNotFPM,
						Command:      &workerCmd,
					},
				},
			},
			Locks: php.DefinitionLocks{},
		},
	}
}

func initParseRawDefinitionWithWebserverTC() newDefinitionTC {
	devStageDevMode := true
	prodStageDevMode := false
	inferMode := false
	baseStageFPM := true

	baseStage := emptyStage()
	baseStage.Healthcheck = &builddef.HealthcheckConfig{
		HealthcheckFCGI: &builddef.HealthcheckFCGI{
			Path:     "/ping",
			Expected: "pong",
		},
		Type:     builddef.HealthcheckTypeFCGI,
		Interval: 10 * time.Second,
		Timeout:  1 * time.Second,
		Retries:  3,
	}
	baseStage.FPM = &baseStageFPM
	baseStage.ComposerDumpFlags = &php.ComposerDumpFlags{
		ClassmapAuthoritative: true,
	}

	devStage := emptyStage()
	prodStage := emptyStage()

	return newDefinitionTC{
		file: "testdata/def/with-webserver.yml",
		expected: php.Definition{
			BaseStage:     baseStage,
			Version:       "7.4.0",
			MajMinVersion: "7.4",
			BaseImage:     "docker.io/library/php:7.4.0-fpm-buster",
			Infer:         &inferMode,
			Stages: map[string]php.DerivedStage{
				"dev": {
					DeriveFrom: "base",
					Dev:        &devStageDevMode,
					Stage:      devStage,
				},
				"prod": {
					DeriveFrom: "base",
					Dev:        &prodStageDevMode,
					Stage:      prodStage,
				},
			},
		},
	}
}

func initParseRawDefinitionWithCustomFCGIHealthcheckTC() newDefinitionTC {
	devStageDevMode := true
	prodStageDevMode := false
	inferMode := false
	baseStageFPM := true

	devStage := emptyStage()
	prodStage := emptyStage()

	return newDefinitionTC{
		file: "testdata/def/with-custom-fcgi-healthcheck.yml",
		expected: php.Definition{
			BaseStage: php.Stage{
				ExternalFiles:  []llbutils.ExternalFile{},
				SystemPackages: &builddef.VersionMap{},
				Extensions:     &builddef.VersionMap{},
				GlobalDeps:     &builddef.VersionMap{},
				ConfigFiles:    builddef.PathsMap{},
				Sources:        []string{},
				Integrations:   []string{},
				StatefulDirs:   []string{},
				PostInstall:    []string{},
				FPM:            &baseStageFPM,
				ComposerDumpFlags: &php.ComposerDumpFlags{
					ClassmapAuthoritative: true,
				},
				Healthcheck: &builddef.HealthcheckConfig{
					HealthcheckFCGI: &builddef.HealthcheckFCGI{
						Path:     "/some-custom-path",
						Expected: "some-output",
					},
					Type:     builddef.HealthcheckTypeFCGI,
					Interval: 20 * time.Second,
					Timeout:  5 * time.Second,
					Retries:  3,
				},
			},
			Version:       "7.4.0",
			MajMinVersion: "7.4",
			BaseImage:     "docker.io/library/php:7.4.0-fpm-buster",
			Infer:         &inferMode,
			Stages: map[string]php.DerivedStage{
				"dev": {
					DeriveFrom: "base",
					Dev:        &devStageDevMode,
					Stage:      devStage,
				},
				"prod": {
					DeriveFrom: "base",
					Dev:        &prodStageDevMode,
					Stage:      prodStage,
				},
			},
		},
	}
}

func initParseRawDefinitionWithCustomSourceContextTC() newDefinitionTC {
	devStageDevMode := true
	prodStageDevMode := false
	inferMode := true
	baseStageFPM := true

	devStage := emptyStage()
	prodStage := emptyStage()

	return newDefinitionTC{
		file: "testdata/def/with-source-context.yml",
		expected: php.Definition{
			BaseStage: php.Stage{
				ExternalFiles:  []llbutils.ExternalFile{},
				SystemPackages: &builddef.VersionMap{},
				Extensions:     &builddef.VersionMap{},
				GlobalDeps:     &builddef.VersionMap{},
				ConfigFiles:    builddef.PathsMap{},
				Sources: []string{
					"src/",
				},
				Integrations: []string{},
				StatefulDirs: []string{},
				PostInstall:  []string{},
				FPM:          &baseStageFPM,
				ComposerDumpFlags: &php.ComposerDumpFlags{
					ClassmapAuthoritative: true,
				},
				Healthcheck: &builddef.HealthcheckConfig{
					HealthcheckFCGI: &builddef.HealthcheckFCGI{
						Path:     "/ping",
						Expected: "pong",
					},
					Type:     builddef.HealthcheckTypeFCGI,
					Interval: 10 * time.Second,
					Timeout:  1 * time.Second,
					Retries:  3,
				},
			},
			Infer: &inferMode,
			SourceContext: &builddef.Context{
				Source: "git://github.com/api-platform/demo",
				Type:   builddef.ContextTypeGit,
				GitContext: builddef.GitContext{
					Reference: "5ecd217",
				},
			},
			Stages: map[string]php.DerivedStage{
				"dev": {
					DeriveFrom: "base",
					Dev:        &devStageDevMode,
					Stage:      devStage,
				},
				"prod": {
					DeriveFrom: "base",
					Dev:        &prodStageDevMode,
					Stage:      prodStage,
				},
			},
		},
	}
}

func initFailToParseUnknownPropertiesTC() newDefinitionTC {
	return newDefinitionTC{
		file:        "testdata/def/with-invalid-properties.yml",
		lockFile:    "",
		expectedErr: errors.New("could not decode build manifest: invalid config parameter: foo"),
	}
}

func initAlpineWithoutBaseImageTC() newDefinitionTC {
	file := "testdata/def/empty.yml"

	isFPM := true
	isDev := true
	isNotDev := false
	inferMode := true

	devStage := emptyStage()
	prodStage := emptyStage()

	return newDefinitionTC{
		file: file,
		expected: php.Definition{
			BaseStage: php.Stage{
				ExternalFiles:  []llbutils.ExternalFile{},
				SystemPackages: &builddef.VersionMap{},
				FPM:            &isFPM,
				Extensions:     &builddef.VersionMap{},
				GlobalDeps:     &builddef.VersionMap{},
				ConfigFiles:    builddef.PathsMap{},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					APCU:                  false,
					ClassmapAuthoritative: true,
				},
				Sources:      []string{},
				Integrations: []string{},
				StatefulDirs: []string{},
				Healthcheck: &builddef.HealthcheckConfig{
					Type: builddef.HealthcheckTypeDisabled,
				},
				PostInstall: []string{},
			},
			Version:       "7.4.0",
			Alpine:        true,
			BaseImage:     "docker.io/library/php:7.4.0-fpm-alpine",
			MajMinVersion: "7.4",
			Infer:         &inferMode,
			Stages: map[string]php.DerivedStage{
				"dev": {
					DeriveFrom: "base",
					Dev:        &isDev,
					Stage:      devStage,
				},
				"prod": {
					DeriveFrom: "base",
					Dev:        &isNotDev,
					Stage:      prodStage,
				},
			},
			Locks: php.DefinitionLocks{},
		},
	}
}

func TestNewKind(t *testing.T) {
	if *flagTestdata {
		return
	}

	testcases := map[string]func() newDefinitionTC{
		"without stages":                   initParseRawDefinitionWithoutStagesTC,
		"with stages":                      initParseRawDefinitionWithStagesTC,
		"with webserver":                   initParseRawDefinitionWithWebserverTC,
		"with custom fcgi healthcheck":     initParseRawDefinitionWithCustomFCGIHealthcheckTC,
		"with source context":              initParseRawDefinitionWithCustomSourceContextTC,
		"fail to parse unknown properties": initFailToParseUnknownPropertiesTC,
		"alpine without base image":        initAlpineWithoutBaseImageTC,
	}

	for tcname := range testcases {
		tcinit := testcases[tcname]

		t.Run(tcname, func(t *testing.T) {
			t.Parallel()
			tc := tcinit()

			generic := loadBuildDef(t, tc.file)
			if tc.lockFile != "" {
				generic.RawLocks = loadDefLocks(t, tc.lockFile)
			}

			def, err := php.NewKind(generic)
			if tc.expectedErr != nil {
				if err == nil || tc.expectedErr.Error() != err.Error() {
					t.Fatalf("Expected: %v\nGot: %v", tc.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if diff := deep.Equal(def, tc.expected); diff != nil {
				t.Fatal(diff)
			}
		})
	}
}

type resolveStageTC struct {
	file               string
	lockFile           string
	stage              string
	osrelease          builddef.OSRelease
	composerLockLoader func(*php.StageDefinition) error
	expected           php.StageDefinition
	expectedErr        error
}

func initSuccessfullyResolveDefaultDevStageTC(t *testing.T, mockCtrl *gomock.Controller) resolveStageTC {
	file := "testdata/def/without-stages.yml"
	lockFile := "testdata/def/without-stages.lock"
	isFPM := true

	return resolveStageTC{
		file:     file,
		lockFile: lockFile,
		stage:    "dev",
		osrelease: builddef.OSRelease{
			Name: "debian",
		},
		composerLockLoader: func(stageDef *php.StageDefinition) error {
			return nil
		},
		expected: php.StageDefinition{
			Name:          "dev",
			Version:       "7.4.0",
			MajMinVersion: "7.4",
			Infer:         false,
			Dev:           true,
			PlatformReqs:  &builddef.VersionMap{},
			Stage: php.Stage{
				ExternalFiles: []llbutils.ExternalFile{
					{
						URL:         "https://blackfire.io/api/v1/releases/probe/php/linux/amd64/72",
						Compressed:  true,
						Pattern:     "blackfire-*.so",
						Destination: "/usr/local/lib/php/extensions/no-debug-non-zts-20190902/blackfire.so",
						Mode:        0644,
					},
				},
				SystemPackages: &builddef.VersionMap{},
				FPM:            &isFPM,
				Extensions: &builddef.VersionMap{
					"intl":      "*",
					"pdo_mysql": "*",
					"soap":      "*",
				},
				GlobalDeps:   &builddef.VersionMap{},
				Sources:      []string{"./src"},
				StatefulDirs: []string{"./public/uploads"},
				ConfigFiles: builddef.PathsMap{
					"docker/app/php.ini":  "${php_ini}",
					"docker/app/fpm.conf": "${fpm_conf}",
				},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					APCU:                  false,
					ClassmapAuthoritative: true,
				},
				Integrations: []string{"blackfire"},
				PostInstall:  []string{"some more commands", "another one"},
			},
			DefLocks: php.DefinitionLocks{
				ExtensionDir: "/usr/local/lib/php/extensions/no-debug-non-zts-20190902/",
				OSRelease: builddef.OSRelease{
					Name: "debian",
				},
				Stages: map[string]php.StageLocks{
					"dev": {
						SystemPackages: map[string]string{
							"git":        "1:2.1.4-2.1+deb8u7",
							"libicu-dev": "52.1-8+deb8u7",
						},
						Extensions: map[string]string{
							"intl":      "*",
							"pdo_mysql": "*",
							"soap":      "*",
						},
					},
				},
			},
		},
	}
}

func initSuccessfullyResolveWorkerStageTC(t *testing.T, mockCtrl *gomock.Controller) resolveStageTC {
	isNotFPM := false
	workerCmd := []string{"bin/worker"}

	return resolveStageTC{
		file:     "testdata/def/worker.yml",
		lockFile: "",
		stage:    "prod",
		osrelease: builddef.OSRelease{
			Name: "debian",
		},
		composerLockLoader: mockComposerLockLoader(
			map[string]string{
				"mbstring": "*",
			},
		),
		expected: php.StageDefinition{
			Name:          "prod",
			Version:       "7.4",
			MajMinVersion: "7.4",
			Infer:         true,
			Dev:           false,
			PlatformReqs: &builddef.VersionMap{
				"mbstring": "*",
			},
			Stage: php.Stage{
				ExternalFiles: []llbutils.ExternalFile{},
				SystemPackages: &builddef.VersionMap{
					"zlib1g-dev": "*",
					"unzip":      "*",
					"git":        "*",
					"libzip-dev": "*",
				},
				FPM:     &isNotFPM,
				Command: &workerCmd,
				Extensions: &builddef.VersionMap{
					"mbstring": "*",
					"zip":      "*",
				},
				GlobalDeps: &builddef.VersionMap{},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					APCU:                  false,
					ClassmapAuthoritative: true,
				},
				ConfigFiles:  builddef.PathsMap{},
				Sources:      []string{"bin/", "src/"},
				Integrations: []string{},
				StatefulDirs: []string{},
				PostInstall:  []string{},
			},
			DefLocks: php.DefinitionLocks{
				OSRelease: builddef.OSRelease{
					Name: "debian",
				},
			},
		},
	}
}

func initFailToResolveUnknownStageTC(t *testing.T, mockCtrl *gomock.Controller) resolveStageTC {
	file := "testdata/def/without-stages.yml"
	lockFile := "testdata/def/without-stages.lock"

	composerLockLoader := mockComposerLockLoader(map[string]string{})

	return resolveStageTC{
		file:     file,
		lockFile: lockFile,
		stage:    "foo",
		osrelease: builddef.OSRelease{
			Name: "debian",
		},
		composerLockLoader: composerLockLoader,
		expectedErr:        errors.New(`stage "foo" not found`),
	}
}

func initFailToResolveStageWithCyclicDepsTC(t *testing.T, mockCtrl *gomock.Controller) resolveStageTC {
	composerLockLoader := mockComposerLockLoader(map[string]string{})

	return resolveStageTC{
		file:     "testdata/def/cyclic-stage-deps.yml",
		lockFile: "",
		stage:    "dev",
		osrelease: builddef.OSRelease{
			Name: "debian",
		},
		composerLockLoader: composerLockLoader,
		expectedErr:        errors.New(`there's a cyclic dependency between "dev" and itself`),
	}
}

func initRemoveDefaultExtensionsTC(t *testing.T, mockCtrl *gomock.Controller) resolveStageTC {
	fpm := true

	composerLockLoader := mockComposerLockLoader(map[string]string{})

	return resolveStageTC{
		file:     "testdata/def/remove-default-exts.yml",
		lockFile: "",
		stage:    "dev",
		osrelease: builddef.OSRelease{
			Name: "debian",
		},
		composerLockLoader: composerLockLoader,
		expected: php.StageDefinition{
			Name:          "dev",
			Version:       "7.4",
			MajMinVersion: "7.4",
			Infer:         true,
			Dev:           true,
			PlatformReqs:  &builddef.VersionMap{},
			Stage: php.Stage{
				ExternalFiles: []llbutils.ExternalFile{},
				SystemPackages: &builddef.VersionMap{
					"zlib1g-dev":    "*",
					"unzip":         "*",
					"git":           "*",
					"libsodium-dev": "*",
					"libzip-dev":    "*",
				},
				FPM: &fpm,
				Extensions: &builddef.VersionMap{
					"zip":        "*",
					"mbstring":   "*",
					"reflection": "*",
					"sodium":     "*",
					"spl":        "*",
					"standard":   "*",
					"filter":     "*",
					"json":       "*",
					"session":    "*",
				},
				GlobalDeps:  &builddef.VersionMap{},
				ConfigFiles: builddef.PathsMap{},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					ClassmapAuthoritative: true,
				},
				Sources:      []string{},
				Integrations: []string{},
				StatefulDirs: []string{},
				PostInstall:  []string{},
			},
			DefLocks: php.DefinitionLocks{
				OSRelease: builddef.OSRelease{
					Name: "debian",
				},
			},
		},
	}
}

// This TC ensures that the extensions infered from composer.lock aren't
// erasing version constraints defined in the zbuildfile.
func initPreservePredefinedExtensionConstraintsTC(t *testing.T, mockCtrl *gomock.Controller) resolveStageTC {
	fpm := true

	return resolveStageTC{
		file:     "testdata/def/with-predefined-extension.yml",
		lockFile: "",
		stage:    "dev",
		osrelease: builddef.OSRelease{
			Name: "debian",
		},
		composerLockLoader: mockComposerLockLoader(
			map[string]string{
				"redis": "*",
			},
		),
		expected: php.StageDefinition{
			Name:          "dev",
			Version:       "7.4",
			MajMinVersion: "7.4",
			Infer:         true,
			Dev:           true,
			PlatformReqs: &builddef.VersionMap{
				"redis": "*",
			},
			Stage: php.Stage{
				ExternalFiles: []llbutils.ExternalFile{},
				SystemPackages: &builddef.VersionMap{
					"zlib1g-dev": "*",
					"unzip":      "*",
					"git":        "*",
					"libzip-dev": "*",
				},
				FPM: &fpm,
				Extensions: &builddef.VersionMap{
					"zip":   "*",
					"redis": "^5.1",
				},
				GlobalDeps:  &builddef.VersionMap{},
				ConfigFiles: builddef.PathsMap{},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					ClassmapAuthoritative: true,
				},
				Sources:      []string{},
				Integrations: []string{},
				StatefulDirs: []string{},
				PostInstall:  []string{},
			},
			DefLocks: php.DefinitionLocks{
				OSRelease: builddef.OSRelease{
					Name: "debian",
				},
			},
		},
	}
}

func initFailWhenComposerFlagsAreInvalidTC(t *testing.T, _ *gomock.Controller) resolveStageTC {
	composerLockLoader := mockComposerLockLoader(map[string]string{})

	return resolveStageTC{
		file:     "testdata/def/invalid-composer-flags.yml",
		lockFile: "",
		stage:    "dev",
		osrelease: builddef.OSRelease{
			Name: "debian",
		},
		composerLockLoader: composerLockLoader,
		expectedErr:        errors.New(`invalid final stage config: you can't use both --apcu and --classmap-authoritative flags. See https://getcomposer.org/doc/articles/autoloader-optimization.md`),
	}
}

func initInferAlpinePackagesRequiredByExtsTC(t *testing.T, mockCtrl *gomock.Controller) resolveStageTC {
	fpmMode := true

	return resolveStageTC{
		file:     "testdata/def/alpine.yml",
		lockFile: "",
		stage:    "prod",
		osrelease: builddef.OSRelease{
			Name: "alpine",
		},
		composerLockLoader: mockComposerLockLoader(
			map[string]string{},
		),
		expected: php.StageDefinition{
			Name:          "prod",
			Version:       "7.4",
			MajMinVersion: "7.4",
			Infer:         true,
			Dev:           false,
			PlatformReqs:  &builddef.VersionMap{},
			Stage: php.Stage{
				FPM:           &fpmMode,
				ExternalFiles: []llbutils.ExternalFile{},
				SystemPackages: &builddef.VersionMap{
					"git":            "*",
					"icu-dev":        "*",
					"libzip-dev":     "*",
					"postgresql-dev": "*",
					"unzip":          "*",
				},
				Extensions: &builddef.VersionMap{
					"apcu":      "*",
					"intl":      "*",
					"opcache":   "*",
					"pdo_pgsql": "*",
					"zip":       "*",
				},
				GlobalDeps: &builddef.VersionMap{},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					APCU:                  false,
					ClassmapAuthoritative: true,
				},
				Sources:      []string{},
				ConfigFiles:  builddef.PathsMap{},
				Integrations: []string{},
				StatefulDirs: []string{},
				PostInstall:  []string{},
				Healthcheck: &builddef.HealthcheckConfig{
					Type: builddef.HealthcheckTypeDisabled,
				},
			},
			DefLocks: php.DefinitionLocks{
				OSRelease: builddef.OSRelease{
					Name: "alpine",
				},
			},
		},
	}
}

func TestResolveStageDefinition(t *testing.T) {
	if *flagTestdata {
		return
	}

	testcases := map[string]func(*testing.T, *gomock.Controller) resolveStageTC{
		"successfully resolve default dev stage":    initSuccessfullyResolveDefaultDevStageTC,
		"successfully resolve worker stage":         initSuccessfullyResolveWorkerStageTC,
		"fail to resolve unknown stage":             initFailToResolveUnknownStageTC,
		"fail to resolve stage with cyclic deps":    initFailToResolveStageWithCyclicDepsTC,
		"fail when composer flags are invalid":      initFailWhenComposerFlagsAreInvalidTC,
		"remove default extensions":                 initRemoveDefaultExtensionsTC,
		"preserve predefined extension constraints": initPreservePredefinedExtensionConstraintsTC,
		"infer alpine packages required by exts":    initInferAlpinePackagesRequiredByExtsTC,
	}

	for tcname := range testcases {
		tcinit := testcases[tcname]

		t.Run(tcname, func(t *testing.T) {
			t.Parallel()

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			tc := tcinit(t, mockCtrl)

			generic := loadBuildDef(t, tc.file)
			if tc.lockFile != "" {
				generic.RawLocks = loadDefLocks(t, tc.lockFile)
			}

			def, err := php.NewKind(generic)
			if err != nil {
				t.Fatal(err)
			}
			def.Locks.OSRelease = tc.osrelease

			stageDef, err := def.ResolveStageDefinition(tc.stage, tc.composerLockLoader, false)
			if tc.expectedErr != nil {
				if err == nil || err.Error() != tc.expectedErr.Error() {
					t.Fatalf("Expected: %v\nGot: %v", tc.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if diff := deep.Equal(stageDef, tc.expected); diff != nil {
				t.Fatal(diff)
			}
		})
	}
}

func TestComposerDumpFlags(t *testing.T) {
	if !*flagTestdata {
		return
	}

	testcases := map[string]struct {
		obj         php.ComposerDumpFlags
		expected    string
		expectedErr error
	}{
		"with apcu optimization": {
			obj:      php.ComposerDumpFlags{APCU: true},
			expected: "--no-dev --optimize --apcu",
		},
		"with authoritative classmap": {
			obj:      php.ComposerDumpFlags{ClassmapAuthoritative: true},
			expected: "--no-dev --optimize --classmap-authoritative",
		},
		"with no particular optimization": {
			obj:      php.ComposerDumpFlags{},
			expected: "--no-dev --optimize",
		},
	}

	for tcname := range testcases {
		tc := testcases[tcname]

		t.Run(tcname, func(t *testing.T) {
			t.Parallel()

			out, err := tc.obj.Flags()
			if tc.expectedErr != nil {
				if err == nil || err.Error() != tc.expectedErr.Error() {
					t.Fatalf("Expected error: %v\nGot: %v", tc.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if out != tc.expected {
				t.Fatalf("Expected: %s\nGot: %s", tc.expected, out)
			}
		})
	}
}

func loadBuildDef(t *testing.T, filepath string) *builddef.BuildDef {
	raw := loadRawTestdata(t, filepath)

	var def builddef.BuildDef
	if err := yaml.Unmarshal(raw, &def); err != nil {
		t.Fatal(err)
	}

	return &def
}

// @TODO: use a proper ComposerLock struct
func mockComposerLockLoader(
	PlatformReqs map[string]string,
) func(*php.StageDefinition) error {
	return func(stageDef *php.StageDefinition) error {
		*stageDef.PlatformReqs = builddef.VersionMap(PlatformReqs)
		return nil
	}
}

type mergeDefinitionTC struct {
	base       func() php.Definition
	overriding func() php.Definition
	expected   func() php.Definition
}

func TestMergeDefinition(t *testing.T) {
	testcases := map[string]mergeDefinitionTC{
		"merge base stage with base": {
			base: func() php.Definition {
				return php.Definition{
					BaseStage: php.Stage{
						Sources: []string{"src/"},
					},
				}
			},
			overriding: func() php.Definition {
				return php.Definition{
					BaseStage: php.Stage{
						Sources: []string{"bin/"},
					},
				}
			},
			expected: func() php.Definition {
				baseStage := emptyStage()
				baseStage.Sources = []string{"src/", "bin/"}

				return php.Definition{
					BaseStage: baseStage,
					Stages:    php.DerivedStageSet{},
				}
			},
		},
		"merge base stage without base": {
			base: func() php.Definition {
				return php.Definition{}
			},
			overriding: func() php.Definition {
				return php.Definition{
					BaseStage: php.Stage{
						Sources: []string{"bin/"},
					},
				}
			},
			expected: func() php.Definition {
				baseStage := emptyStage()
				baseStage.Sources = []string{"bin/"}

				return php.Definition{
					BaseStage: baseStage,
					Stages:    php.DerivedStageSet{},
				}
			},
		},
		"merge base image with base": {
			base: func() php.Definition {
				return php.Definition{
					BaseImage: "docker.io/library/php:7.3-fpm-buster",
				}
			},
			overriding: func() php.Definition {
				return php.Definition{
					BaseImage: "docker.io/library/php:7.4-fpm-buster",
				}
			},
			expected: func() php.Definition {
				return php.Definition{
					BaseImage: "docker.io/library/php:7.4-fpm-buster",
					BaseStage: emptyStage(),
					Stages:    php.DerivedStageSet{},
				}
			},
		},
		"merge base image without base": {
			base: func() php.Definition {
				return php.Definition{}
			},
			overriding: func() php.Definition {
				return php.Definition{
					BaseImage: "docker.io/library/php:7.4-fpm-buster",
				}
			},
			expected: func() php.Definition {
				return php.Definition{
					BaseImage: "docker.io/library/php:7.4-fpm-buster",
					BaseStage: emptyStage(),
					Stages:    php.DerivedStageSet{},
				}
			},
		},
		"merge version with base": {
			base: func() php.Definition {
				return php.Definition{
					Version: "7.3",
				}
			},
			overriding: func() php.Definition {
				return php.Definition{
					Version: "7.4",
				}
			},
			expected: func() php.Definition {
				return php.Definition{
					Version:   "7.4",
					BaseStage: emptyStage(),
					Stages:    php.DerivedStageSet{},
				}
			},
		},
		"merge version without base": {
			base: func() php.Definition {
				return php.Definition{}
			},
			overriding: func() php.Definition {
				return php.Definition{
					Version: "7.4",
				}
			},
			expected: func() php.Definition {
				return php.Definition{
					Version:   "7.4",
					BaseStage: emptyStage(),
					Stages:    php.DerivedStageSet{},
				}
			},
		},
		"merge alpine with base": {
			base: func() php.Definition {
				return php.Definition{
					Alpine: true,
				}
			},
			overriding: func() php.Definition {
				return php.Definition{
					Alpine: false,
				}
			},
			expected: func() php.Definition {
				return php.Definition{
					Alpine:    false,
					BaseStage: emptyStage(),
					Stages:    php.DerivedStageSet{},
				}
			},
		},
		"merge alpine without base": {
			base: func() php.Definition {
				return php.Definition{}
			},
			overriding: func() php.Definition {
				return php.Definition{
					Alpine: true,
				}
			},
			expected: func() php.Definition {
				return php.Definition{
					Alpine:    true,
					BaseStage: emptyStage(),
					Stages:    php.DerivedStageSet{},
				}
			},
		},
		"merge infer with base": {
			base: func() php.Definition {
				infer := true
				return php.Definition{
					Infer: &infer,
				}
			},
			overriding: func() php.Definition {
				infer := false
				return php.Definition{
					Infer: &infer,
				}
			},
			expected: func() php.Definition {
				infer := false
				return php.Definition{
					Infer:     &infer,
					BaseStage: emptyStage(),
					Stages:    php.DerivedStageSet{},
				}
			},
		},
		"merge infer without base": {
			base: func() php.Definition {
				return php.Definition{}
			},
			overriding: func() php.Definition {
				infer := true
				return php.Definition{
					Infer: &infer,
				}
			},
			expected: func() php.Definition {
				infer := true
				return php.Definition{
					Infer:     &infer,
					BaseStage: emptyStage(),
					Stages:    php.DerivedStageSet{},
				}
			},
		},
		"ignore nil infer": {
			base: func() php.Definition {
				infer := true
				return php.Definition{
					Infer: &infer,
				}
			},
			overriding: func() php.Definition {
				return php.Definition{}
			},
			expected: func() php.Definition {
				infer := true
				return php.Definition{
					Infer:     &infer,
					BaseStage: emptyStage(),
					Stages:    php.DerivedStageSet{},
				}
			},
		},
		"merge stages with base": {
			base: func() php.Definition {
				return php.Definition{
					Stages: php.DerivedStageSet{
						"staging": php.DerivedStage{
							DeriveFrom: "dev",
						},
					},
				}
			},
			overriding: func() php.Definition {
				return php.Definition{
					Stages: php.DerivedStageSet{
						"staging": php.DerivedStage{
							DeriveFrom: "prod",
						},
					},
				}
			},
			expected: func() php.Definition {
				return php.Definition{
					BaseStage: emptyStage(),
					Stages: php.DerivedStageSet{
						"staging": php.DerivedStage{
							DeriveFrom: "prod",
							Stage:      emptyStage(),
						},
					},
				}
			},
		},
		"merge stages without base": {
			base: func() php.Definition {
				return php.Definition{
					Stages: php.DerivedStageSet{},
				}
			},
			overriding: func() php.Definition {
				return php.Definition{
					Stages: php.DerivedStageSet{
						"staging": php.DerivedStage{
							DeriveFrom: "prod",
						},
					},
				}
			},
			expected: func() php.Definition {
				return php.Definition{
					BaseStage: emptyStage(),
					Stages: php.DerivedStageSet{
						"staging": php.DerivedStage{
							DeriveFrom: "prod",
						},
					},
				}
			},
		},
	}

	for tcname := range testcases {
		tc := testcases[tcname]

		t.Run(tcname, func(t *testing.T) {
			t.Parallel()

			base := tc.base()
			new := base.Merge(tc.overriding())

			if diff := deep.Equal(new, tc.expected()); diff != nil {
				t.Fatal(diff)
			}

			if diff := deep.Equal(base, tc.base()); diff != nil {
				t.Fatalf("Base stages don't match: %v", diff)
			}
		})
	}
}

type mergeStageTC struct {
	base       func() php.Stage
	overriding php.Stage
	expected   func() php.Stage
}

func emptyStage() php.Stage {
	return php.Stage{
		ExternalFiles:  []llbutils.ExternalFile{},
		SystemPackages: &builddef.VersionMap{},
		Extensions:     &builddef.VersionMap{},
		GlobalDeps:     &builddef.VersionMap{},
		ConfigFiles:    builddef.PathsMap{},
		Sources:        []string{},
		Integrations:   []string{},
		StatefulDirs:   []string{},
		PostInstall:    []string{},
	}
}

func initMergeExternalFilesWithBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				ExternalFiles: []llbutils.ExternalFile{
					{
						URL:         "https://github.com/some/tool",
						Destination: "/usr/local/bin/some-tool",
						Mode:        0750,
					},
				},
			}
		},
		overriding: php.Stage{
			ExternalFiles: []llbutils.ExternalFile{
				{
					URL:         "https://github.com/some/other/tool",
					Destination: "/usr/local/bin/some-other-tool",
					Mode:        0750,
				},
			},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.ExternalFiles = []llbutils.ExternalFile{
				{
					URL:         "https://github.com/some/tool",
					Destination: "/usr/local/bin/some-tool",
					Mode:        0750,
				},
				{
					URL:         "https://github.com/some/other/tool",
					Destination: "/usr/local/bin/some-other-tool",
					Mode:        0750,
				},
			}
			return s
		},
	}
}

func initMergeExternalFilesWithoutBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{}
		},
		overriding: php.Stage{
			ExternalFiles: []llbutils.ExternalFile{
				{
					URL:         "https://github.com/some/other/tool",
					Destination: "/usr/local/bin/some-other-tool",
					Mode:        0750,
				},
			},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.ExternalFiles = []llbutils.ExternalFile{
				{
					URL:         "https://github.com/some/other/tool",
					Destination: "/usr/local/bin/some-other-tool",
					Mode:        0750,
				},
			}
			return s
		},
	}
}

func initMergeSystemPackagesWithBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				SystemPackages: &builddef.VersionMap{
					"curl": "*",
				},
			}
		},
		overriding: php.Stage{
			SystemPackages: &builddef.VersionMap{
				"chromium": "*",
			},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.SystemPackages = &builddef.VersionMap{
				"curl":     "*",
				"chromium": "*",
			}
			return s
		},
	}
}

func initMergeSystemPackagesWithoutBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{}
		},
		overriding: php.Stage{
			SystemPackages: &builddef.VersionMap{
				"chromium": "*",
			},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.SystemPackages = &builddef.VersionMap{
				"chromium": "*",
			}
			return s
		},
	}
}

func initMergeFPMWithBaseTC() mergeStageTC {
	baseFPM := true
	overridingFPM := false
	expectedFPM := false

	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				FPM: &baseFPM,
			}
		},
		overriding: php.Stage{
			FPM: &overridingFPM,
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.FPM = &expectedFPM
			return s
		},
	}
}

func initMergeFPMWithoutBaseTC() mergeStageTC {
	overridingFPM := true
	expectedFPM := true

	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{}
		},
		overriding: php.Stage{
			FPM: &overridingFPM,
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.FPM = &expectedFPM
			return s
		},
	}
}

func initMergeCommandWithBaseTC() mergeStageTC {
	baseCmd := []string{"bin/some-worker"}
	overridingCmd := []string{"bin/some-other-worker"}
	expectedCmd := []string{"bin/some-other-worker"}

	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				Command: &baseCmd,
			}
		},
		overriding: php.Stage{
			Command: &overridingCmd,
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.Command = &expectedCmd
			return s
		},
	}
}

func initMergeCommandWithoutBaseTC() mergeStageTC {
	overridingCmd := []string{"bin/some-other-worker"}
	expectedCmd := []string{"bin/some-other-worker"}

	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{}
		},
		overriding: php.Stage{
			Command: &overridingCmd,
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.Command = &expectedCmd
			return s
		},
	}
}

func initMergeExtensionsWithBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				Extensions: &builddef.VersionMap{
					"apcu": "*",
				},
			}
		},
		overriding: php.Stage{
			Extensions: &builddef.VersionMap{
				"opcache": "*",
			},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.Extensions = &builddef.VersionMap{
				"apcu":    "*",
				"opcache": "*",
			}
			return s
		},
	}
}

func initMergeExtensionsWithoutBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{}
		},
		overriding: php.Stage{
			Extensions: &builddef.VersionMap{
				"opcache": "*",
			},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.Extensions = &builddef.VersionMap{
				"opcache": "*",
			}
			return s
		},
	}
}

func initMergeConfigFilesWithBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				ConfigFiles: builddef.PathsMap{
					"docker/app/php.dev.ini":  "${php_ini}",
					"docker/app/fpm.dev.conf": "${fpm_conf}",
				},
			}
		},
		overriding: php.Stage{
			ConfigFiles: builddef.PathsMap{
				"docker/app/php.prod.ini":  "${php_ini}",
				"docker/app/fpm.prod.conf": "${fpm_conf}",
			},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.ConfigFiles = builddef.PathsMap{
				"docker/app/php.prod.ini":  "${php_ini}",
				"docker/app/fpm.prod.conf": "${fpm_conf}",
			}
			return s
		},
	}
}

func initMergeConfigFilesWithoutBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{}
		},
		overriding: php.Stage{
			ConfigFiles: builddef.PathsMap{
				"docker/app/php.prod.ini":  "${php_ini}",
				"docker/app/fpm.prod.conf": "${fpm_conf}",
			},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.ConfigFiles = builddef.PathsMap{
				"docker/app/php.prod.ini":  "${php_ini}",
				"docker/app/fpm.prod.conf": "${fpm_conf}",
			}
			return s
		},
	}
}

func initMergeComposerDumpFlagsWithBaseTC() mergeStageTC {
	baseFlags := php.ComposerDumpFlags{
		ClassmapAuthoritative: true,
	}
	overridingFlags := php.ComposerDumpFlags{
		APCU: true,
	}
	expectedFlags := php.ComposerDumpFlags{
		APCU: true,
	}

	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				ComposerDumpFlags: &baseFlags,
			}
		},
		overriding: php.Stage{
			ComposerDumpFlags: &overridingFlags,
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.ComposerDumpFlags = &expectedFlags
			return s
		},
	}
}

func initMergeComposerDumpFlagsWithoutBaseTC() mergeStageTC {
	overridingFlags := php.ComposerDumpFlags{
		APCU: true,
	}
	expectedFlags := php.ComposerDumpFlags{
		APCU: true,
	}

	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{}
		},
		overriding: php.Stage{
			ComposerDumpFlags: &overridingFlags,
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.ComposerDumpFlags = &expectedFlags
			return s
		},
	}
}

func initMergeSourcesWithBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				Sources: []string{"src/"},
			}
		},
		overriding: php.Stage{
			Sources: []string{"bin/worker"},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.Sources = []string{"src/", "bin/worker"}
			return s
		},
	}
}

func initMergeSourcesWithoutBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{}
		},
		overriding: php.Stage{
			Sources: []string{"bin/worker"},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.Sources = []string{"bin/worker"}
			return s
		},
	}
}

func initMergeIntegrationsWithBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				Integrations: []string{"blackfire"},
			}
		},
		overriding: php.Stage{
			Integrations: []string{"some-other"},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.Integrations = []string{"blackfire", "some-other"}
			return s
		},
	}
}

func initMergeIntegrationsWithoutBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{}
		},
		overriding: php.Stage{
			Integrations: []string{"some-other"},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.Integrations = []string{"some-other"}
			return s
		},
	}
}

func initMergeStatefulDirsWithBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				StatefulDirs: []string{"var/sessions/"},
			}
		},
		overriding: php.Stage{
			StatefulDirs: []string{"public/uploads/"},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.StatefulDirs = []string{"var/sessions/", "public/uploads/"}
			return s
		},
	}
}

func initMergeStatefulDirsWithoutBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{}
		},
		overriding: php.Stage{
			StatefulDirs: []string{"public/uploads/"},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.StatefulDirs = []string{"public/uploads/"}
			return s
		},
	}
}

func initMergeHealthcheckWithBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				Healthcheck: &builddef.HealthcheckConfig{
					HealthcheckFCGI: &builddef.HealthcheckFCGI{
						Path:     "/ping",
						Expected: "pong",
					},
					Type:     builddef.HealthcheckTypeFCGI,
					Interval: 10 * time.Second,
					Timeout:  1 * time.Second,
					Retries:  3,
				},
			}
		},
		overriding: php.Stage{
			Healthcheck: &builddef.HealthcheckConfig{
				Type: builddef.HealthcheckTypeDisabled,
			},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.Healthcheck = &builddef.HealthcheckConfig{
				Type: builddef.HealthcheckTypeDisabled,
			}
			return s
		},
	}
}

func initMergeHealthcheckWithoutBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{}
		},
		overriding: php.Stage{
			Healthcheck: &builddef.HealthcheckConfig{
				Type: builddef.HealthcheckTypeDisabled,
			},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.Healthcheck = &builddef.HealthcheckConfig{
				Type: builddef.HealthcheckTypeDisabled,
			}
			return s
		},
	}
}

func initMergePostInstallWithBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				PostInstall: []string{"some-step"},
			}
		},
		overriding: php.Stage{
			PostInstall: []string{"some-other-step"},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.PostInstall = []string{"some-step", "some-other-step"}
			return s
		},
	}
}

func initMergePostInstallWithoutBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{}
		},
		overriding: php.Stage{
			PostInstall: []string{"some-other"},
		},
		expected: func() php.Stage {
			s := emptyStage()
			s.PostInstall = []string{"some-other"}
			return s
		},
	}
}

func initMergeGlobalDepsWithBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				GlobalDeps: &builddef.VersionMap{
					"symfony/flex": "*",
				},
			}
		},
		overriding: php.Stage{
			GlobalDeps: &builddef.VersionMap{
				"symfony/flex":      "1.6.0",
				"hirak/prestissimo": "0.3.9",
			},
		},
		expected: func() php.Stage {
			stage := emptyStage()
			stage.GlobalDeps = &builddef.VersionMap{
				"symfony/flex":      "1.6.0",
				"hirak/prestissimo": "0.3.9",
			}

			return stage
		},
	}
}

func initMergeGlobalDepsWithoutBaseTC() mergeStageTC {
	return mergeStageTC{
		base: func() php.Stage {
			return php.Stage{
				GlobalDeps: &builddef.VersionMap{},
			}
		},
		overriding: php.Stage{
			GlobalDeps: &builddef.VersionMap{
				"symfony/flex":      "1.6.0",
				"hirak/prestissimo": "0.3.9",
			},
		},
		expected: func() php.Stage {
			stage := emptyStage()
			stage.GlobalDeps = &builddef.VersionMap{
				"symfony/flex":      "1.6.0",
				"hirak/prestissimo": "0.3.9",
			}

			return stage
		},
	}
}

func TestStageMerge(t *testing.T) {
	testcases := map[string]func() mergeStageTC{
		"merge external files with base":         initMergeExternalFilesWithBaseTC,
		"merge external files without base":      initMergeExternalFilesWithoutBaseTC,
		"merge system packages with base":        initMergeSystemPackagesWithBaseTC,
		"merge system packages without base":     initMergeSystemPackagesWithoutBaseTC,
		"merge fpm with base":                    initMergeFPMWithBaseTC,
		"merge fpm without base":                 initMergeFPMWithoutBaseTC,
		"merge command with base":                initMergeCommandWithBaseTC,
		"merge command without base":             initMergeCommandWithoutBaseTC,
		"merge extensions with base":             initMergeExtensionsWithBaseTC,
		"merge extensions without base":          initMergeExtensionsWithoutBaseTC,
		"merge global deps with base":            initMergeGlobalDepsWithBaseTC,
		"merge global deps without base":         initMergeGlobalDepsWithoutBaseTC,
		"merge config files with base":           initMergeConfigFilesWithBaseTC,
		"merge config files without base":        initMergeConfigFilesWithoutBaseTC,
		"merge composer dump flags with base":    initMergeComposerDumpFlagsWithBaseTC,
		"merge composer dump flags without base": initMergeComposerDumpFlagsWithoutBaseTC,
		"merge sources with base":                initMergeSourcesWithBaseTC,
		"merge sources without base":             initMergeSourcesWithoutBaseTC,
		"merge integrations with base":           initMergeIntegrationsWithBaseTC,
		"merge integrations without base":        initMergeIntegrationsWithoutBaseTC,
		"merge stateful dirs with base":          initMergeStatefulDirsWithBaseTC,
		"merge stateful dirs without base":       initMergeStatefulDirsWithoutBaseTC,
		"merge healthcheck with base":            initMergeHealthcheckWithBaseTC,
		"merge healthcheck without base":         initMergeHealthcheckWithoutBaseTC,
		"merge post install with base":           initMergePostInstallWithBaseTC,
		"merge post install without base":        initMergePostInstallWithoutBaseTC,
	}

	for tcname := range testcases {
		tcinit := testcases[tcname]

		t.Run(tcname, func(t *testing.T) {
			tc := tcinit()
			base := tc.base()
			new := base.Merge(tc.overriding)

			if diff := deep.Equal(new, tc.expected()); diff != nil {
				t.Fatal(diff)
			}

			if diff := deep.Equal(base, tc.base()); diff != nil {
				t.Fatalf("Base stages don't match: %v", diff)
			}
		})
	}
}
