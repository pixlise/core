# Pixlise Core Library

Welcome to the technical documentation covering the Pixlise Core library.

This library provides a number of services for other Pixlise components. 
It is designed to be agnostic, and portable, containing no component specific business logic.

The current list of packages:

* api - generic API utilities for running any API services
* auth0login - JWT services for Auth0
* awsutil - AWS specific utility functions
* cloudwatch - Cloudwatch log functions
* dataset - Dataset processing functions
* detector - Configuration utilities for the detectors
* downloader - Downloading optimisations
* export - Export functionality and services
* expression - Expression language parsing helpers
* fileaccess - File access, both S3 and local storage
* kubernetes - Kubernetes wrapping functionality
* logger - Logging helpers
* mongo - Mongo connectivity 
* notifications - Notification services
* piquant - Piquant hooks
* pixlUser - User services
* quantModel - Quantification Model parsing
* roiModel - Region of Interest parser
* tagModel - Tag model parsing
* timestamper - Timestamp utilities
* utils - Other small utilities
