## Development Guide
This document is intended to be the canonical source of truth for things like supported toolchain versions for building Stash.
If you find a requirement that this doc does not capture, please submit an issue on github.

This document is intended to be relative to the branch in which it is found. It is guaranteed that requirements will change over time
for the development branch, but release branches of Stash should not change.

### Build Stash
Some of the Stash development helper scripts rely on a fairly up-to-date GNU tools environment, so most recent Linux distros should
work just fine out-of-the-box.

#### Setup GO
Stash is written in Google's GO programming language. Currently, Stash is developed and tested on **go 1.8.3**. If you haven't set up a GO
development environment, please follow [these instructions](https://golang.org/doc/code.html) to install GO.

#### Download Source

```sh
$ go get github.com/appscode/stash
$ cd $(go env GOPATH)/src/github.com/appscode/stash
```

#### Install Dev tools
To install various dev tools for Stash, run the following command:
```sh
$ ./hack/builddeps.sh
```

#### Build Binary
```
$ ./hack/make.py
$ stash version
```

#### Dependency management
Stash uses [Glide](https://github.com/Masterminds/glide) to manage dependencies. Dependencies are already checked in the `vendor` folder.
If you want to update/add dependencies, run:
```sh
$ glide slow
```

#### Build Docker images
To build and push your custom Docker image, follow the steps below. To release a new version of Stash, please follow the [release guide](/docs/developer-guide/release.md).

```sh
# Build Docker image
$ ./hack/docker/stash/setup.sh; ./hack/docker/stash/setup.sh push

# Add docker tag for your repository
$ docker tag appscode/stash:<tag> <image>:<tag>

# Push Image
$ docker push <image>:<tag>
```

#### Generate CLI Reference Docs
```sh
$ ./hack/gendocs/make.sh 
```

### Testing Stash
#### Unit tests
```sh
$ ./hack/make.py test unit
```

#### Run e2e tests
Stash uses [Ginkgo](http://onsi.github.io/ginkgo/) to run e2e tests.
```sh
$ ./hack/make.py test e2e
```
