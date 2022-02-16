# Poetry Cloud Native Buildpack
The Paketo Poetry Buildpack is a Cloud Native Buildpack that installs [Poetry](https://python-poetry.org/) into a
layer and places it on the `PATH`.

The buildpack is published for consumption at `gcr.io/paketo-buildpacks/poetry` and
`paketobuildpacks/poetry`.

## Behavior
This buildpack always participates.

The buildpack will do the following:
* At build time:
  - Contributes the `poetry` binary to a layer
  - Prepends the `poetry` layer to the `PYTHONPATH` environment variable
  - Adds the newly installed `poetry` location to the `PATH` environment variable
* At run time:
  - Does nothing

## Configuration
| Environment Variable | Description
| -------------------- | -----------
| `$BP_POETRY_VERSION` | Configure the version of Poetry to install. Buildpack releases (and the Poetry versions for each release) can be found [here](https://github.com/paketo-buildpacks/poetry/releases).

## Integration

The Poetry CNB provides Poetry as a dependency. Downstream buildpacks can require the Poetry
dependency by generating a [Build Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the Poetry dependency is "poetry". This value is considered
  # part of the public API for the buildpack and will not change without a plan
  # for deprecation.
  name = "poetry"

  # The version of the Poetry dependency is not required. In the case it
  # is not specified, the buildpack will select the latest supported version in
  # the buildpack.toml.
  # If you wish to request a specific version, the buildpack supports
  # specifying a semver constraint in the form of "21.*", "21.0.*", or even
  # "21.0.1".
  version = "21.0.1"

  # The Poetry buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the build flag to true will ensure that the Poetry dependency is
    # available on the $PATH, and that $PYTHONPATH contains the path to `poetry` for
    # subsequent buildpacks during their build phase. If you are writing a
    # buildpack that needs to run Poetry during its build process, this flag should
    # be set to true.
    build = true

    # Setting the launch flag to true will ensure that the Poetry
    # dependency is available on the $PATH, and that $PYTHONPATH contains the
    # path to `poetry` for the running application. If you are writing an
    # application that needs to run Poetry at runtime, this flag should be set to
    # true.
    launch = true
```

## Usage

To package this buildpack for consumption:
```
$ ./scripts/package.sh --version x.x.x
```
This will create a `buildpackage.cnb` file under the build directory which you
can use to build your app as follows: `pack build <app-name> -p <path-to-app> -b
build/buildpackage.cnb -b <other-buildpacks..>`.

To run the unit and integration tests for this buildpack:
```
$ ./scripts/unit.sh && ./scripts/integration.sh
```
