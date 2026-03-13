module github.com/thecodearcher/aegis/plugins/oauth-twitter

go 1.24.0

require (
	github.com/thecodearcher/aegis/plugins/oauth v0.0.0
	golang.org/x/oauth2 v0.35.0
)

require (
	github.com/golang/mock v1.6.0 // indirect
	github.com/thecodearcher/aegis v0.0.0 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

replace github.com/thecodearcher/aegis => ../..

replace github.com/thecodearcher/aegis/plugins/oauth => ../oauth
