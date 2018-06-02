okgo
====
okgo is a gödel plugin that coordinates and runs Go checks. Individual checks are written as assets.

Plugin Tasks
------------
okgo provides the following tasks:

* `check [checks]`: runs the specified checks (which must be loaded as assets). If no checks are specified, runs all
  checks.
* `run-check [check] [flags] [args]`: runs the specified check "directly" using the specified flags and args. Most check
  assets wrap an underlying check executable and the arguments that are provided to that underlying executable are
  determined based on the plugin configuration. "run-check" allows the underlying check to be called directly. For
  example, `errcheck-asset` wraps the [errcheck](https://github.com/kisielk/errcheck) check. The task `check errcheck`
  invokes `errcheck` using the configuration specified in `check.yml` and the project packages as determined by gödel
  and its configuration. However, one may want to run the underlying `errcheck` check directly -- for example, to run it
  with a specific flag or on a specific input. The `run-check` task allows this. For example, the task
  `run-check errcheck -- -verbose .` runs the errcheck check with the arguments "-verbose ." (the `--` after `errcheck`
  is necessary to signal that all of the arguments that follow should be interpreted literally rather than as flags).

Assets
------
okgo assets are executables that run specific checks. Assets must provide the following commands:

* `type`: prints the name of the check as a JSON string (for example, `"errcheck"`). The value of `type` must be unique
  among all loaded assets.
* `priority`: prints the priority of the check as a JSON integer (for example, `0`). This value is used to determine the
  order in which checks are run. Checks with lower priority values are run first. If multiple checks have the same
  priority, they are run in alphabetic order of `type`.
* `verify-config --config-yml [configuration YAML]`: exits with a non-0 exit code if the provided configuration YAML is
  not valid for the check.
* `check [--project-dir [project directory]] --config-yml [configuration YAML] [packages]`: runs the check on the
  specified packages using the provided configuration. Packages are specified relative to the working directory. Writes
  the JSON representation of `github.com/palantir/okgo/okgo.Issue` to `stdout` for each issue encountered, one per line.
  These issues should be the only output written to `stdout`.
* `run-check-cmd [flags] [args]`: runs the underlying check directly using the provided flags and arguments.

Writing an asset
----------------
okgo provides helper APIs to facilitate writing new assets. More detailed instructions for writing assets are
forthcoming. In the meantime, the most effective way to write an asset is to examine the implementation of an existing
asset.
