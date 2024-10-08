load("@bazel_skylib//lib:shell.bzl", "shell")
load("//:go_protogen.bzl", "gen_copy_files_script")

def go_protogen_impl(ctx, **kwargs):
    return gen_copy_files_script(ctx, ctx.attr.dep[OutputGroupInfo].go_generated_srcs.to_list())

_go_protogen = rule(
    implementation = go_protogen_impl,
    attrs = {
        "dir": attr.string(),
        "dep": attr.label(),
        "_template": attr.label(
            default = "//:copy_into_workspace.sh",
            allow_single_file = True,
        ),
        # It is not used, just used for versioning since this is experimental
        "version": attr.string(),
    },
)

def go_protogen(name, visibility = None, **kwargs):
    if not "dir" in kwargs:
        dir = native.package_name()
        kwargs["dir"] = dir

    gen_rule_name = "%s_copy_gen" % name
    _go_protogen(name = gen_rule_name, **kwargs)

    native.sh_binary(
        name = name,
        srcs = [":%s" % gen_rule_name],
        visibility = visibility,
    )
