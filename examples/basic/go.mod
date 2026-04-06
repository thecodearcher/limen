module github.com/thecodearcher/limen/examples/basic

go 1.25.0

require (
	github.com/gin-contrib/cors v1.7.6
	github.com/gin-gonic/gin v1.11.0
	github.com/go-sql-driver/mysql v1.8.1
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/thecodearcher/limen v0.0.0
	github.com/thecodearcher/limen/adapters/gorm v0.0.0
	github.com/thecodearcher/limen/adapters/sql v0.0.0
	github.com/thecodearcher/limen/plugins/credential-password v0.0.0
	github.com/thecodearcher/limen/plugins/oauth v0.0.0
	github.com/thecodearcher/limen/plugins/oauth-apple v0.0.0
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
	gorm.io/gorm v1.30.1
)

replace github.com/thecodearcher/limen => ../../

replace github.com/thecodearcher/limen/adapters/gorm => ../../adapters/gorm

replace github.com/thecodearcher/limen/adapters/sql => ../../adapters/sql

replace github.com/thecodearcher/limen/plugins/credential-password => ../../plugins/credential-password

replace github.com/thecodearcher/limen/plugins/two-factor => ../../plugins/two-factor

replace github.com/thecodearcher/limen/plugins/oauth => ../../plugins/oauth

replace github.com/thecodearcher/limen/plugins/oauth-apple => ../../plugins/oauth-apple

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
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/bytedance/sonic v1.15.0 // indirect
	github.com/bytedance/sonic/loader v0.5.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/gabriel-vasile/mimetype v1.4.12 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.1 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmoiron/sqlx v1.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pquerna/otp v1.4.0 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.59.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.1 // indirect
	go.uber.org/mock v0.6.0 // indirect
	golang.org/x/arch v0.23.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	modernc.org/libc v1.70.0 // indirect
)
