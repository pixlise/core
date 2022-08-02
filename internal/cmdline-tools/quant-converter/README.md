# Quantification Converter for PIXLISE

## What this is
Go program that converts quantifications to the PIXLISE protobuf-based binary format.

## Folder structure
`/converter` contains code that can be reused externally to convert CSVs to quants
`/internal/quant-converter` contains the code to build the executable cmd, this just calls on code in `/converter`.

## Setup
Make sure you have the `data-formats` git submodule set up

## Building locally
To build, run: `local-build.sh`
This should generate protobuf serialization code in `experiment` and `quantification`, run tests, and build an executable `quant-converter`

**NOTE:** Since the generated code is now checked in (so it can be included as a go module in other projects, such as the pixlise go API), the
above will regenerate it, but in a Gitlab CI build if the generated files differ, the build script will fail, as it's assumed you have locally
done development, tested it, and checked in the generated protobuf serialisation code.

Specifically we're talking about `./experiment/experiment.pb.go` and `./quantification/quantification.pb.go`.

## File Format
Output file format is binary, format defined using Protobuf (https://developers.google.com/protocol-buffers).

## More information
There are many parameters that can help convert a dataset if there are columns missing. This was done so the initial test datasets work, however
in production we'd expect to have a regular set of columns output from piquant and won't need tweaks.

To see the flags in use, look at the `test-data` repo: `convert-quants.sh` converts all the test quant maps.
