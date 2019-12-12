# defining macros needed by SELinux
%global selinuxtype targeted
%global modulename virt_launcher
%global base_module kubevirt_container

Name: kubevirt-selinux
Version: 0.1
Release: 1%{?dist}
License: Apache V2
URL: https://github.com/kubevirt/kubevirt
Summary: SELinux policy modules for KubeVirt
Source0: %{modulename}.tar.gz
Requires: (%{name} if selinux-policy-%{selinuxtype})
BuildRequires: selinux-policy
BuildRequires: selinux-policy-devel
BuildArch: noarch
%{?selinux_requires}
%description
SELinux policy modules for KubeVirt.

#%prep
#%setup -q

%pre
%selinux_relabel_pre -s %{selinuxtype}

%install
# install policy modules
install -d %{buildroot}%{_datadir}/selinux/packages/
install -m 0644 %{base_module}.cil %{buildroot}%{_datadir}/selinux/packages/
install -m 0644 %{modulename}.cil %{buildroot}%{_datadir}/selinux/packages/


%check

%post
%selinux_modules_install -s %{selinuxtype} %{_datadir}/selinux/packages/%{base_module}.cil
%selinux_modules_install -s %{selinuxtype} %{_datadir}/selinux/packages/%{modulename}.cil

%postun
if [ $1 -eq 0 ]; then
    %selinux_modules_uninstall -s %{selinuxtype} %{modulename}
    %selinux_modules_uninstall -s %{selinuxtype} %{base_module}
fi

%posttrans
%selinux_relabel_post -s %{selinuxtype}

%files
%{_datadir}/selinux/packages/%{base_module}.cil
%{_datadir}/selinux/packages/%{modulename}.cil
%ghost %{_sharedstatedir}/selinux/%{selinuxtype}/active/modules/200/%{base_module}
%ghost %{_sharedstatedir}/selinux/%{selinuxtype}/active/modules/200/%{modulename}
%license LICENSE

%changelog
* Tue Nov 26 2019 The KubeVirt Project <kubevirt-dev@googlegroups.com> - 0.1
- Initial Specfile
