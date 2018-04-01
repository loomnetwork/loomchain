#!/usr/bin/env python3

import os
import subprocess

import yaml
import toml

gopaths = list(filter(None, os.environ.get('GOPATH', "").split(":")))
vendor_dir = gopaths[0]


def hide_git(path):
    os.rename(path + "/.git", path + "/.checkout_git")


def show_git(path):
    os.rename(path + "/.checkout_git", path + "/.git")


def list_dirs(path, depth=3):
    dirs = [os.path.join(path, d) for d in os.listdir(path)]
    dirs = list(filter(os.path.isdir, dirs))
    if depth == 1:
        return dirs

    return dirs + [d2 for d in dirs for d2 in list_dirs(d, depth - 1)]


def install_dep(name, ver):
    path = os.path.join(vendor_dir, "src", name)

    # Put dep back into a good state for go get
    try:
        show_git(path)
    except FileNotFoundError:
        pass
    subprocess.run(["git", "checkout", "master"], cwd=path)

    subprocess.run(["go", "get", "-u", "-d", "-v", name])
    subprocess.run(["git", "checkout", ver], cwd=path)


def glide_vendor(lock_path):
    for dep in yaml.load(open(lock_path))['imports']:
        print("vendoring " + dep['name'])
        install_dep(dep['name'], dep['version'])


def gopkg_vendor(lock_path):
    for dep in toml.load(open(lock_path))['projects']:
        print("vendoring " + dep['name'])
        install_dep(dep['name'], dep['revision'])


def list_deps():
    return [d for p in gopaths for d in list_dirs(p + "/src")]


print("Vendoring cosmos deps")

gopkg_vendor(vendor_dir + "/src/github.com/cosmos/cosmos-sdk/Gopkg.lock")

for path in list_deps():
    try:
        hide_git(path)
    except FileNotFoundError:
        pass
