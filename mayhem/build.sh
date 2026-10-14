#!/usr/bin/env bash
#
# go-redis/mayhem/build.sh — mining branch: build the proto.Writer round-trip harness
# (mayhem/harness/writer, `func Fuzz([]byte) int`) as a sanitized libFuzzer binary with
# go114-fuzz-build, linked with clang++ against $LIB_FUZZING_ENGINE.
#
# We produce:
#   /mayhem/fuzz_writer — proto.Writer.WriteArg encode -> proto.Reader decode, with the oracle that a nil
#                         pointer to a numeric/bool type encodes as the zero value (ASan+libFuzzer)
#
# AIR-GAPPED CONTRACT (SPEC §6.5): the PATCH tier re-runs THIS script OFFLINE.
#   - The Dockerfile ENV pins GOROOT/GOPATH/GOMODCACHE under /opt/toolchains (absolute,
#     $HOME-independent) so the module cache the FIRST (online) build populates is found
#     again by the SAME GOMODCACHE path when this script re-runs offline.
#   - We set GOPROXY to the in-image file proxy FIRST, then network as fallback; the
#     offline re-run resolves entirely from the cache.  GOPROXY=off is NOT enough — it
#     blocks reading the version list from the cache, which `go get` needs.
set -euo pipefail

# clang rejects SOURCE_DATE_EPOCH='' — must be unset or a valid integer.
[ -n "${SOURCE_DATE_EPOCH:-}" ] || unset SOURCE_DATE_EPOCH

: "${CC:=clang}" ; : "${CXX:=clang++}" ; : "${LIB_FUZZING_ENGINE:=-fsanitize=fuzzer}"
# OSS-Fuzz Go path is ASAN-only (project.yaml sanitizers: [address]); UBSan is not
# part of the Go libFuzzer link.  Keep ASan regardless of the base default.
: "${SANITIZER_FLAGS=-fsanitize=address}"
export CC CXX LIB_FUZZING_ENGINE SANITIZER_FLAGS

# DWARF < 4 required by §6.2 item 10 so the verify-repo DWARF gate passes.
# Pass $GO_DEBUG_FLAGS on the final clang++ link of the .a → the first C shim
# compilation unit carries DWARF3.
: "${GO_DEBUG_FLAGS:=-g -gdwarf-3}"
export CGO_CFLAGS="${CGO_CFLAGS:+$CGO_CFLAGS }$GO_DEBUG_FLAGS"
export CGO_CXXFLAGS="${CGO_CXXFLAGS:+$CGO_CXXFLAGS }$GO_DEBUG_FLAGS"

# Resolve modules offline-first from the in-image cache; network only as fallback.
# $(go env GOMODCACHE) reads the pinned ENV, so it is correct under ANY $HOME.
export GOFLAGS="${GOFLAGS:--mod=mod}"
export GOPROXY="${GOPROXY:-file://$(go env GOMODCACHE)/cache/download,https://proxy.golang.org,direct}"
export GOTOOLCHAIN="${GOTOOLCHAIN:-local}"

cd "$SRC"
go version

# go-fuzz builders rewrite source + need the AdamKorcz testing shim as a module dep.
# Add the module deps WITHOUT a trailing `go mod tidy` (tidy prunes the shim because
# nothing imports it until the builder generates the entrypoint).  Order matters:
# tidy first, then `go get` the shim.
go mod tidy 2>&1 | tail -2 || true
go get github.com/AdamKorcz/go-118-fuzz-build/testing@latest 2>&1 | tail -2 || true

mkdir -p "$SRC/mayhem-build"

# Init-ordering shim (go_runtime_ready.c), linked FIRST (#1644).
SHIM_O="$SRC/mayhem-build/go_runtime_ready.o"
$CC $GO_DEBUG_FLAGS -c "$SRC/mayhem/go_runtime_ready.c" -o "$SHIM_O"

echo "=== building writer (mayhem/harness/writer.Fuzz, go114-fuzz-build) ==="
go-fuzz -func Fuzz -o "$SRC/mayhem-build/writer.a" \
    github.com/redis/go-redis/v9/mayhem/harness/writer
# Link with DWARF3 flags so the ELF carries DWARF < 4 symbols (§6.2 item 10).
$CXX $SANITIZER_FLAGS $LIB_FUZZING_ENGINE $GO_DEBUG_FLAGS "$SHIM_O" \
    "$SRC/mayhem-build/writer.a" -o /mayhem/fuzz_writer
# No `grep -q`: under pipefail nm would die of SIGPIPE.
nm /mayhem/fuzz_writer | grep ' T LLVMFuzzerInitialize$' >/dev/null \
  || { echo "build.sh: LLVMFuzzerInitialize init-ordering shim missing from /mayhem/fuzz_writer" >&2; exit 1; }
echo "built /mayhem/fuzz_writer"

echo "build.sh complete:"
ls -la /mayhem/fuzz_writer 2>&1 || true
