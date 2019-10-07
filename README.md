<!--

    TODO:

    - Add the project to the CircleCI:
      https://circleci.com/setup-project/gh/giantswarm/REPOSITORY_NAME

    - Import RELEASE_TOKEN variable from template repository for the builds:
      https://circleci.com/gh/giantswarm/REPOSITORY_NAME/edit#env-vars

    - Change the badge (with style=shield):
      https://circleci.com/gh/giantswarm/REPOSITORY_NAME/edit#badges
      If this is a private repository token with scope `status` will be needed.

    - Change the top level header from `# template` to `# REPOSITORY_NAME` and
      add appropriate description.

    - If the repository is public consider adding godoc badge. This should be
      the first badge separated with a single space.
      [![GoDoc](https://godoc.org/github.com/giantswarm/REPOSITORY_NAME?status.svg)](http://godoc.org/github.com/giantswarm/REPOSITORY_NAME)

-->

# Node Pools Acceptance Tests

Usage:

```
# cd to gopath
git clone https://github.com/giantswarm/node-pools-acceptance-test.git
cd node-pools-acceptance-test
go build

./node-pools-acceptance-test run \
  --endpoint (gsctl info|grep "API endpoint:"|awk '{print $3}') \
  --scheme Bearer --token (gsctl info -v|grep "Auth token:"|awk '{print $3}')
```

Make sure that you are logged in to a `gsctl` endpoint. Check using `gsctl info` when in doubt.
