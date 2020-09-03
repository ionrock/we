#!/bin/sh

# We run this from make in the root of the project.
./we -e examples/hello_world.yml echo 'Hello $NAME!'
