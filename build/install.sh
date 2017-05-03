#!/bin/bash

# N.B.:
# You should have 'golang' already installed to proceed!
echo "Importing necessary packages..."
go get "github.com/llgcode/draw2d/draw2dimg"
echo "Compiling the source..."
cd "${PWD}/../src"
go build
mv src ../build/saga-mikron
cp -R tpl ../build/tpl
cp -R ver ../build/ver
cp -R dat ../build/dat
rm ../build/dat/.gitignore
echo "Compilation has finished. Have a good day :)"
