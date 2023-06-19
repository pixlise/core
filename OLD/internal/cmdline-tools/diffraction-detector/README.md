# Diffraction Peak Detection

Detects diffraction peaks in spectral data

## Run from command line

Build command line app

```bash
$ go build ./internal/cmdline/main.go
```

Run Options:

-path : path to load dataset file
-save : if scanning dataset file, can save resulting diffraction protobuf file to specified location

```bash
$ ./main -path <PATH_TO_DATASET_PB_FILE> -save <PATH_TO_SAVE_DIFFRACTION_PEAK_PB_FILE>
```

-query : if scanning dataset file, can query peaks found at specific loaction or over all locations

```bash
$ ./main -path <PATH_TO_DATASET_PB_FILE> -query <LOCATION_NUMBER_TO_PRINT_PEAKS_OR_"ALL">
```

-load : if you already have a protobuf diffraction file can print out the peaks from that file

```bash
$ ./main -load <PATH_TO_LOAD_DIFFRACTION_PEAK_PB_FILE>
```
