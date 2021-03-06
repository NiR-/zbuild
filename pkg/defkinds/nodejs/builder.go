package nodejs

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/NiR-/zbuild/pkg/builddef"
	"github.com/NiR-/zbuild/pkg/image"
	"github.com/NiR-/zbuild/pkg/llbutils"
	"github.com/NiR-/zbuild/pkg/registry"
	"github.com/NiR-/zbuild/pkg/statesolver"
	"github.com/moby/buildkit/client/llb"
	"golang.org/x/xerrors"
)

var SharedKeys = struct {
	BuildContext string
	ConfigFiles  string
	PackageFiles string
}{
	BuildContext: "build-context",
	ConfigFiles:  "config-files",
	PackageFiles: "package-files",
}

const WorkingDir = "/app"

func init() {
	RegisterKind(registry.Registry)
}

func RegisterKind(reg *registry.KindRegistry) {
	reg.Register("nodejs", &NodeJSHandler{}, true)
}

type NodeJSHandler struct {
	solver statesolver.StateSolver
}

func (h *NodeJSHandler) WithSolver(solver statesolver.StateSolver) {
	h.solver = solver
}

func (h *NodeJSHandler) DebugConfig(
	buildOpts builddef.BuildOpts,
) (interface{}, error) {
	stageDef, err := h.loadDefs(buildOpts)
	if err != nil {
		return nil, err
	}

	// This property would pollute the dumped config
	stageDef.DefLocks.Stages = map[string]StageLocks{}

	return stageDef, nil
}

func (h *NodeJSHandler) Build(
	ctx context.Context,
	buildOpts builddef.BuildOpts,
) (llb.State, *image.Image, error) {
	var state llb.State
	var img *image.Image

	stageDef, err := h.loadDefs(buildOpts)
	if err != nil {
		return state, img, err
	}

	stageDef.PackageManager, err = h.determinePackageManager(ctx, stageDef, buildOpts)
	if err != nil {
		return state, img, err
	}

	state, img, err = h.buildNodeJS(ctx, stageDef, buildOpts)
	if err != nil {
		err = xerrors.Errorf("could not build nodejs stage: %w", err)
		return state, img, err
	}

	return state, img, nil
}

func (h *NodeJSHandler) buildNodeJS(
	ctx context.Context,
	stageDef StageDefinition,
	buildOpts builddef.BuildOpts,
) (llb.State, *image.Image, error) {
	state := llbutils.ImageSource(stageDef.DefLocks.BaseImage, true)
	baseImg, err := image.LoadMeta(ctx, stageDef.DefLocks.BaseImage)
	if err != nil {
		return state, nil, xerrors.Errorf("failed to load %q metadata: %w", stageDef.DefLocks.BaseImage, err)
	}

	img := image.CloneMeta(baseImg)
	img.Config.Labels[builddef.ZbuildLabel] = "true"

	pkgManager := llbutils.APT
	if stageDef.DefLocks.OSRelease.Name == "alpine" {
		pkgManager = llbutils.APK
	}

	if buildOpts.WithCacheMounts && len(stageDef.StageLocks.SystemPackages) > 0 {
		state = llbutils.SetupSystemPackagesCache(state, pkgManager)
	}

	state, err = llbutils.InstallSystemPackages(state, pkgManager,
		stageDef.StageLocks.SystemPackages,
		llbutils.NewCachingStrategyFromBuildOpts(buildOpts))
	if err != nil {
		return state, img, xerrors.Errorf("failed to add \"install system pacakges\" steps: %w", err)
	}

	state = llbutils.CopyExternalFiles(state, stageDef.ExternalFiles)
	state = llbutils.Mkdir(state, "1000:1000",
		append([]string{WorkingDir}, stageDef.StatefulDirs...)...)
	state = state.User("1000")
	state = state.Dir(WorkingDir)

	state = h.globalPackagesInstall(state, stageDef, buildOpts)
	if !*stageDef.Dev {
		state = h.depsInstall(stageDef, state, buildOpts)
		state, err = h.copyConfigFiles(stageDef, state, buildOpts)
		if err != nil {
			return state, img, err
		}

		state = h.copySources(stageDef, state, buildOpts)
		state = h.build(stageDef, state, buildOpts)
	}

	setImageMetadata(stageDef, state, img)

	return state, img, nil
}

func setImageMetadata(stageDef StageDefinition, state llb.State, img *image.Image) {
	for _, dir := range stageDef.StatefulDirs {
		fullpath := dir
		if !path.IsAbs(fullpath) {
			fullpath = path.Join(WorkingDir, dir)
		}

		img.Config.Volumes[fullpath] = struct{}{}
	}

	if stageDef.Healthcheck != nil {
		img.Config.Healthcheck = stageDef.Healthcheck.ToImageConfig()
	}

	nodeEnv := "development"
	if !*stageDef.Dev {
		nodeEnv = "production"
	}

	img.Config.User = "1000"
	img.Config.WorkingDir = WorkingDir
	img.Config.Env = []string{
		"PATH=" + getEnv(state, "PATH"),
		"NODE_ENV=" + nodeEnv,
	}
	now := time.Now()
	img.Created = &now

	if stageDef.Command != nil {
		img.Config.Cmd = *stageDef.Command
	}
}

