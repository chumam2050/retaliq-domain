#!/usr/bin/env bash
# helper to build the .deb package with versioning based on git tags
set -euo pipefail

dir=$(dirname "$0")
# work directory for building; change to debian dir only for relative paths
cd "$dir" || exit 1

# derive project root directory (parent of debian)
root=$(cd .. && pwd)

# derive version from git tags, fall back to 0.1.0
version=$(git describe --tags --abbrev=0 2>/dev/null || echo "0.1.0")
# strip leading 'v' if present
version=${version#v}

# update changelog with version if necessary (requires dch/devscripts)
changelog="$root/debian/changelog"
if [ ! -f "$changelog" ]; then
    echo "warning: changelog not found at $changelog" >&2
else
    if ! grep -q "^retaliq-domain (${version})" "$changelog"; then
    if command -v dch >/dev/null 2>&1; then
        # ensure DEBEMAIL is set to avoid interactive prompt
        if [ -z "${DEBEMAIL:-}" ]; then
            DEBEMAIL=$(git config --get user.email || true)
            export DEBEMAIL
        fi
        # disable directory-name check since we run from debian/
        dch --check-dirname-level=0 -v "${version}-1" "Automatic release"
        else
            echo "warning: dch not found, please update debian/changelog manually" >&2
        fi
    fi
fi

# build package
# require debhelper etc
if ! command -v dpkg-buildpackage >/dev/null; then
    echo "dpkg-buildpackage not found; install build-essential debhelper" >&2
    exit 1
fi

# build in parent directory
cd ..
# disable dwz invocation which sometimes fails on Go binaries
export DH_NO_DWZ=1
# -b: binary-only build, avoids debhelper snapshotting the full source tree
# (which would include read-only Go module cache files)
dpkg-buildpackage -us -uc -b

# move deb to dist if exists
pkgroot=$(dirname "$0")/..
mkdir -p "$pkgroot/dist"
# dpkg-buildpackage places the .deb and other metadata one level above project root
debfile=$(ls "$root/../retaliq-domain_"*.deb 2>/dev/null || true)
if [ -n "$debfile" ]; then
    mv "$debfile" "$pkgroot/dist/"
    echo "package built and moved to dist/$(basename "$debfile")"
else
    echo "package built, see parent directory for .deb files"
fi

# remove other build artifacts from parent
rm -f "$root/../"retaliq-domain_*.{buildinfo,changes,dsc,tar.gz} || true

# cleanup leftover artefacts in project
rm -f "$root/retaliq-domain"    # remove built binary
rm -rf "$root/debian/retaliq-domain"    # remove temporary packaging dir
# remove various debhelper-generated files to mimic initial clean state
rm -f "$root/debian/debhelper-build-stamp" "$root/debian/files" \
       "$root/debian/retaliq-domain.postrm.debhelper" "$root/debian/retaliq-domain.substvars"
rm -rf "$root/debian/.debhelper"
