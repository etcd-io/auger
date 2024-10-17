# Release Process

The Auger Project is released on an as-needed basis. The process is as follows:

1. Create a new git tag with `git tag -s $VERSION`
1. Push the tag with `git push $VERSION`
1. Once pushed, the [github workflow](.github/workflows/release.yaml) will automatically create a new release with the tag and changelog.