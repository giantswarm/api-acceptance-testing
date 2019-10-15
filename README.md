# Node Pools Acceptance Tests

## Usage

```
opsctl gsctl login -i gauss

go run main.go runtests \
  --endpoint (gsctl info|grep "API endpoint:"|awk '{print $3}') \
  --token (gsctl info -v|grep "Auth token:"|awk '{print $3}') \
  --scheme Bearer
```

The above command will ensure that you have `gauss` as your selected gsctl endpoint and will give you a valid SSO token to use
for executing the tests.

