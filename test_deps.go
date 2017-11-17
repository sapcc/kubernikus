//this file is only here so that `glide list` picks
//up our test dependencies. Otherwise glide-vc will delete
//out test dependencies and the build will go kaputt...
package main

import "github.com/stretchr/testify/assert"

func main() {
	assert.CallerInfo()

}
