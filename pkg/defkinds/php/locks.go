package php

import (
	"context"
	"strings"

	"github.com/NiR-/notpecl/extindex"
	"github.com/NiR-/zbuild/pkg/builddef"
	"github.com/NiR-/zbuild/pkg/pkgsolver"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"
)

var defaultBaseImages = map[string]struct {
	FPM string
	CLI string
}{
	"7.2": {
		FPM: "docker.io/library/php:7.2-fpm-buster",
		CLI: "docker.io/library/php:7.2-cli-buster",
	},
	"7.3": {
		FPM: "docker.io/library/php:7.3-fpm-buster",
		CLI: "docker.io/library/php:7.3-cli-buster",
	},
	"7.4": {
		FPM: "docker.io/library/php:7.4-fpm-buster",
		CLI: "docker.io/library/php:7.4-cli-buster",
	},
}

// DefinitionLocks defines version locks for system packages and PHP extensions used
// by each stage.
type DefinitionLocks struct {
	BaseImage string                `yaml:"base_image"`
	Stages    map[string]StageLocks `yaml:"stages"`
}

func (l DefinitionLocks) RawLocks() ([]byte, error) {
	lockdata, err := yaml.Marshal(l)
	if err != nil {
		return lockdata, xerrors.Errorf("could not marshal php locks: %w", err)
	}
	return lockdata, nil
}

// StageLocks represents the version locks for a single stage.
type StageLocks struct {
	SystemPackages map[string]string `yaml:"system_packages"`
	Extensions     map[string]string `yaml:"extensions"`
}

func (h *PHPHandler) UpdateLocks(
	ctx context.Context,
	pkgSolver pkgsolver.PackageSolver,
	genericDef *builddef.BuildDef,
) (builddef.Locks, error) {
	def, err := NewKind(genericDef)
	if err != nil {
		return nil, err
	}

	// @TODO: resolve sha256 of the base image and lock it
	def.Locks.BaseImage = def.BaseImage

	osrelease, err := builddef.ResolveImageOS(ctx, h.solver, def.Locks.BaseImage)
	if err != nil {
		return nil, xerrors.Errorf("could not resolve OS details from base image: %w", err)
	}
	if osrelease.Name != "debian" {
		return nil, xerrors.Errorf("unsupported OS %q: only debian-based base images are supported", osrelease.Name)
	}

	solverCfg, err := pkgsolver.GuessSolverConfig(osrelease, "amd64")
	if err != nil {
		return nil, xerrors.Errorf("could not update stage locks: %w", err)
	}
	err = pkgSolver.Configure(solverCfg)
	if err != nil {
		return nil, xerrors.Errorf("could not update stage locks: %w", err)
	}

	stagesLocks, err := h.updateStagesLocks(ctx, pkgSolver, def)
	def.Locks.Stages = stagesLocks
	return def.Locks, err
}

func (h *PHPHandler) updateStagesLocks(
	ctx context.Context,
	pkgSolver pkgsolver.PackageSolver,
	def Definition,
) (map[string]StageLocks, error) {
	locks := map[string]StageLocks{}
	composerLockLoader := func(stageDef *StageDefinition) error {
		return LoadComposerLock(ctx, h.solver, stageDef)
	}

	for name := range def.Stages {
		stage, err := def.ResolveStageDefinition(name, composerLockLoader)
		if err != nil {
			return nil, xerrors.Errorf("could not resolve stage %q: %w", name, err)
		}

		stageLocks := StageLocks{}
		stageLocks.SystemPackages, err = pkgSolver.ResolveVersions(stage.SystemPackages)
		if err != nil {
			return nil, xerrors.Errorf("could not resolve systems package versions: %w", err)
		}

		stageLocks.Extensions, err = h.lockExtensions(stage.Extensions)
		if err != nil {
			return nil, xerrors.Errorf("could not resolve php extension versions: %w", err)
		}

		locks[name] = stageLocks
	}

	return locks, nil
}

func (h *PHPHandler) lockExtensions(extensions map[string]string) (map[string]string, error) {
	resolved := map[string]string{}
	ctx := context.Background()

	for extName, constraint := range extensions {
		if isCoreExtension(extName) {
			resolved[extName] = constraint
			continue
		}

		segments := strings.SplitN(constraint, "@", 2)
		stability := extindex.Stable
		if len(segments) == 2 {
			stability = extindex.StabilityFromString(segments[1])
		}

		extVer, err := h.NotPecl.ResolveConstraint(ctx, extName, segments[0], stability)
		if err != nil {
			return resolved, err
		}

		resolved[extName] = extVer
	}

	return resolved, nil
}