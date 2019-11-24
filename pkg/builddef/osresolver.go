package builddef

import (
	"context"
	"strconv"
	"strings"

	"github.com/NiR-/webdf/pkg/filefetch"
)

// OSRelease represents the data about the base image OS needed by webdf. These
// data typically come from /etc/os-release file.
type OSRelease struct {
	Name        string
	VersionName string
	VersionID   int
}

func ParseOSRelease(file []byte) (OSRelease, error) {
	var res OSRelease

	lines := strings.Split(string(file), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)

		switch parts[0] {
		case "ID":
			res.Name = parts[1]
		case "VERSION_CODENAME":
			res.VersionName = parts[1]
		case "VERSION_ID":
			val, err := strconv.Atoi(strings.Trim(parts[1], `"`))
			if err != nil {
				return res, err
			}
			res.VersionID = val
		}
	}

	return res, nil
}

func ResolveImageOS(
	ctx context.Context,
	fetcher filefetch.FileFetcher,
	imageRef string,
) (OSRelease, error) {
	var res OSRelease

	raw, err := fetcher.FetchFile(ctx, imageRef, "/etc/os-release")
	if err != nil {
		return res, err
	}

	return ParseOSRelease(raw)
}
