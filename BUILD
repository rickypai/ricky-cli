load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")
load("@bazel_gazelle//:def.bzl", "gazelle")
load("@com_github_bazelbuild_buildtools//buildifier:def.bzl", "buildifier")

gazelle(
    name = "gazelle",
    external = "vendored",
    prefix = "github.com/rickypai/ricky-cli",
)

buildifier(
    name = "buildifier",
)

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/rickypai/ricky-cli",
    visibility = ["//visibility:private"],
    deps = [
        "//vendor/github.com/google/go-github/github:go_default_library",
        "//vendor/golang.org/x/oauth2:go_default_library",
    ],
)

go_binary(
    name = "ricky-cli",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)
