%global project containerd/runwasi
%global repo runwasi
%global gitcommit eb9967aae75bfc8ddf810590810f6962a8dc7c37
%global shortcommit eb9967aa
%global shim_version 0.6.0
%global rpmver %{shim_version}

# The wasmtime shim pulls in ring transitively via wasmtime-wasi-http/rustls.
# This is guest-facing crypto (WASI HTTP for wasm modules), not system TLS.
# Disable the FIPS crypto provider check for this package.
%undefine cross_check_fips

Name: %{_cross_os}runwasi-shims
Version: %{rpmver}
Release: 1%{?dist}
Summary: Containerd shims for running WebAssembly workloads
License: Apache-2.0
URL: https://github.com/%{project}
Source0: https://github.com/%{project}/archive/%{gitcommit}.tar.gz#/%{repo}-%{shortcommit}.tar.gz
Source1: %{repo}-%{shortcommit}-vendor.tar.gz

BuildRequires: %{_cross_os}glibc-devel
BuildRequires: %{_cross_os}libseccomp-devel

%description
%{summary}.

%prep
%setup -n %{repo}-%{gitcommit}
tar xzf %{SOURCE1}

# The vendored libseccomp crate uses pkg-config to detect libseccomp >= 2.5.0
# and conditionally compiles the notify module (needed by libcontainer).
# The cross-compilation sysroot lacks .pc files so detection fails.
# The SDK ships libseccomp 2.5.5, so force-enable the cfg flag.
sed -i '/^fn main/a\    println!("cargo:rustc-cfg=libseccomp_v2_5");' vendor/libseccomp/build.rs
# Clear vendor file checksums so cargo accepts the patched file.
sed -i 's/"files":{[^}]*}/"files":{}/' vendor/libseccomp/.cargo-checksum.json

%cargo_prep

# Point cargo at the vendored dependencies.
cat >> %{_builddir}/.cargo/config.toml << 'EOF'

[source.crates-io]
replace-with = "vendored-sources"

[source."git+https://github.com/bytecodealliance/wamr-rust-sdk?tag=v1.1.0"]
git = "https://github.com/bytecodealliance/wamr-rust-sdk"
tag = "v1.1.0"
replace-with = "vendored-sources"

[source.vendored-sources]
directory = "%{_builddir}/%{repo}-%{gitcommit}/vendor"
EOF

%build
%cargo_build --manifest-path %{_builddir}/%{repo}-%{gitcommit}/Cargo.toml \
    -p containerd-shim-wasmtime

%install
install -d %{buildroot}%{_cross_bindir}
# The binary may be in the release dir or deps/ depending on cargo behavior
if [ -f %{__cargo_outdir}/containerd-shim-wasmtime-v1 ]; then
    install -p -m 0755 %{__cargo_outdir}/containerd-shim-wasmtime-v1 %{buildroot}%{_cross_bindir}/
else
    find %{__cargo_outdir} -name 'containerd_shim_wasmtime_v1*' -not -name '*.d' -not -name '*.rmeta' -type f \
        -exec install -p -m 0755 {} %{buildroot}%{_cross_bindir}/containerd-shim-wasmtime-v1 \;
fi

%files
%license LICENSE
%{_cross_attribution_file}
%{_cross_bindir}/containerd-shim-wasmtime-v1

%changelog