func getEnv(src llb.State, name string) string {
	val, _ := src.GetEnv(name)
	return val
}

func (h *NodeJSHandler) globalPackagesInstall(
	state llb.State,
	stageDef StageDefinition,
	buildOpts builddef.BuildOpts,
) llb.State {
	if stageDef.GlobalPackages.Len() == 0 {
		return state
	}

	pkgs := make([]string, 0, stageDef.GlobalPackages.Len())
	for pkg, constraint := range stageDef.GlobalPackages.Map() {
		if constraint != "" && constraint != "*" {
			pkg += "@" + constraint
		}
		pkgs = append(pkgs, pkg)
	}

	runOpts := []llb.RunOption{llb.User("1000")}
	if stageDef.PackageManager == pkgManagerYarn {
		runOpts = append(runOpts,
			llbutils.Shell("yarn global add "+strings.Join(pkgs, " ")),
			llb.WithCustomName("Run yarn global add"))
	} else {
		runOpts = append(runOpts,
			llbutils.Shell("npm install -g "+strings.Join(pkgs, " ")),
			llb.WithCustomName("Run npm install"))
	}

	if buildOpts.IgnoreLayerCache {
		runOpts = append(runOpts, llb.IgnoreCache)
	}

	runOpts, state = cacheMountOptForJSDeps(
		runOpts, state, buildOpts, stageDef.PackageManager)

	return state.Run(runOpts...).Root()
}

func cacheMountOptForJSDeps(
	runOpts []llb.RunOption,
	state llb.State,
	buildOpts builddef.BuildOpts,
	pkgMgr string,
) ([]llb.RunOption, llb.State) {
	if !buildOpts.WithCacheMounts {
		return runOpts, state
	}

	if pkgMgr == pkgManagerYarn {
		runOpts = append(runOpts, llbutils.CacheMountOpt(
			"/home/node/.cache/yarn", buildOpts.CacheIDNamespace, "1000"))
		return runOpts, state
	}

	state = state.AddEnv("NPM_CONFIG_PREFIX", "/home/node/.npm")
	runOpts = append(runOpts, llbutils.CacheMountOpt(
		"/home/node/.npm/", buildOpts.CacheIDNamespace, "1000"))

	return runOpts, state
}

func (h *NodeJSHandler) determinePackageManager(
	ctx context.Context,
	stageDef StageDefinition,
	buildOpts builddef.BuildOpts,
) (string, error) {
	srcContext := resolveSourceContext(stageDef, buildOpts)
	lockpath := prefixContextPath(srcContext, "package-lock.json")
	packageLock, err := h.solver.FileExists(ctx, lockpath, srcContext)
	if err != nil {
		return "", xerrors.Errorf("could not determine which package manager should be used (from %s context): %w", srcContext.Type, err)
	}

	if packageLock {
		return pkgManagerNpm, nil
	}
	return pkgManagerYarn, nil
}

func (h *NodeJSHandler) depsInstall(
	stageDef StageDefinition,
	state llb.State,
	buildOpts builddef.BuildOpts,
) llb.State {
	srcContext := resolveSourceContext(stageDef, buildOpts)

	var installCmd, installLabel string
	include := make([]string, 2)
	include[0] = prefixContextPath(srcContext, "package.json")

	if stageDef.PackageManager == pkgManagerYarn {
		include[1] = prefixContextPath(srcContext, "yarn.lock")
		installCmd = "yarn install --frozen-lockfile"
		installLabel = "Run yarn install"
	} else {
		include[1] = prefixContextPath(srcContext, "package-lock.json")
		installCmd = "npm ci"
		installLabel = "Run npm install"
	}

	srcLabel := fmt.Sprintf("load %s from build context",
		strings.Join(include, " and "))
	srcState := llbutils.FromContext(srcContext,
		llb.IncludePatterns(include),
		llb.LocalUniqueID(buildOpts.LocalUniqueID),
		llb.SessionID(buildOpts.SessionID),
		llb.SharedKeyHint(SharedKeys.PackageFiles),
		llb.WithCustomName(srcLabel))

	state = llbutils.Copy(
		srcState, include[0], state, "/app/", "1000:1000", buildOpts.IgnoreLayerCache)
	state = llbutils.Copy(
		srcState, include[1], state, "/app/", "1000:1000", buildOpts.IgnoreLayerCache)

	runOpts := []llb.RunOption{
		llbutils.Shell(installCmd),
		llb.Dir(state.GetDir()),
		llb.User("1000"),
		llb.WithCustomName(installLabel)}

	if buildOpts.IgnoreLayerCache {
		runOpts = append(runOpts, llb.IgnoreCache)
	}

	runOpts, state = cacheMountOptForJSDeps(
		runOpts, state, buildOpts, stageDef.PackageManager)

	return state.Run(runOpts...).Root()
}

