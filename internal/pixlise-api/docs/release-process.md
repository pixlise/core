---
description: Releasing the Core and UI components for use in deployments
---

# Release Process

To update the environments running the Pixlise code base we need to release either the core repository, the ui repository or both.

The process is the same for both environments with slightly different outputs depending on the repo released.&#x20;

* Merge any pull requests into the development branch that need merging
* Let the development build complete and deploy this to the dev environment
* Ensure the integration tests run without error
* PR and merge development branch into main branch
* Deploy the build to staging
* Ensure the integration tests run without error
* Tag the main branch using an **annotated tag** \* `git tag -a <tag> -m "commit message"`
* Push tag to repository `git push --tags`
* You can then create a release in the github UI that uses that tag as the release tag. Check generate release notes if you want automated release notes
* Check Github Actions and ensure the CI jobs run cleanly
* Merge the main branch back into development and then on to active feature branches to ensure they pick up the latest build number

**Using an annotated tag is mandatory, otherwise the semver version picker cannot figure out the next version.**
