[![Build Status](https://travis-ci.org/databus23/requestutil.png?branch=master)](https://travis-ci.org/databus23/requestutil)

requestutil
===========
Requestutil is a small go library for extracting host, port and scheme information from a `http.Request`.
It is usefully for generating absolute urls and considers headers set by loadbalancers (e.g. `X-Forwarded-Host` ...)

It is inspired by [ActionDispatch::Http::URL](http://api.rubyonrails.org/classes/ActionDispatch/Http/URL.html)