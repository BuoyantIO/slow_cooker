# Releasing slow_cooker

Once all of the branches for the release have been merged and the CHANGELOG.md
file in master has been updated to describe the new release, use these
instructions to publish the release to Github and Dockerhub.

## Github

Start by running `./release.sh` to generate the binaries that will be attached
to the release.

Next [create a new release](https://github.com/BuoyantIO/slow_cooker/releases/new)
on Github.

* For the version, enter the numeric release version.
* For the title, also enter the numeric release version, or something snappier.
* For the description, copy over the entire CHANGELOG.md entry for this release.
* For the binaries, attach all three binaries from the `./release.sh` script.
* Then click "Publish release"

## Dockerhub

Next build and push the docker images to Dockerhub.

First build and push the versioned image (replacing `<release-version>` with the
actual numeric release version that you are releasing):

```bash
$ docker build -t buoyantio/slow_cooker:<release-version> .
$ docker push buoyantio/slow_cooker:<release-version>
```

Then update the image that's tagged as `latest` to match the new release:

```bash
$ docker tag buoyantio/slow_cooker:<release-version> buoyantio/slow_cooker:latest
$ docker push buoyantio/slow_cooker:latest
```

Check the Dockerhub [tags page](https://hub.docker.com/r/buoyantio/slow_cooker/tags/)
to make sure that the new release tag shows up.
