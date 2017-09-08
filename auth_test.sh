#!/bin/bash

set -v

# TestUserCanAccessAuthedSigner
$GOPATH/bin/autograph-client -u bob -p 9vh6bhlc10y63ow2k4zke7k0c3l9hpr8mo96p92jmbfqngs9e7d -t http://localhost:8000/sign/data -D -r '[{"input": "Y2FyaWJvdW1hdXJpY2UK", "keyid": "appkey2"}]'

# TestUserCannotAccessUnauthedSigner
$GOPATH/bin/autograph-client -u bob -p 9vh6bhlc10y63ow2k4zke7k0c3l9hpr8mo96p92jmbfqngs9e7d -t http://localhost:8000/sign/data -D -r '[{"input": "Y2FyaWJvdW1hdXJpY2UK", "keyid": "appkey1"}]'

# TestUserCannotAccessNonexistentSigner
$GOPATH/bin/autograph-client -u bob -p 9vh6bhlc10y63ow2k4zke7k0c3l9hpr8mo96p92jmbfqngs9e7d -t http://localhost:8000/sign/data -D -r '[{"input": "Y2FyaWJvdW1hdXJpY2UK", "keyid": "notanappkey"}]'

# TestUnauthedUserCannotAccessSigner
$GOPATH/bin/autograph-client -u blob -p 9vh6bhlc10y63ow2k4zke7k0c3l9hpr8mo96p92jmbfqngs9e7d -t http://localhost:8000/sign/data -D -r '[{"input": "Y2FyaWJvdW1hdXJpY2UK", "keyid": "appkey1"}]'

# TestUnauthedUserCannotAccessNonexistentSigner
$GOPATH/bin/autograph-client -u blob -p 9vh6bhlc10y63ow2k4zke7k0c3l9hpr8mo96p92jmbfqngs9e7d -t http://localhost:8000/sign/data -D -r '[{"input": "Y2FyaWJvdW1hdXJpY2UK", "keyid": "notanappkey"}]'
