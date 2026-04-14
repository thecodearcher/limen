module github.com/thecodearcher/limen/plugins/oauth-twitter

go 1.25.0

require (
	github.com/thecodearcher/limen/plugins/oauth v0.0.5
	golang.org/x/oauth2 v0.35.0
)

require (
	github.com/thecodearcher/limen v0.0.6 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

replace github.com/thecodearcher/limen => ../..

replace github.com/thecodearcher/limen/plugins/oauth => ../oauth
