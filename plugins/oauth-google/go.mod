module github.com/thecodearcher/limen/plugins/oauth-google

go 1.24.0

require (
	github.com/thecodearcher/limen/plugins/oauth v0.0.0
	golang.org/x/oauth2 v0.35.0
)

require (
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/thecodearcher/limen v0.0.0 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

replace github.com/thecodearcher/limen => ../..

replace github.com/thecodearcher/limen/plugins/oauth => ../oauth
