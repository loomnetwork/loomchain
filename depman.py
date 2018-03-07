#!/usr/bin/env python3

import os
import subprocess

import yaml

gopaths = list(filter(None, os.environ.get('GOPATH', "").split(":")))


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
    subprocess.run(["go", "get", "-u", "-d", name])
    path = os.path.join(gopaths[0], "src", name)
    subprocess.run(["git", "checkout", ver], cwd=path)


def glide_vendor(lock_path):
    for dep in yaml.load(open(lock_path))['imports']:
        print("vendoring " + dep['name'])
        install_dep(dep['name'], dep['version'])


print("Vendoring cosmos glide deps")
glide_vendor(gopaths[0] + "/src/github.com/cosmos/cosmos-sdk/glide.lock")

deps = [d for p in gopaths for d in list_dirs(p + "/src")]

for dep in deps:
    try:
        hide_git(dep)
        print("Fixed .git for " + dep)
    except FileNotFoundError:
        pass
