module github.com/thecodearcher/limen/examples/basic

go 1.25.0

require (
	github.com/gin-gonic/gin v1.11.0
	github.com/joho/godotenv v1.5.1
	github.com/thecodearcher/limen v0.0.0
	github.com/thecodearcher/limen/adapters/gorm v0.0.0
	github.com/thecodearcher/limen/adapters/sql v0.0.0
	github.com/thecodearcher/limen/plugins/credential-password v0.0.0
	github.com/thecodearcher/limen/plugins/oauth v0.0.0
	github.com/thecodearcher/limen/plugins/oauth-discord v0.0.0
	github.com/thecodearcher/limen/plugins/oauth-facebook v0.0.0-00010101000000-000000000000
	github.com/thecodearcher/limen/plugins/oauth-generic v0.0.0
	github.com/thecodearcher/limen/plugins/oauth-github v0.0.0-00010101000000-000000000000
	github.com/thecodearcher/limen/plugins/oauth-google v0.0.0-00010101000000-000000000000
	github.com/thecodearcher/limen/plugins/oauth-linkedin v0.0.0
	github.com/thecodearcher/limen/plugins/oauth-microsoft v0.0.0
	github.com/thecodearcher/limen/plugins/oauth-spotify v0.0.0
	github.com/thecodearcher/limen/plugins/oauth-twitch v0.0.0
	github.com/thecodearcher/limen/plugins/oauth-twitter v0.0.0
	github.com/thecodearcher/limen/plugins/two-factor v0.0.0
	gorm.io/driver/postgres v1.6.0
	github.com/google/uuid v1.6.0
)

replace github.com/thecodearcher/limen => ../../

replace github.com/thecodearcher/limen/adapters/gorm => ../../adapters/gorm

replace github.com/thecodearcher/limen/adapters/sql => ../../adapters/sql

replace github.com/thecodearcher/limen/plugins/credential-password => ../../plugins/credential-password

replace github.com/thecodearcher/limen/plugins/two-factor => ../../plugins/two-factor

replace github.com/thecodearcher/limen/plugins/oauth => ../../plugins/oauth

replace github.com/thecodearcher/limen/plugins/oauth-google => ../../plugins/oauth-google

replace github.com/thecodearcher/limen/plugins/oauth-discord => ../../plugins/oauth-discord

replace github.com/thecodearcher/limen/plugins/oauth-generic => ../../plugins/oauth-generic

replace github.com/thecodearcher/limen/plugins/oauth-facebook => ../../plugins/oauth-facebook

replace github.com/thecodearcher/limen/plugins/oauth-github => ../../plugins/oauth-github

replace github.com/thecodearcher/limen/plugins/oauth-microsoft => ../../plugins/oauth-microsoft

replace github.com/thecodearcher/limen/plugins/oauth-spotify => ../../plugins/oauth-spotify

replace github.com/thecodearcher/limen/plugins/oauth-twitch => ../../plugins/oauth-twitch

replace github.com/thecodearcher/limen/plugins/oauth-twitter => ../../plugins/oauth-twitter

replace github.com/thecodearcher/limen/plugins/oauth-linkedin => ../../plugins/oauth-linkedin

require (
	github.com/bytedance/sonic v1.14.0 // indirect
	github.com/bytedance/sonic/loader v0.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.27.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/goccy/go-yaml v1.18.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/quic-go/quic-go v0.54.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.0 // indirect
	go.uber.org/mock v0.5.0 // indirect
	golang.org/x/arch v0.20.0 // indirect
	golang.org/x/mod v0.28.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/tools v0.37.0 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
)
