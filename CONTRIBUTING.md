# Contributing

Getting started is _super easy_: just run these commands under Linux or MacOS:

```bash
git clone git@github.com:fornellas/resonance.git
cd resonance/
./make.sh ci # runs linters, tests and build
```

That's it! The local build happens inside a [Docker](https://www.docker.com/), so it is easily reproducible on any machine.

If you're running an old Arm Mac (eg: M1), Docker is _very_ slow, you should consider a no container build (see below).

## Shell

You can start a development shell, which gives you access to the whole build environemnt, including all build tools with the correct versions. Eg:

```shell
./make.sh shell
make ci
go get -u
```

## No container

You may also run the _unofficial_[^1] build, without a container. This requires Installing [GNU Make](https://www.gnu.org/software/make/):

- Ubuntu: `apt install make`.
- MacOS: `brew install make`[^2]

Then:

```shell
make ci # on MacOS, gmake ci
```

Or start a bash[^3] development shell:

```shell
make shell
```

[^1]: unnofficial in the sense that it is _not_ the same as it happens on CI, as unforeseen environment differences may impact the signal. Additionally, a lot of code / tests is Linux only (eg: `*_linux.go`), so none of this signal will be available on MacOS.

[^2]: note that brew install the command as `gmake`. Apple's ancient `make` shipped with MacOS will NOT work.

[^3]: bash must be installed separately, eg, under MacOS, `brew install bash`.

## Using gopls, the Go Language Server

Your editor may already support using [gopls](https://github.com/golang/tools/tree/master/gopls), and you should follow its documentation on how to set it up. This may require having the correct go (and gopls) versions installed and available for your editor. This can be annoying and error prone.

In this scenario, you should leverage the "no container" option:

```shell
make shell
make ci # installs all development tools
```

And then start you code editor from the development shell (eg: for Sublime, do `subl .`). This enables the code editor to have access to all the _exact_ versions of tools required.

## Faster builds

The default build with `ci` reproduces the official build, but this may be too slow during development. You can use one of the `*-dev` targets to do a "dev build": bulid is a lot faster, at the expense of minimal signal loss:

```shell
./make.sh ci-dev # or "make ci-dev"
```

## Automatic builds on Linux

The build system is integrated with [rrb](https://github.com/fornellas/rrb), which enables the build to run automatically as you edit the files.

First, start rrb:

```shell
./make.sh rrb # or "make rrb"
```

then just edit the files with your preferred editor. As soon as you save any file, the build will be automatically run, interrupting any ongoing build, so you always get a fresh signal.

There's also a `rrb-dev` target, which yields faster builds during development.