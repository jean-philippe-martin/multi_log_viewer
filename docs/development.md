# development

## Starting the project

First activate the .venv, then run `mlv`. 

There is a `run-log-generators.sh` script. It will start writing test logs in the
existing logs/ folder.

## Running tests

run `./run-tests.sh` to run all tests. You can also run just a subset of the tests with various arguments. Here's the help text.

```
 % ./run-tests.sh --help
Usage: ./run-tests.sh [OPTIONS] [PYTEST_ARGS...]

Options:
  --only-fast           Unit tests only (skip integration and performance)
  --only-integration    Textual TUI integration tests only
  --only-perf           Performance / timing tests only (prints timing vs log)
  -h, --help            Show this help

Any other arguments are passed to pytest (e.g. -v, -q, -k, tests/test_foo.py).
Use -- to pass flags to pytest that look like run-tests options:
```