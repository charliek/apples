# apples

`apples` is a program to help setup and update a development environment when
working with a microservice architecture. Using toml files you are able to
configure scripts on how to setup your services, and when complete a Procfile
is written out which can be used by [foreman](http://ddollar.github.io/foreman/).

The idea is that the scripts run by `apples` will keep the services configured updated, and then you are
able to quickly change the files beeing run as you work on others. This is not meant to take the place
of a proper vagrant/configuration management setup, but might be able to help provide a simple fast
environment for local development.

## Installation

For a mac install you can download a [version from the github release page](https://github.com/charliek/apples/releases).

For a source install you should be able to just do:

```
$ go get github.com/charliek/apples
```

This program has only been tested on a mac with go 1.3.

## Usage & Example

An example configuration can be found [within the project](https://github.com/charliek/apples/blob/master/applications.toml).
To use `apples` create a config file named `applications.toml` in a format similar to the example, and then run
the `apples` command in the same directory. This will run the scripts you have configured and write out a Procfile in
the same directory. Each time the `apples` program is run the application scripts will be executed and the Procfile will
be re-written.  Once the proc file has been written you should use foreman to run the programs.
