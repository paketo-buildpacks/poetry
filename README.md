# Poetry Cloud Native Buildpack
## `gcr.io/paketo-buildpacks/poetry`

The Paketo Buildpack for Poetry is a Cloud Native Buildpack that installs [Poetry](https://python-poetry.org/) into a
layer and places it on the `PATH`.

The buildpack is published for consumption at `gcr.io/paketo-buildpacks/poetry` and
`paketobuildpacks/poetry`.

## Detection

* Detects when `pyproject.toml` exists.
* Provides `poetry`.
* Always requires `cpython` and `pip`.
* Optionally requires `poetry` when `BP_POETRY_VERSION` is set.

## Build
* Contributes the `poetry` binary to a layer
* Prepends the `poetry` layer to the `PYTHONPATH` environment variable
* Adds the newly installed `poetry` location to the `PATH` environment variable

## Configuration
| Environment Variable | Description                                                                                                                                                                          |
|----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `$BP_POETRY_VERSION` | Configure the version of Poetry to install. Buildpack releases (and the Poetry versions for each release) can be found [here](https://github.com/paketo-buildpacks/poetry/releases). |

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

  # The Poetry buildpack supports some non-required metadata options.
  [requires.metadata]

    # Optional.
    # When not specified, the buildpack will select the latest supported version from buildpack.toml
    # This buildpack only supports exact version numbers.
    version = "21.0.1"

    # Set to true to ensure that `poetry` is avilable on both `$PATH` and `$PYTHONPATH` for subsequent buildpacks.
    build = true

    # Set to true to ensure that `poetry` is avilable on both `$PATH` and `$PYTHONPATH` for the launch container.
    launch = true
```

## Usage

To package this buildpack for consumption:
```
$ ./scripts/package.sh --version x.x.x
```
This will create a `buildpackage.cnb` file under the build directory which you
can use to build your app as follows:

```shell
pack build <app-name> \
  --path <path-to-app> \
  --buildpack build/buildpackage.cnb \
  --buildpack <other-buildpacks..>
```

To run the unit and integration tests for this buildpack:
```shell
$ ./scripts/unit.sh && ./scripts/integration.sh
```

## Known issues and limitations

* This buildpack does not work in an offline/air-gapped environment; it
  requires internet access to install `poetry`. The impact of this limitation
  is mitigated by the fact that `poetry` itself does not support vendoring of
  dependencies, and so cannot function in an offline/air-gapped environment.
