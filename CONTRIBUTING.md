### Setup your mc GitHub Repository

This repository is maintained at [delta592/mc](https://github.com/delta592/mc). It was forked from the archived upstream [minio/mc](https://github.com/minio/mc) project.

```
git clone https://github.com/delta592/mc.git
cd mc
make
./mc --help
```

### Developer Guidelines

``mc`` welcomes your contribution. To make the process as seamless as possible, we ask for the following:

* Go ahead and fork the project and make your changes. We encourage pull requests to discuss code changes.
    - Fork it
    - Create your feature branch (git checkout -b my-new-feature)
    - Commit your changes (git commit -am 'Add some feature')
    - Push to the branch (git push origin my-new-feature)
    - Create new Pull Request

* If you have additional dependencies for ``mc``, ``mc`` manages its dependencies using `go mod`
    - Run `go get foo/bar`
    - Edit your code to import foo/bar
    - Run `go mod tidy` from top-level folder

* When you're ready to create a pull request, be sure to:
    - Have test cases for the new code. If you have questions about how to do it, please ask in your pull request.
    - Run `make fmt`
    - Run `make verifiers`
    - Run `make test-short`

* Sign your work

We use the Developer Certificate of Origin (DCO) in lieu of a CLA for contributions to this project. Please refer to [docs/DCO](https://github.com/delta592/mc/blob/master/docs/DCO).
