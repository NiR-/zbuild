package php_test

import (
	"errors"
	"testing"

	"github.com/NiR-/zbuild/pkg/builddef"
	"github.com/NiR-/zbuild/pkg/defkinds/php"
	"github.com/NiR-/zbuild/pkg/llbutils"
	"github.com/go-test/deep"
	"golang.org/x/xerrors"
)

type newDefinitionTC struct {
	file        string
	lockFile    string
	expected    php.Definition
	expectedErr error
}

func initSuccessfullyParseRawDefinitionWithoutStagesTC() newDefinitionTC {
	file := "testdata/def/without-stages.yml"
	lockFile := "testdata/def/without-stages.lock"

	isFPM := true
	iniFile := "docker/app/php.ini"
	fpmConfigFile := "docker/app/fpm.conf"
	healthcheck := true
	isDev := true

	return newDefinitionTC{
		file:     file,
		lockFile: lockFile,
		expected: php.Definition{
			BaseStage: php.Stage{
				BaseConfig: builddef.BaseConfig{
					ExternalFiles:  []llbutils.ExternalFile{},
					SystemPackages: map[string]string{},
				},
				FPM: &isFPM,
				Extensions: map[string]string{
					"intl":      "*",
					"pdo_mysql": "*",
					"soap":      "*",
				},
				ConfigFiles: php.PHPConfigFiles{
					IniFile:       &iniFile,
					FPMConfigFile: &fpmConfigFile,
				},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					APCU:                  false,
					ClassmapAuthoritative: true,
				},
				SourceDirs:   []string{"./src"},
				ExtraScripts: []string{"./public/index.php"},
				Integrations: []string{"blackfire"},
				StatefulDirs: []string{"./public/uploads"},
				Healthcheck:  &healthcheck,
				PostInstall: []string{
					"some more commands",
					"another one",
				},
			},
			Version:       "7.4.0",
			MajMinVersion: "7.4",
			BaseImage:     "docker.io/library/php:7.4-fpm-buster",
			Infer:         false,
			Stages: map[string]php.DerivedStage{
				"dev": {
					DeriveFrom: "base",
					Dev:        &isDev,
				},
			},
			Locks: php.DefinitionLocks{
				Stages: map[string]php.StageLocks{
					"base": {
						BaseStageLocks: builddef.BaseStageLocks{
							SystemPackages: map[string]string{
								"git":        "1:2.1.4-2.1+deb8u7",
								"libicu-dev": "52.1-8+deb8u7",
							},
						},
						Extensions: map[string]string{
							"intl":      "*",
							"pdo_mysql": "*",
							"soap":      "*",
						},
					},
					"dev": {
						BaseStageLocks: builddef.BaseStageLocks{
							SystemPackages: map[string]string{
								"git":        "1:2.1.4-2.1+deb8u7",
								"libicu-dev": "52.1-8+deb8u7",
							},
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

func initSuccessfullyParseRawDefinitionWithStagesTC() newDefinitionTC {
	iniDevFile := "docker/app/php.dev.ini"
	iniProdFile := "docker/app/php.prod.ini"
	fpmConfigFile := "docker/app/fpm.conf"
	healthcheckEnabled := true
	healthcheckDisabled := false
	isDev := true
	isFPM := true
	isNotFPM := false
	workerCmd := "bin/worker"

	return newDefinitionTC{
		file:     "testdata/def/with-stages.yml",
		lockFile: "",
		expected: php.Definition{
			BaseStage: php.Stage{
				BaseConfig: builddef.BaseConfig{
					ExternalFiles:  []llbutils.ExternalFile{},
					SystemPackages: map[string]string{},
				},
				FPM: &isFPM,
				Extensions: map[string]string{
					"intl":      "*",
					"pdo_mysql": "*",
					"soap":      "*",
				},
				ConfigFiles: php.PHPConfigFiles{
					FPMConfigFile: &fpmConfigFile,
				},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					APCU:                  false,
					ClassmapAuthoritative: true,
				},
				SourceDirs: []string{"generated/"},
				ExtraScripts: []string{
					"gencode.php",
				},
				Integrations: []string{"symfony"},
				StatefulDirs: []string{"public/uploads"},
				Healthcheck:  &healthcheckDisabled,
				PostInstall:  []string{"echo some command"},
			},
			Version:       "7.4.0",
			MajMinVersion: "7.4",
			BaseImage:     "docker.io/library/php:7.4-fpm-buster",
			Infer:         true,
			Stages: map[string]php.DerivedStage{
				"dev": {
					DeriveFrom: "",
					Dev:        &isDev,
					Stage: php.Stage{
						ConfigFiles: php.PHPConfigFiles{
							IniFile: &iniDevFile,
						},
					},
				},
				"prod": {
					DeriveFrom: "",
					Stage: php.Stage{
						ConfigFiles: php.PHPConfigFiles{
							IniFile: &iniProdFile,
						},
						Healthcheck:  &healthcheckEnabled,
						Integrations: []string{"blackfire"},
					},
				},
				"worker": {
					DeriveFrom: "prod",
					Stage: php.Stage{
						ConfigFiles: php.PHPConfigFiles{},
						ComposerDumpFlags: &php.ComposerDumpFlags{
							APCU:                  true,
							ClassmapAuthoritative: false,
						},
						SourceDirs:   []string{"worker/"},
						ExtraScripts: []string{"bin/worker"},
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

func initFailToParseUnknownPropertiesTC() newDefinitionTC {
	file := "testdata/def/with-invalid-properties.yml"
	lockFile := "testdata/def/with-invalid-properties.lock"

	return newDefinitionTC{
		file:        file,
		lockFile:    lockFile,
		expectedErr: errors.New("could not decode build manifest: 1 error(s) decoding:\n\n* '' has invalid keys: foo"),
	}
}

func TestNewKind(t *testing.T) {
	if *flagTestdata {
		return
	}

	testcases := map[string]func() newDefinitionTC{
		"successfully parse raw definition without stages": initSuccessfullyParseRawDefinitionWithoutStagesTC,
		"successfully parse raw definition with stages":    initSuccessfullyParseRawDefinitionWithStagesTC,
		"fail to parse unknown properties":                 initFailToParseUnknownPropertiesTC,
	}

	for tcname := range testcases {
		tcinit := testcases[tcname]

		t.Run(tcname, func(t *testing.T) {
			t.Parallel()
			tc := tcinit()

			generic, err := builddef.LoadFromFS(tc.file, tc.lockFile)
			if err != nil {
				t.Fatal(err)
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
	file        string
	lockFile    string
	stage       string
	expected    php.StageDefinition
	expectedErr error
}

func initSuccessfullyResolveDefaultDevStageTC() resolveStageTC {
	file := "testdata/def/without-stages.yml"
	lockFile := "testdata/def/without-stages.lock"

	isFPM := true
	isDev := true
	healthcheckEnabled := true
	phpIni := "docker/app/php.ini"
	fpmConfigFile := "docker/app/fpm.conf"

	return resolveStageTC{
		file:     file,
		lockFile: lockFile,
		stage:    "dev",
		expected: php.StageDefinition{
			Name:          "dev",
			BaseImage:     "docker.io/library/php:7.4-fpm-buster",
			Version:       "7.4.0",
			MajMinVersion: "7.4",
			Infer:         false,
			Dev:           &isDev,
			Stage: php.Stage{
				BaseConfig: builddef.BaseConfig{
					ExternalFiles: []llbutils.ExternalFile{
						{
							URL:         "https://github.com/NiR-/fcgi-client/releases/download/v0.1.0/fcgi-client.phar",
							Compressed:  false,
							Destination: "/usr/local/bin/fcgi-client",
							Mode:        0750,
							Owner:       "1000:1000",
						},
					},
					SystemPackages: map[string]string{},
				},
				FPM: &isFPM,
				Extensions: map[string]string{
					"intl":      "*",
					"pdo_mysql": "*",
					"soap":      "*",
				},
				SourceDirs:   []string{"./src"},
				ExtraScripts: []string{"./public/index.php"},
				StatefulDirs: []string{"./public/uploads"},
				ConfigFiles: php.PHPConfigFiles{
					IniFile:       &phpIni,
					FPMConfigFile: &fpmConfigFile,
				},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					APCU:                  false,
					ClassmapAuthoritative: true,
				},
				Integrations: []string{"blackfire"},
				Healthcheck:  &healthcheckEnabled,
				PostInstall:  []string{"some more commands", "another one"},
			},
		},
	}
}

func initSuccessfullyResolveWorkerStageTC() resolveStageTC {
	isNotFPM := false
	isNotDev := false
	healthcheckDisabled := false
	phpIni := "docker/app/php.prod.ini"
	workerCmd := "bin/worker"

	return resolveStageTC{
		file:     "testdata/def/with-stages.yml",
		lockFile: "",
		stage:    "worker",
		expected: php.StageDefinition{
			Name:          "worker",
			BaseImage:     "docker.io/library/php:7.4-fpm-buster",
			Version:       "7.4.0",
			MajMinVersion: "7.4",
			Infer:         true,
			Dev:           &isNotDev,
			Stage: php.Stage{
				BaseConfig: builddef.BaseConfig{
					ExternalFiles: []llbutils.ExternalFile{
						{
							URL:         "https://blackfire.io/api/v1/releases/probe/php/linux/amd64/72",
							Compressed:  true,
							Pattern:     "blackfire-*.so",
							Destination: "/usr/local/lib/php/extensions/no-debug-non-zts-20190902/blackfire.so",
							Mode:        0644,
						},
					},
					SystemPackages: map[string]string{
						"zlib1g-dev":   "*",
						"libicu-dev":   "*",
						"libxml2-dev":  "*",
						"unzip":        "*",
						"git":          "*",
						"libpcre3-dev": "*",
						"libssl-dev":   "*",
						"openssl":      "*",
					},
				},
				FPM:     &isNotFPM,
				Command: &workerCmd,
				Extensions: map[string]string{
					"intl":      "*",
					"pdo_mysql": "*",
					"soap":      "*",
					"sockets":   "*",
					"apcu":      "*",
					"opcache":   "*",
					"zip":       "*",
				},
				ConfigFiles: php.PHPConfigFiles{
					IniFile: &phpIni,
				},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					APCU:                  true,
					ClassmapAuthoritative: false,
				},
				SourceDirs: []string{"generated/", "worker/", "app/", "src/"},
				ExtraScripts: []string{
					"gencode.php",
					"bin/worker",
					"bin/console",
					"web/app.php",
				},
				Integrations: []string{"symfony", "blackfire"},
				StatefulDirs: []string{
					"public/uploads",
					"data/imports",
				},
				Healthcheck: &healthcheckDisabled,
				PostInstall: []string{
					"php -d display_errors=on bin/console cache:warmup --env=prod",
					"echo some command",
					"echo some other command",
				},
			},
		},
	}
}

func initFailToResolveUnknownStageTC() resolveStageTC {
	file := "testdata/def/without-stages.yml"
	lockFile := "testdata/def/without-stages.lock"

	return resolveStageTC{
		file:        file,
		lockFile:    lockFile,
		stage:       "foo",
		expectedErr: errors.New(`stage "foo" not found`),
	}
}

func initFailToResolveStageWithCyclicDepsTC() resolveStageTC {
	file := "testdata/def/cyclic-stage-deps.yml"
	lockFile := "testdata/def/cyclic-stage-deps.lock"
	return resolveStageTC{
		file:        file,
		lockFile:    lockFile,
		stage:       "dev",
		expectedErr: errors.New(`there's a cyclic dependency between "dev" and itself`),
	}
}

func initSuccessfullyAddSymfonyIntegrationTC() resolveStageTC {
	dev := true
	fpm := true
	healthcheck := false

	return resolveStageTC{
		file:     "testdata/def/symfony-integration.yml",
		lockFile: "",
		stage:    "dev",
		expected: php.StageDefinition{
			Name:          "dev",
			BaseImage:     "docker.io/library/php:7.2-fpm-buster",
			Version:       "7.2",
			MajMinVersion: "7.2",
			Infer:         true,
			Dev:           &dev,
			Stage: php.Stage{
				BaseConfig: builddef.BaseConfig{
					ExternalFiles: []llbutils.ExternalFile{},
					SystemPackages: map[string]string{
						"git":          "*",
						"libpcre3-dev": "*",
						"unzip":        "*",
						"zlib1g-dev":   "*",
					},
				},
				FPM: &fpm,
				Extensions: map[string]string{
					"zip": "*",
				},
				ConfigFiles: php.PHPConfigFiles{},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					ClassmapAuthoritative: true,
				},
				SourceDirs:   []string{"app/", "src/"},
				ExtraScripts: []string{"bin/console", "web/app.php"},
				Integrations: []string{"symfony"},
				StatefulDirs: []string{},
				Healthcheck:  &healthcheck,
				PostInstall: []string{
					"php -d display_errors=on bin/console cache:warmup --env=prod",
				},
			},
		},
	}
}

func initRemoveDefaultExtensionsTC() resolveStageTC {
	dev := true
	fpm := true
	healthcheck := false

	return resolveStageTC{
		file:     "testdata/def/remove-default-exts.yml",
		lockFile: "",
		stage:    "dev",
		expected: php.StageDefinition{
			Name:          "dev",
			BaseImage:     "docker.io/library/php:7.4-fpm-buster",
			Version:       "7.4",
			MajMinVersion: "7.4",
			Infer:         true,
			Dev:           &dev,
			Stage: php.Stage{
				BaseConfig: builddef.BaseConfig{
					ExternalFiles: []llbutils.ExternalFile{},
					SystemPackages: map[string]string{
						"zlib1g-dev":   "*",
						"unzip":        "*",
						"git":          "*",
						"libpcre3-dev": "*",
					},
				},
				FPM: &fpm,
				Extensions: map[string]string{
					"zip": "*",
				},
				ConfigFiles: php.PHPConfigFiles{},
				ComposerDumpFlags: &php.ComposerDumpFlags{
					ClassmapAuthoritative: true,
				},
				SourceDirs:   []string{},
				ExtraScripts: []string{},
				Integrations: []string{},
				StatefulDirs: []string{},
				Healthcheck:  &healthcheck,
				PostInstall:  []string{},
			},
		},
	}
}

func TestResolveStageDefinition(t *testing.T) {
	if *flagTestdata {
		return
	}

	testcases := map[string]func() resolveStageTC{
		"successfully resolve default dev stage": initSuccessfullyResolveDefaultDevStageTC,
		"successfully resolve worker stage":      initSuccessfullyResolveWorkerStageTC,
		"successfully add symfony integration":   initSuccessfullyAddSymfonyIntegrationTC,
		"fail to resolve unknown stage":          initFailToResolveUnknownStageTC,
		"fail to resolve stage with cyclic deps": initFailToResolveStageWithCyclicDepsTC,
		"remove default extensions":              initRemoveDefaultExtensionsTC,
	}

	for tcname := range testcases {
		tcinit := testcases[tcname]

		t.Run(tcname, func(t *testing.T) {
			t.Parallel()
			tc := tcinit()

			generic, err := builddef.LoadFromFS(tc.file, tc.lockFile)
			if err != nil {
				t.Fatal(err)
			}

			def, err := php.NewKind(generic)
			if err != nil {
				t.Fatal(err)
			}

			platformReqsLoader := func(stage *php.StageDefinition) error {
				return php.LoadPlatformReqsFromFS(stage, "")
			}
			stageDef, err := def.ResolveStageDefinition(tc.stage, platformReqsLoader)
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
		"fail when both optimizations are enabled": {
			obj:         php.ComposerDumpFlags{APCU: true, ClassmapAuthoritative: true},
			expectedErr: xerrors.New("you can't use both --apcu and --classmap-authoritative flags. See https://getcomposer.org/doc/articles/autoloader-optimization.md"),
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
