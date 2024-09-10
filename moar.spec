%define name moar
%define version 1.27.0
%define release 1%{?dist}

Summary:  Moar is a pager. It's designed to just do the right thing without any configuration.
Name:     %{name}
Version:  %{version}
Release:  %{release}
License:  MIT License
URL:      https://github.com/walles/moar
Source0:  https://github.com/walles/moar/archive/refs/tags/v%{version}.tar.gz

%define debug_package %{nil}

BuildRequires: curl
BuildRequires: gcc
BuildRequires: make
BuildRequires: gzip
BuildRequires: golang
BuildRequires: upx
BuildRequires: git

%description
Moar is a pager. It reads and displays UTF-8 encoded text from files or pipelines.

%prep
%setup -q

%build
GO111MODULE=on CGO_ENABLED=0 go build -v -trimpath -modcacherw -tags netgo -ldflags="-s -w -X main.versionString=%{version}" -o "%{name}"
strip --strip-all %{name}
upx %{name}
gzip %{name}.1

%install
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_mandir}/man1
install -m 755 %{name} %{buildroot}%{_bindir}
install -m 644 %{name}.1.gz %{buildroot}%{_mandir}/man1

%files
%doc README.md
%license LICENSE
%{_bindir}/%{name}
%{_mandir}/man1/%{name}.1.gz

%changelog
* Tue Sep 9 2024 - Danie de Jager - 1.27.0-1
* Tue Aug 27 2024 - Danie de Jager - 1.26.0-1
* Mon Aug 12 2024 - Danie de Jager - 1.25.4-1
* Tue Aug 6 2024 - Danie de Jager - 1.25.2-1
* Tue Jul 16 2024 - Danie de Jager - 1.25.1-1
* Sun Jul 14 2024 - Danie de Jager - 1.25.0-1
* Tue Jul 9 2024 - Danie de Jager - 1.24.6-1
* Tue May 21 2024 - Danie de Jager - 1.23.15-1
* Fri Mar 1 2024 - Danie de Jager - 1.23.6-1
- Initial RPM build