func (h *NodeJSHandler) copySources(
	stageDef StageDefinition,
	state llb.State,
	buildOpts builddef.BuildOpts,
) llb.State {
	sourceContext := resolveSourceContext(stageDef, buildOpts)
	srcState := llbutils.FromContext(sourceContext,
		llb.IncludePatterns(includePatterns(sourceContext, stageDef)),
		llb.ExcludePatterns(excludePatterns(sourceContext, stageDef)),
		llb.LocalUniqueID(buildOpts.LocalUniqueID),
		llb.SessionID(buildOpts.SessionID),
		llb.SharedKeyHint(SharedKeys.BuildContext),
		llb.WithCustomName("load build context"))

	if sourceContext.Type == builddef.ContextTypeLocal {
		srcPath := prefixContextPath(sourceContext, "/")
		return llbutils.Copy(
			srcState, srcPath, state, WorkingDir, "1000:1000", buildOpts.IgnoreLayerCache)
	}

	// Despite the IncludePatterns() above, the source state might also
	// contain files that were not including if the conext is non-local.
	// As such, we can't just copy the whole source state to the dest state
	// in such case.
	for _, srcfile := range stageDef.Sources {
		srcPath := prefixContextPath(sourceContext, srcfile)
		destPath := path.Join(WorkingDir, srcfile)
		state = llbutils.Copy(
			srcState, srcPath, state, destPath, "1000:1000", buildOpts.IgnoreLayerCache)
	}

	return state
}

func (h *NodeJSHandler) copyConfigFiles(
	stageDef StageDefinition,
	state llb.State,
	buildOpts builddef.BuildOpts,
) (llb.State, error) {
	if len(stageDef.ConfigFiles) == 0 {
		return state, nil
	}

	srcContext := buildOpts.BuildContext
	srcPrefix := srcContext.Subdir()
	include := stageDef.ConfigFiles.SourcePaths(srcPrefix)
	srcState := llbutils.FromContext(srcContext,
		llb.IncludePatterns(include),
		llb.LocalUniqueID(buildOpts.LocalUniqueID),
		llb.SessionID(buildOpts.SessionID),
		llb.SharedKeyHint(SharedKeys.BuildContext),
		llb.WithCustomName("load config files"))

	interpolated, err := stageDef.ConfigFiles.Interpolate(
		srcPrefix, WorkingDir, map[string]string{})
	if err != nil {
		return state, err
	}

	// Despite the IncludePatterns() above, the source state might also
	// contain files that were not including, for instance if the conext is
	// non-local. However, including precise patterns help buildkit determine
	// if the cache is fresh (when using a local context). As such, we can't
	// just copy the whole source state to the dest state.
	state = llbutils.CopyAll(
		srcState, state, interpolated, "1000:1000", buildOpts.IgnoreLayerCache)

	return state, nil
}

func resolveSourceContext(
	stageDef StageDefinition,
	buildOpts builddef.BuildOpts,
) *builddef.Context {
	if stageDef.DefLocks.SourceContext != nil {
		return stageDef.DefLocks.SourceContext
	}

	return buildOpts.BuildContext
}

func excludePatterns(srcContext *builddef.Context, stageDef StageDefinition) []string {
	excludes := []string{}
	// Explicitly exclude stateful dirs to ensure they aren't included when
	// they're in one of Sources
	for _, dir := range stageDef.StatefulDirs {
		dirpath := prefixContextPath(srcContext, dir)
		excludes = append(excludes, dirpath)
	}
	return excludes
}

func includePatterns(srcContext *builddef.Context, stageDef StageDefinition) []string {
	includes := []string{}
	for _, srcpath := range stageDef.Sources {
		fullpath := prefixContextPath(srcContext, srcpath)
		includes = append(includes, fullpath)
	}
	return includes
}

func prefixContextPath(srcContext *builddef.Context, p string) string {
	if srcContext.IsGitContext() && srcContext.Path != "" {
		return path.Join("/", srcContext.Path, p)
	}

	return p
}

func (h *NodeJSHandler) build(
	stageDef StageDefinition,
	state llb.State,
	buildOpts builddef.BuildOpts,
) llb.State {
	if stageDef.BuildCommand == nil {
		return state
	}

	envPath := strings.Join([]string{
		"/home/node/.npm/bin/",
		"/home/node/.yarn/bin/",
		getEnv(state, "PATH"),
	}, ":")
	runOpts := []llb.RunOption{
		llbutils.Shell(*stageDef.BuildCommand),
		llb.Dir(state.GetDir()),
		llb.AddEnv("NODE_ENV", "production"),
		llb.AddEnv("PATH", envPath),
		llb.WithCustomName("Build")}

	if buildOpts.IgnoreLayerCache {
		runOpts = append(runOpts, llb.IgnoreCache)
	}

	return state.Run(runOpts...).Root()
}
