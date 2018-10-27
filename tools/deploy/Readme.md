# Install

To install the hyperdrive, you need to have the go compiler installed on
your local machine.

The hyperdrive installs 2 cloudformation stacks for every region on
which it is installed.

1. `Hyperdrive-Artefacts`: an S3 bucket to hold artefacts of the
   hyperdrive: compiled aws lambdas, cloudformation templates, etc..
2. `Hyperdrive-Core`: the collection of lambda functions: cloudformation
   resources, services and cloudformation templates.

To install and update, run the `./basic.sh` script.

