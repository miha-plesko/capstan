# Capstan

Capstan is a tool for packing, shipping, and running applications in VMs - just
like Docker but on top of a hypervisor!

## Installation

To build an install Capstan you need Go installed on your machine.  You can
then run:

```
$ go get github.com/codegangsta/cli
$ go get github.com/vaughan0/go-ini
```

```
$ ./install
```

to install Capstan to ``$GOPATH/bin`` of your machine.

## Usage

First, you need to push a VM image to your local Capstan repository:

```
$ capstan push <image>
```

You can then launch the image in a VM with:

```
$ capstan run <image>
```

To print a list of images in your repository, do:

```
$ capstan images
```
